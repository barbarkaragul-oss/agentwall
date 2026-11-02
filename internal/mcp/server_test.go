package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

const initializeMessage = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0"}}}`
const initializedMessage = `{"jsonrpc":"2.0","method":"notifications/initialized"}`

type wireResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

func exchange(t *testing.T, messages ...string) []wireResponse {
	t.Helper()
	var out bytes.Buffer
	if err := serveStdio(strings.NewReader(strings.Join(messages, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	var responses []wireResponse
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var response wireResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("non-JSON output %q: %v", line, err)
		}
		if response.JSONRPC != "2.0" || len(response.ID) == 0 || (response.Error == nil) == (len(response.Result) == 0) {
			t.Fatalf("invalid JSON-RPC response: %s", line)
		}
		responses = append(responses, response)
	}
	return responses
}

func TestStdioLifecycleAndInspection(t *testing.T) {
	responses := exchange(t, initializeMessage, initializedMessage,
		`{"jsonrpc":"2.0","id":"list","method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"inspect_payload","arguments":{"payload":"AWS_KEY=AKIAIOSFODNN7EXAMPLE path=/home/test/.ssh/id_rsa"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`)
	if len(responses) != 4 {
		t.Fatalf("want four replies, got %d", len(responses))
	}
	var init struct {
		ProtocolVersion string                     `json:"protocolVersion"`
		Capabilities    map[string]json.RawMessage `json:"capabilities"`
	}
	if err := json.Unmarshal(responses[0].Result, &init); err != nil || init.ProtocolVersion != "2025-11-25" || init.Capabilities["tools"] == nil {
		t.Fatalf("unexpected initialization: %s", responses[0].Result)
	}
	var list struct {
		Tools []struct {
			Name        string `json:"name"`
			InputSchema struct {
				Type       string   `json:"type"`
				Required   []string `json:"required"`
				Properties map[string]struct {
					Type string `json:"type"`
				} `json:"properties"`
			} `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(responses[1].Result, &list); err != nil || len(list.Tools) != 1 || list.Tools[0].Name != "inspect_payload" {
		t.Fatalf("unexpected tool list: %s", responses[1].Result)
	}
	tool := list.Tools[0]
	if tool.InputSchema.Type != "object" || len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "payload" || tool.InputSchema.Properties["payload"].Type != "string" {
		t.Fatalf("invalid input schema: %+v", tool.InputSchema)
	}
	if string(responses[1].ID) != `"list"` {
		t.Fatalf("string ID not preserved: %s", responses[1].ID)
	}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(responses[2].Result, &result); err != nil || len(result.Content) != 1 || result.Content[0].Type != "text" {
		t.Fatalf("unexpected tool response: %s", responses[2].Result)
	}
	var inspection struct {
		Sanitized  string `json:"sanitized"`
		Secrets    int    `json:"secrets"`
		Violations int    `json:"violations"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &inspection); err != nil {
		t.Fatal(err)
	}
	if inspection.Secrets != 1 || inspection.Violations != 1 || strings.Contains(inspection.Sanitized, "AKIAIOSFODNN7EXAMPLE") || !strings.Contains(inspection.Sanitized, ".ssh/id_rsa") {
		t.Fatalf("unexpected inspection: %+v", inspection)
	}
	if string(responses[3].Result) != "{}" {
		t.Fatalf("unexpected ping: %s", responses[3].Result)
	}
}

func TestProtocolVersionNegotiation(t *testing.T) {
	for _, version := range []string{"2024-11-05", "2025-03-26", "2025-06-18", "2025-11-25", "2099-01-01"} {
		t.Run(version, func(t *testing.T) {
			responses := exchange(t, strings.Replace(initializeMessage, "2025-11-25", version, 1))
			want := version
			if version == "2099-01-01" {
				want = "2025-11-25"
			}
			var result struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			if len(responses) != 1 || json.Unmarshal(responses[0].Result, &result) != nil || result.ProtocolVersion != want {
				t.Fatalf("unexpected negotiation: %+v", responses)
			}
		})
	}
}

func TestJSONRPCErrorsAndRecovery(t *testing.T) {
	cases := []struct {
		name, message string
		code          int
		id            string
	}{
		{"parse", `{`, -32700, "null"},
		{"batch", `[]`, -32600, "null"},
		{"null", `null`, -32600, "null"},
		{"version", `{"jsonrpc":"1.0","id":7,"method":"ping"}`, -32600, "null"},
		{"method absent", `{"jsonrpc":"2.0","id":7}`, -32600, "null"},
		{"boolean id", `{"jsonrpc":"2.0","id":true,"method":"ping"}`, -32600, "null"},
		{"null id", `{"jsonrpc":"2.0","id":null,"method":"ping"}`, -32600, "null"},
		{"unknown method", `{"jsonrpc":"2.0","id":7,"method":"missing"}`, -32601, "7"},
		{"unknown tool", `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"missing","arguments":{"payload":"text"}}}`, -32602, "7"},
		{"missing name", `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"arguments":{}}}`, -32602, "7"},
		{"non-object params", `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":[]}`, -32602, "7"},
		{"non-object arguments", `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"inspect_payload","arguments":[]}}`, -32602, "7"},
		{"unexpected cursor", `{"jsonrpc":"2.0","id":7,"method":"tools/list","params":{"cursor":"not-issued"}}`, -32602, "7"},
		{"reinitialize", initializeMessage, -32600, "1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := exchange(t, initializeMessage, initializedMessage, tc.message, `{"jsonrpc":"2.0","id":99,"method":"ping"}`)
			if len(r) != 3 || r[1].Error == nil || r[1].Error.Code != tc.code || string(r[1].ID) != tc.id || string(r[2].Result) != "{}" {
				t.Fatalf("unexpected error/recovery: %+v", r)
			}
		})
	}
}

func TestInitializationValidation(t *testing.T) {
	for _, params := range []string{`{}`, `{"protocolVersion":"2025-11-25"}`, `{"protocolVersion":"2025-11-25","capabilities":null,"clientInfo":{"name":"x","version":"1"}}`, `{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{}}`} {
		r := exchange(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":`+params+`}`)
		if len(r) != 1 || r[0].Error == nil || r[0].Error.Code != -32602 {
			t.Fatalf("accepted invalid initialization: %s", params)
		}
	}
}

func TestInitializationOrdering(t *testing.T) {
	list := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	for _, messages := range [][]string{{list}, {initializeMessage, list}, {initializedMessage, initializeMessage, list}} {
		r := exchange(t, messages...)
		last := r[len(r)-1]
		if last.Error == nil || last.Error.Code != -32002 {
			t.Fatalf("accepted tools before handshake: %+v", r)
		}
	}
	r := exchange(t, `{"jsonrpc":"2.0","id":0,"method":"ping"}`)
	if len(r) != 1 || string(r[0].ID) != "0" || string(r[0].Result) != "{}" {
		t.Fatalf("pre-initialize ping failed: %+v", r)
	}
}

func TestNotificationsAreSilent(t *testing.T) {
	r := exchange(t, initializeMessage, initializedMessage,
		`{"jsonrpc":"2.0","method":"ping"}`,
		`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"inspect_payload","arguments":{"payload":"text"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":7}}`,
		`{"jsonrpc":"2.0","method":"unknown"}`)
	if len(r) != 1 {
		t.Fatalf("notifications received responses: %+v", r)
	}
}

func TestToolInputValidation(t *testing.T) {
	for _, args := range []string{`{}`, `{"payload":null}`, `{"payload":1}`, `{"payload":true}`, `{"payload":[]}`} {
		r := exchange(t, initializeMessage, initializedMessage, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"inspect_payload","arguments":`+args+`}}`)
		var result struct {
			IsError bool `json:"isError"`
		}
		if len(r) != 2 || r[1].Error != nil || json.Unmarshal(r[1].Result, &result) != nil || !result.IsError {
			t.Fatalf("accepted invalid payload: %s", args)
		}
	}
}

func TestJSONEscapingAndEmptyInput(t *testing.T) {
	for _, payload := range []string{"", "line\nquote\" tab\t backslash\\", "\u0085\u200b İstanbul 🚀"} {
		input, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "escaped", "method": "tools/call", "params": map[string]any{"name": "inspect_payload", "arguments": map[string]any{"payload": payload, "extra": 42}}})
		r := exchange(t, initializeMessage, initializedMessage, string(input))
		var result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if len(r) != 2 || json.Unmarshal(r[1].Result, &result) != nil || len(result.Content) != 1 {
			t.Fatalf("invalid JSON result: %+v", r)
		}
		var inspection struct {
			Sanitized string `json:"sanitized"`
		}
		if json.Unmarshal([]byte(result.Content[0].Text), &inspection) != nil || inspection.Sanitized != payload {
			t.Fatalf("payload changed: %q", payload)
		}
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestStdioIOErrors(t *testing.T) {
	if err := serveStdio(failingReader{}, io.Discard); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("read error lost: %v", err)
	}
	if err := serveStdio(strings.NewReader(initializeMessage+"\n"), failingWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write error lost: %v", err)
	}
	if err := serveStdio(strings.NewReader(strings.Repeat("x", maxMessageBytes+1)), io.Discard); err == nil {
		t.Fatal("oversized input accepted")
	}
}

func Example_serveStdio() {
	input := initializeMessage + "\n" + initializedMessage + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"inspect_payload","arguments":{"payload":"hello"}}}` + "\n"
	var out bytes.Buffer
	_ = serveStdio(strings.NewReader(input), &out)
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var result wireResponse
	_ = json.Unmarshal([]byte(lines[1]), &result)
	fmt.Println(string(result.Result))
	// Output: {"content":[{"text":"{\"sanitized\":\"hello\",\"secrets\":0,\"violations\":0}","type":"text"}]}
}
