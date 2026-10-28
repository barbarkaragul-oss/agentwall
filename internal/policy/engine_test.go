package policy

import "testing"

func TestEvaluatePayloadViolations(t *testing.T) {
pol := DefaultPolicy()
v := pol.EvaluatePayload(`{"uri":"file:///home/user/.ssh/id_rsa"}`)
if len(v) == 0 || v[0].Rule != ".ssh/id_rsa" {
t.Fatalf("expected .ssh/id_rsa violation, got %+v", v)
}
}