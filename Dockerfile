FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/agentwall ./cmd/agentwall

FROM alpine:3.20
COPY --from=builder /usr/local/bin/agentwall /usr/local/bin/agentwall
ENTRYPOINT ["agentwall", "serve"]