<div align="center">

# ğŸ›¡ï¸ AgentWall

**Zero-Dependency Runtime Firewall, Secret DLP & Process Sandbox for AI Agents and MCP Servers**

[![CI Status](https://github.com/barbarkaragul-oss/agentwall/actions/workflows/ci.yml/badge.svg)](https://github.com/barbarkaragul-oss/agentwall/actions)
[![Release](https://img.shields.io/github/v/release/barbarkaragul-oss/agentwall?color=brightgreen)](https://github.com/barbarkaragul-oss/agentwall/releases)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-0-blue.svg)](#)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

*Stop autonomous AI agents (Claude Code, Cursor, OpenClaw, MCP Servers) from leaking your .env secrets, reading ~/.ssh/id_rsa, executing m -rf /, or falling for Indirect Prompt Injections.*

<br/>
<img src="assets/demo.svg" alt="AgentWall Live Demo" width="100%"/>

</div>

---

## ğŸš€ One-Line Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/barbarkaragul-oss/agentwall/main/install.sh | sh
```

Or build from source with **zero external dependencies**:
```bash
git clone https://github.com/barbarkaragul-oss/agentwall.git
cd agentwall
go build -ldflags="-s -w" -o bin/agentwall ./cmd/agentwall
```

---

## ğŸ–¥ï¸ Real-Time Terminal UI Dashboard (gentwall ui)

Launch the built-in interactive telemetry monitor to inspect JSON-RPC packets, microsecond filter latency, and blocked exfiltration attempts in real time:

```text
â•”â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•—
â•‘ ğŸ›¡ï¸  AGENTWALL v1.1.0 â€” REAL-TIME AI AGENT & MCP SECURITY DASHBOARD         â•‘
â• â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•£
â•‘ PACKETS INSPECTED: 6        â”‚ THREATS BLOCKED: 3      â”‚ SECRETS MASKED: 2  â•‘
â•‘ AVG FILTER LATENCY: 21 Âµs   â”‚ SANDBOX STATUS:  ACTIVE â”‚ PROCFS WATCHER: ON â•‘
â• â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•£
â•‘ TIME     â”‚ ACTION   â”‚ SEVERITY â”‚ LATENCY â”‚ EVENT DETAIL                    â•‘
â•Ÿâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â•¢
â•‘ 22:35:01 â”‚ ALLOWED  â”‚ CLEAN    â”‚ 12 Âµs   â”‚ Verified JSON-RPC packet        â•‘
â•‘ 22:35:01 â”‚ REDACTED â”‚ HIGH     â”‚ 19 Âµs   â”‚ AWS_ACCESS_KEY masked           â•‘
â•‘ 22:35:02 â”‚ BLOCKED  â”‚ CRITICAL â”‚ 26 Âµs   â”‚ SENSITIVE_PATH (.ssh/id_rsa)    â•‘
â•‘ 22:35:02 â”‚ BLOCKED  â”‚ HIGH     â”‚ 33 Âµs   â”‚ INDIRECT_PROMPT_INJECTION       â•‘
â•‘ 22:35:03 â”‚ REDACTED â”‚ HIGH     â”‚ 40 Âµs   â”‚ GITHUB_PAT masked               â•‘
â•‘ 22:35:03 â”‚ BLOCKED  â”‚ CRITICAL â”‚ 18 Âµs   â”‚ DESTRUCTIVE_CMD (rm -rf /)      â•‘
â•šâ•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•
```

---

## ğŸ—ï¸ Architecture

```mermaid
flowchart LR
    Host["IDE / LLM Host\n(Claude Code, Cursor)"] <-->|JSON-RPC / Stdio| AW["ğŸ›¡ï¸ AgentWall Proxy\nâ€¢ Real-time Secret DLP\nâ€¢ Prompt Injection Filter\nâ€¢ Policy Engine"]
    AW <-->|Sanitized Stream| Agent["MCP Server / AI Agent\n(Untrusted Process)"]
    ProcMon["ğŸ” OS ProcFS Monitor\n(/proc/pid/fd)"] -.->|Watches Open Files| Agent
    AW --- ProcMon
```

---

## ğŸ”Œ Drop-In MCP Server Integration

Add gentwall wrap -- in front of any untrusted MCP server inside claude_desktop_config.json or .cursor/mcp.json:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "agentwall",
      "args": ["wrap", "--", "npx", "-y", "@modelcontextprotocol/server-filesystem", "."]
    }
  }
}
```

## ğŸ“„ License
MIT License â€” Copyright (c) 2026 Barbaros Karagulmez (arbarkaragul-oss)