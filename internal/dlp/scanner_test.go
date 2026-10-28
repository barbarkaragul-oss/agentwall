package dlp

import (
"strings"
"testing"
)

func TestScannerRedaction(t *testing.T) {
s := NewScanner()
payload := `{"tool":"env_reader","aws":"AKIAIOSFODNN7EXAMPLE","gh":"ghp_1234567890abcdefghijklmnopqrstuvwxyzABCD"}`
sanitized, findings := s.InspectAndRedact(payload, true)

if len(findings) != 2 {
t.Fatalf("expected 2 findings, got %d", len(findings))
}
if strings.Contains(sanitized, "AKIAIOSFODNN7EXAMPLE") {
t.Errorf("AWS key was not redacted: %s", sanitized)
}
if !strings.Contains(sanitized, "[REDACTED_BY_AGENTWALL:AWS_ACCESS_KEY]") {
t.Errorf("expected AWS redaction marker in output")
}
}

func TestShannonEntropy(t *testing.T) {
low := ShannonEntropy("aaaaaaaaaaaaaaaaaaaa")
high := ShannonEntropy("aB3$x9!mQ2@zL8#vR5^w")
if high <= low {
t.Errorf("expected high entropy > low entropy, got %f <= %f", high, low)
}
}