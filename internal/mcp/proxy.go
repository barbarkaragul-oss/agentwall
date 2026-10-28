package mcp

import (
"bufio"
"encoding/json"
"fmt"
"io"
"os"
"os/exec"
"strings"
"sync"
"time"

"github.com/barbarkaragul-oss/agentwall/internal/dlp"
"github.com/barbarkaragul-oss/agentwall/internal/policy"
"github.com/barbarkaragul-oss/agentwall/internal/procmon"
)

const (
colorReset  = "\033[0m"
colorRed    = "\033[31m"
colorGreen  = "\033[32m"
colorYellow = "\033[33m"
colorCyan   = "\033[36m"
colorBold   = "\033[1m"
)

type Proxy struct {
pol     *policy.Policy
scanner *dlp.Scanner
logMu   sync.Mutex
}

type jsonRPCMessage struct {
JSONRPC string          `json:"jsonrpc"`
ID      json.RawMessage `json:"id,omitempty"`
Method  string          `json:"method,omitempty"`
}

func NewProxy(pol *policy.Policy) *Proxy {
return &Proxy{
pol:     pol,
scanner: dlp.NewScanner(),
}
}

func (p *Proxy) logEvent(level, color, msg string) {
p.logMu.Lock()
defer p.logMu.Unlock()
ts := time.Now().Format("15:04:05.000")
fmt.Fprintf(os.Stderr, "%s[AgentWall %s]%s %s%s%s: %s\n",
colorCyan, ts, colorReset, colorBold, color+level, colorReset, msg)
}

func (p *Proxy) Run(cmdName string, cmdArgs []string) error {
cmd := exec.Command(cmdName, cmdArgs...)
cmd.Env = os.Environ()

childStdin, err := cmd.StdinPipe()
if err != nil {
return err
}
childStdout, err := cmd.StdoutPipe()
if err != nil {
return err
}
cmd.Stderr = os.Stderr

if err := cmd.Start(); err != nil {
return err
}

p.logEvent("ACTIVE", colorGreen, fmt.Sprintf("Firewall attached to PID %d (%s %s)",
cmd.Process.Pid, cmdName, strings.Join(cmdArgs, " ")))

stopProcMon := make(chan struct{})
go procmon.StartMonitor(cmd.Process.Pid, p.pol.BlockedPaths, stopProcMon, func(alert string) {
p.logEvent("KERNEL/FD ALERT", colorRed, alert)
if p.pol.BlockOnViolation {
_ = cmd.Process.Kill()
}
})

var wg sync.WaitGroup
wg.Add(2)

go func() {
defer wg.Done()
defer childStdin.Close()
p.streamFilter("HOST->AGENT", os.Stdin, childStdin, true)
}()

go func() {
defer wg.Done()
p.streamFilter("AGENT->HOST", childStdout, os.Stdout, false)
}()

wg.Wait()
close(stopProcMon)
return cmd.Wait()
}

func (p *Proxy) streamFilter(direction string, r io.Reader, w io.Writer, isInbound bool) {
scanner := bufio.NewScanner(r)
buf := make([]byte, 0, 1024*1024)
scanner.Buffer(buf, 10*1024*1024)

for scanner.Scan() {
line := scanner.Text()

violations := p.pol.EvaluatePayload(line)
if len(violations) > 0 {
for _, v := range violations {
p.logEvent("BLOCKED ["+direction+"]", colorRed,
fmt.Sprintf("Category=%s Rule='%s' Severity=%s", v.Category, v.Rule, v.Severity))
}

if p.pol.BlockOnViolation {
if !isInbound {
errResp := buildJSONRPCError(line, violations[0].Category, violations[0].Rule)
if errResp != "" {
fmt.Fprintln(w, errResp)
}
}
continue
}
}

sanitized, findings := p.scanner.InspectAndRedact(line, p.pol.RedactSecrets)
for _, f := range findings {
p.logEvent("DLP REDACTED ["+direction+"]", colorYellow,
fmt.Sprintf("%s (%s) -> Masked in-flight", f.RuleID, f.Sample))
}

fmt.Fprintln(w, sanitized)
}
}

func buildJSONRPCError(rawLine, category, rule string) string {
var msg jsonRPCMessage
if err := json.Unmarshal([]byte(rawLine), &msg); err == nil && len(msg.ID) > 0 {
return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32600,"message":"Blocked by AgentWall Firewall [%s]: %s"}}`,
string(msg.ID), category, rule)
}
return fmt.Sprintf(`{"error":"Blocked by AgentWall Firewall [%s]: %s"}`, category, rule)
}