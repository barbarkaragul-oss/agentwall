# MCP validation — 2026-11-02

This review covers AgentWall's native stdio `serve` / `mcp` mode and its
`inspect_payload` tool. The reviewed starting commit was
`d057ba8bcc744e5e569417704abebf9f977a4de5`.

## Findings and corrections

- Unknown methods previously returned successful empty results; they now return
  JSON-RPC `-32601`.
- Unknown tool names and malformed call envelopes now return `-32602`.
  Missing, null or non-string payloads produce actionable tool errors.
- Malformed JSON now returns `-32700`; invalid envelopes and IDs return
  `-32600`. A bad message does not break subsequent valid requests.
- Responses are serialized with `encoding/json`, including Unicode that Go's
  quoted-string formatting can otherwise encode with non-JSON escapes.
- Notifications are silent. Tool operations require successful initialization
  and `notifications/initialized`. Ping works during initialization.
- Initialization negotiates the four advertised handshake protocol revisions,
  counter-offering `2025-11-25` for unsupported versions.
- Read/write failures and oversized lines are propagated to the CLI, reported
  on stderr, and produce a nonzero exit code.
- Tool documentation accurately distinguishes secret-pattern redaction from
  policy-match counting; policy matches are not removed from the returned text.

## Reproducible verification

The following checks passed in an isolated checkout using Go 1.22 on Linux:

| Check | Result |
| --- | --- |
| `go test -v -race ./...` | Passed, including native MCP regression tests |
| `go vet ./...` | Passed |
| Linux amd64 build | Passed |
| Darwin arm64 cross-build | Passed |
| Windows amd64 cross-build | Passed |
| stdio process check, `agentwall serve` | Passed |
| stdio process check, `agentwall mcp` | Passed |
| Build with the existing root Dockerfile | Passed |
| stdio process check against Docker ENTRYPOINT | Passed with networking disabled |

Run the commands in the README's **Container and protocol checks** section.
The process checker is `scripts/check_mcp_stdio.py`; it uses Python's standard
library and tests an actual subprocess, including handshake, tool schema,
redaction, Unicode, protocol and tool errors, ping and EOF shutdown.
The GitHub CI workflow includes the binary and container checks.

These results establish the tested native stdio behavior. They do not certify
all MCP features or the separate `wrap` proxy. HTTP transport and the
`2026-07-28` stateless protocol are not implemented.

## External Glama status

On 2026-11-02 the public page at
<https://glama.ai/mcp/servers/barbarkaragul-oss/agentwall> displayed:
“We don’t have an MCP server at barbarkaragul-oss/agentwall.”
No published quality score or successful Glama evaluation could be verified.
The Add Server action required a Glama account session.

The repository already has `glama.json` naming `barbarkaragul-oss` and a
working Dockerfile with `ENTRYPOINT ["agentwall", "serve"]`. These files do not
create a Glama listing automatically.

Remaining external steps:

1. Sign in to Glama, submit this public GitHub repository, and verify the owner.
2. Add the root Dockerfile directly to the Glama build configuration.
3. Run the evaluation and inspect the actual build, introspection and score.
4. Add the resulting verified score badge to the listing PR and mark it ready.

The listing work is consolidated in
<https://github.com/punkpeye/awesome-mcp-servers/pull/17927>.
Until Glama publishes an evaluation, the PR must describe that requirement as
pending and must not claim a quality grade.
