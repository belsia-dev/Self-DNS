#!/bin/bash
# Run This Script For Self Buildinggggggggg
set -e

export PATH="/usr/local/bin:/usr/local/go/bin:/opt/homebrew/bin:$HOME/go/bin:$PATH"
if command -v go >/dev/null; then
    export PATH="$PATH:$(go env GOPATH)/bin"
fi

echo "=== Building SelfDNS Control Center ==="

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

cd ui
wails build -clean
cd ..

echo ""
echo "=== Build complete ==="
echo "Run the app:  open The app in ui/build/bin/SelfDNS Control Center.app"
