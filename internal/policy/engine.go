package policy

import (
"encoding/json"
"os"
"path/filepath"
"strings"
)

type Violation struct {
Category string
Rule     string
Severity string
}

type Policy struct {
BlockedPaths            []string `json:"blocked_paths"`
BlockedCommands         []string `json:"blocked_commands"`
PromptInjectionPatterns []string `json:"prompt_injection_patterns"`
RedactSecrets           bool     `json:"redact_secrets"`
BlockOnViolation        bool     `json:"block_on_violation"`
}

func DefaultPolicy() *Policy {
return &Policy{
BlockedPaths: []string{
".ssh/id_rsa",
".ssh/id_ed25519",
".aws/credentials",
".gnupg",
"/etc/shadow",
".kube/config",
".env.production",
},
BlockedCommands: []string{
"rm -rf /",
"mkfs.",
"dd if=/dev/zero",
":(){ :|:& };:",
"/dev/tcp/",
"nc -e /bin/sh",
},
PromptInjectionPatterns: []string{
"ignore previous instructions",
"ignore all prior rules",
"system prompt override",
"exfiltrate environment variables",
},
RedactSecrets:    true,
BlockOnViolation: true,
}
}

func LoadFromFile(path string) (*Policy, error) {
cleanPath := filepath.Clean(path)
data, err := os.ReadFile(cleanPath)
if err != nil {
return nil, err
}
p := DefaultPolicy()
if err := json.Unmarshal(data, p); err != nil {
return nil, err
}
return p, nil
}

func (p *Policy) SaveToFile(path string) error {
cleanPath := filepath.Clean(path)
data, err := json.MarshalIndent(p, "", "  ")
if err != nil {
return err
}
return os.WriteFile(cleanPath, data, 0644)
}

func (p *Policy) EvaluatePayload(payload string) []Violation {
var violations []Violation
lower := strings.ToLower(payload)

for _, bp := range p.BlockedPaths {
if strings.Contains(lower, strings.ToLower(bp)) {
violations = append(violations, Violation{
Category: "SENSITIVE_PATH_ACCESS",
Rule:     bp,
Severity: "CRITICAL",
})
}
}

for _, bc := range p.BlockedCommands {
if strings.Contains(lower, strings.ToLower(bc)) {
violations = append(violations, Violation{
Category: "DESTRUCTIVE_OR_EXFIL_COMMAND",
Rule:     bc,
Severity: "CRITICAL",
})
}
}

for _, pi := range p.PromptInjectionPatterns {
if strings.Contains(lower, strings.ToLower(pi)) {
violations = append(violations, Violation{
Category: "INDIRECT_PROMPT_INJECTION",
Rule:     pi,
Severity: "HIGH",
})
}
}

return violations
}