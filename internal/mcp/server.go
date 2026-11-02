package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/barbarkaragul-oss/agentwall/internal/dlp"
	"github.com/barbarkaragul-oss/agentwall/internal/policy"
)

const latestProtocolVersion = "2025-11-25"
const maxMessageBytes = 10 * 1024 * 1024

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type stdioServer struct {
	initialized bool
	ready       bool
	scanner     *dlp.Scanner
	policy      *policy.Policy
}

// ServeStdio exposes the local inspect_payload tool over newline-delimited
// JSON-RPC. Stdout contains protocol messages only; the caller reports I/O errors.
func ServeStdio() error {
	return serveStdio(os.Stdin, os.Stdout)
}

func serveStdio(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), maxMessageBytes)
	encoder := json.NewEncoder(out)
	server := &stdioServer{scanner: dlp.NewScanner(), policy: policy.DefaultPolicy()}
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		response := server.handle(line)
		if response != nil {
			if err := encoder.Encode(response); err != nil {
				return fmt.Errorf("write MCP response: %w", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read MCP request (maximum message size %d bytes): %w", maxMessageBytes, err)
	}
	return nil
}

func errorResponse(id json.RawMessage, code int, message string) *rpcResponse {
	return &rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

func validID(id json.RawMessage) bool {
	if len(id) == 0 {
		return true // Notification, not a request.
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(id))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return false
	}
	switch value.(type) {
	case string, json.Number:
		return true
	default:
		return false // MCP request IDs cannot be null, arrays, objects or booleans.
	}
}

func isObject(data json.RawMessage) bool {
	data = bytes.TrimSpace(data)
	return len(data) > 0 && data[0] == '{'
}

func (s *stdioServer) handle(line []byte) *rpcResponse {
	if !json.Valid(line) {
		return errorResponse(nil, -32700, "Parse error")
	}
	var req rpcRequest
	if !isObject(line) || json.Unmarshal(line, &req) != nil ||
		req.JSONRPC != "2.0" || req.Method == "" || !validID(req.ID) {
		return errorResponse(nil, -32600, "Invalid Request")
	}
	if len(req.ID) == 0 {
		// JSON-RPC notifications never receive responses, including unknown ones.
		if req.Method == "notifications/initialized" && s.initialized &&
			(len(req.Params) == 0 || isObject(req.Params)) {
			s.ready = true
		}
		return nil
	}
	if len(req.Params) != 0 && !isObject(req.Params) {
		return errorResponse(req.ID, -32602, "Params must be an object")
	}
	respond := func(result any) *rpcResponse {
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	}

	switch req.Method {
	case "initialize":
		if s.initialized {
			return errorResponse(req.ID, -32600, "Already initialized")
		}
		var params struct {
			ProtocolVersion string          `json:"protocolVersion"`
			Capabilities    json.RawMessage `json:"capabilities"`
			ClientInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
		}
		if json.Unmarshal(req.Params, &params) != nil || params.ProtocolVersion == "" ||
			!isObject(params.Capabilities) || params.ClientInfo.Name == "" || params.ClientInfo.Version == "" {
			return errorResponse(req.ID, -32602, "initialize requires protocolVersion, capabilities and clientInfo (name, version)")
		}
		version := latestProtocolVersion
		switch params.ProtocolVersion {
		case "2024-11-05", "2025-03-26", "2025-06-18", latestProtocolVersion:
			version = params.ProtocolVersion
		}
		s.initialized = true
		return respond(map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]string{"name": "agentwall", "version": "1.1.0"},
		})
	case "ping":
		return respond(map[string]any{})
	case "tools/list", "tools/call":
		if !s.ready {
			return errorResponse(req.ID, -32002, "Send initialize and notifications/initialized before using tools")
		}
	default:
		return errorResponse(req.ID, -32601, "Method not found")
	}

	if req.Method == "tools/list" {
		if len(req.Params) != 0 {
			var params struct {
				Cursor *string `json:"cursor"`
			}
			if json.Unmarshal(req.Params, &params) != nil || params.Cursor != nil {
				return errorResponse(req.ID, -32602, "This server does not issue pagination cursors")
			}
		}
		return respond(map[string]any{"tools": []any{map[string]any{
			"name":        "inspect_payload",
			"description": "Inspect supplied text locally for supported secret patterns and default-policy matches. Returns JSON text with sanitized (matched secrets redacted), secrets (match count), and violations (policy match count). Sensitive paths and prompt-injection matches are counted, not removed. Does not read files, execute commands, or contact external services.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"payload": map[string]string{"type": "string", "description": "Text or serialized JSON-RPC payload to inspect; an empty string is allowed"},
				},
				"required": []string{"payload"},
			},
		}}})
	}

	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if json.Unmarshal(req.Params, &params) != nil || params.Name == "" ||
		(len(params.Arguments) != 0 && !isObject(params.Arguments)) {
		return errorResponse(req.ID, -32602, "tools/call requires a name and object arguments")
	}
	if params.Name != "inspect_payload" {
		return errorResponse(req.ID, -32602, "Unknown tool")
	}
	var args struct {
		Payload *string `json:"payload"`
	}
	if json.Unmarshal(params.Arguments, &args) != nil || args.Payload == nil {
		return respond(map[string]any{
			"isError": true,
			"content": []any{map[string]string{"type": "text", "text": "inspect_payload requires a string argument named payload"}},
		})
	}
	sanitized, findings := s.scanner.InspectAndRedact(*args.Payload, true)
	violations := s.policy.EvaluatePayload(*args.Payload)
	data, _ := json.Marshal(map[string]any{
		"sanitized": sanitized, "secrets": len(findings), "violations": len(violations),
	})
	return respond(map[string]any{
		"content": []any{map[string]string{"type": "text", "text": string(data)}},
	})
}
