#!/bin/bash

set -e

echo "🔨 Building workstation agent for all platforms..."
echo ""

# Create dist directory
mkdir -p dist

# Build for each platform
echo "Building macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/usb-agent-darwin-arm64

echo "Building macOS (Intel)..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/usb-agent-darwin-amd64

echo "Building Linux (amd64)..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/usb-agent-linux-amd64

echo "Building Linux (arm64)..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/usb-agent-linux-arm64

echo "Building Windows (amd64)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/usb-agent-windows-amd64.exe

echo ""
echo "✅ Build complete! Binaries in dist/:"
ls -lh dist/
echo ""
echo "To create a GitHub release:"
echo "  1. Create a git tag: git tag v0.1.0"
echo "  2. Push the tag: git push origin v0.1.0"
echo "  3. GitHub Actions will automatically build and create a release"
echo ""
echo "Or upload manually to:"
echo "  https://github.com/andykrohg/ocp-virt-usb-passthrough-plugin/releases/new"
