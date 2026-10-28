#!/usr/bin/env sh
set -e
REPO="barbarkaragul-oss/agentwall"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac
BIN="agentwall-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${BIN}"
echo "Downloading AgentWall for ${OS}/${ARCH}..."
curl -fsSL "$URL" -o agentwall
chmod +x agentwall
sudo mv agentwall /usr/local/bin/agentwall 2>/dev/null || mv agentwall ./agentwall
echo "✔ AgentWall installed successfully! Run 'agentwall ui' or 'agentwall demo'."