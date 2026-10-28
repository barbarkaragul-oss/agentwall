# Contributing to AgentWall

Contributions, new DLP regex rules, and prompt injection signatures are welcome!

~~~bash
go test -v -race ./...
go build -ldflags="-s -w" -o bin/agentwall ./cmd/agentwall
./bin/agentwall bench
~~~