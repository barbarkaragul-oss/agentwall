#!/usr/bin/env python3
"""Exercise an MCP stdio process using only Python's standard library.

Examples:
  python3 scripts/check_mcp_stdio.py ./bin/agentwall serve
  python3 scripts/check_mcp_stdio.py docker run --rm -i --network none agentwall-mcp
This is a regression check for the advertised tool surface, not certification.
"""
import json
import queue
import subprocess
import sys
import threading


def main():
    command = sys.argv[1:]
    if not command:
        raise SystemExit("usage: check_mcp_stdio.py COMMAND [ARGS...]")
    process = subprocess.Popen(command, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, text=True, encoding="utf-8")
    lines = queue.Queue()
    errors = []

    def read_stdout():
        for line in process.stdout:
            lines.put(line)
        lines.put(None)

    def read_stderr():
        errors.extend(process.stderr)

    stdout_thread = threading.Thread(target=read_stdout, daemon=True)
    stderr_thread = threading.Thread(target=read_stderr, daemon=True)
    stdout_thread.start()
    stderr_thread.start()

    def send(value):
        process.stdin.write(json.dumps(value, ensure_ascii=False) + "\n")
        process.stdin.flush()

    def receive(request_id):
        try:
            line = lines.get(timeout=15)
        except queue.Empty:
            raise AssertionError("MCP response timed out") from None
        assert line is not None, "MCP server exited before replying"
        response = json.loads(line)
        assert response["jsonrpc"] == "2.0", response
        assert response["id"] == request_id, response
        assert ("result" in response) != ("error" in response), response
        return response

    def call(request_id, method, params=None):
        message = {"jsonrpc": "2.0", "id": request_id, "method": method}
        if params is not None:
            message["params"] = params
        send(message)
        return receive(request_id)

    try:
        init = call(1, "initialize", {
            "protocolVersion": "2025-11-25", "capabilities": {},
            "clientInfo": {"name": "agentwall-stdio-check", "version": "1.0"},
        })["result"]
        assert init["protocolVersion"] == "2025-11-25", init
        assert "tools" in init["capabilities"], init
        assert init["serverInfo"]["name"] == "agentwall", init
        send({"jsonrpc": "2.0", "method": "notifications/initialized"})
        listed = call("discover", "tools/list")["result"]["tools"]
        assert len(listed) == 1 and listed[0]["name"] == "inspect_payload", listed
        schema = listed[0]["inputSchema"]
        assert schema["type"] == "object" and schema["required"] == ["payload"], schema
        assert schema["properties"]["payload"]["type"] == "string", schema

        # Public AWS documentation example; never send real credentials in tests.
        payload = "AWS_KEY=AKIAIOSFODNN7EXAMPLE path=/home/test/.ssh/id_rsa İstanbul \u0085\u200b"
        result = call(3, "tools/call", {
            "name": "inspect_payload", "arguments": {"payload": payload},
        })["result"]
        assert not result.get("isError", False), result
        assert result["content"][0]["type"] == "text", result
        inspection = json.loads(result["content"][0]["text"])
        assert inspection["secrets"] == 1 and inspection["violations"] == 1, inspection
        assert "AKIAIOSFODNN7EXAMPLE" not in inspection["sanitized"], inspection
        assert inspection["sanitized"].endswith("İstanbul \u0085\u200b"), inspection

        assert call(4, "missing_method")["error"]["code"] == -32601
        assert call(5, "tools/call", {"name": "missing_tool", "arguments": {}})["error"]["code"] == -32602
        invalid = call(6, "tools/call", {"name": "inspect_payload", "arguments": {"payload": 42}})
        assert invalid["result"]["isError"] is True, invalid
        process.stdin.write("{\n")
        process.stdin.flush()
        assert receive(None)["error"]["code"] == -32700
        send({"jsonrpc": "2.0", "method": "notifications/unknown"})
        assert call(7, "ping")["result"] == {}
        process.stdin.close()
        assert process.wait(timeout=15) == 0, "nonzero server exit"
        stdout_thread.join(timeout=2)
        stderr_thread.join(timeout=2)
        assert lines.get(timeout=2) is None, "extra stdout after final response"
        assert not errors, "unexpected diagnostics: " + "".join(errors)
        print("PASS: initialize, notification, discovery, redaction, Unicode, errors, ping, EOF shutdown")
    finally:
        if process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)


if __name__ == "__main__":
    main()
