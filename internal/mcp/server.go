package mcp

import (
"bufio"
"encoding/json"
"fmt"
"os"

"github.com/barbarkaragul-oss/agentwall/internal/dlp"
"github.com/barbarkaragul-oss/agentwall/internal/policy"
)

type rpcRequest struct {
JSONRPC string          `json:"jsonrpc"`
ID      json.RawMessage `json:"id,omitempty"`
Method  string          `json:"method"`
Params  json.RawMessage `json:"params,omitempty"`
}

// ServeStdio runs AgentWall as a native MCP server exposing DLP and policy inspection tools for Glama introspection.
func ServeStdio() {
scanner := bufio.NewScanner(os.Stdin)
buf := make([]byte, 0, 1024*1024)
scanner.Buffer(buf, 10*1024*1024)

dlpScanner := dlp.NewScanner()
pol := policy.DefaultPolicy()

for scanner.Scan() {
line := scanner.Bytes()
if len(line) == 0 {
continue
}
var req rpcRequest
if err := json.Unmarshal(line, &req); err != nil {
continue
}

switch req.Method {
case "initialize":
resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"agentwall","version":"1.1.0"}}}`, string(req.ID))
fmt.Println(resp)
case "notifications/initialized":
// No response required for notifications
case "ping":
fmt.Printf(`{"jsonrpc":"2.0","id":%s,"result":{}}`+"\n", string(req.ID))
case "tools/list":
resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":{"tools":[{"name":"inspect_payload","description":"Inspect and redact leaked secrets (AWS, GitHub, OpenAI, SSH keys), sensitive file paths, and indirect prompt injections in real time.","inputSchema":{"type":"object","properties":{"payload":{"type":"string","description":"Raw text or JSON-RPC payload to inspect and sanitize"}},"required":["payload"]}}]}}`, string(req.ID))
fmt.Println(resp)
case "tools/call":
var params struct {
Name      string            `json:"name"`
Arguments map[string]string `json:"arguments"`
}
_ = json.Unmarshal(req.Params, &params)
input := params.Arguments["payload"]
sanitized, findings := dlpScanner.InspectAndRedact(input, true)
violations := pol.EvaluatePayload(input)
out, _ := json.Marshal(map[string]interface{}{
"sanitized":  sanitized,
"secrets":    len(findings),
"violations": len(violations),
})
resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":{"content":[{"type":"text","text":%q}]}}`, string(req.ID), string(out))
fmt.Println(resp)
default:
if len(req.ID) > 0 {
fmt.Printf(`{"jsonrpc":"2.0","id":%s,"result":{}}`+"\n", string(req.ID))
}
}
}
}