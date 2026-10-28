package dlp

import (
"math"
"regexp"
"strings"
)

type Finding struct {
RuleID      string
Description string
Sample      string
}

type rule struct {
id          string
description string
pattern     *regexp.Regexp
}

type Scanner struct {
rules []rule
}

func NewScanner() *Scanner {
return &Scanner{
rules: []rule{
{
id:          "AWS_ACCESS_KEY",
description: "AWS Access Key ID detected",
pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
},
{
id:          "GITHUB_PAT",
description: "GitHub Personal Access Token detected",
pattern:     regexp.MustCompile(`(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,255}`),
},
{
id:          "PRIVATE_KEY",
description: "Cryptographic Private Key block detected",
pattern:     regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH |PGP |)PRIVATE KEY-----`),
},
{
id:          "AI_API_KEY",
description: "AI Provider API Key (OpenAI/Anthropic) detected",
pattern:     regexp.MustCompile(`sk-[a-zA-Z0-9_-]{20,}`),
},
{
id:          "SLACK_TOKEN",
description: "Slack Bot/User Token detected",
pattern:     regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z-]{10,48}`),
},
},
}
}

func (s *Scanner) InspectAndRedact(input string, redact bool) (string, []Finding) {
var findings []Finding
output := input

for _, r := range s.rules {
matches := r.pattern.FindAllString(output, -1)
for _, m := range matches {
preview := m
if len(preview) > 8 {
preview = preview[:4] + "..." + preview[len(preview)-4:]
}
findings = append(findings, Finding{
RuleID:      r.id,
Description: r.description,
Sample:      preview,
})
if redact {
replacement := "[REDACTED_BY_AGENTWALL:" + r.id + "]"
output = strings.ReplaceAll(output, m, replacement)
}
}
}

return output, findings
}

func ShannonEntropy(data string) float64 {
if len(data) == 0 {
return 0
}
freq := make(map[rune]float64)
for _, r := range data {
freq[r]++
}
var entropy float64
length := float64(len([]rune(data)))
for _, count := range freq {
p := count / length
entropy -= p * math.Log2(p)
}
return entropy
}