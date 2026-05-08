#!/bin/bash

set -e

IMAGE=${IMAGE:-"quay.io/andy_krohg/usb-listener-operator:latest"}

echo "Building USB Listener Operator..."
echo "Target image: $IMAGE"
echo ""

# Step 1: Cross-compile Go binary for linux/amd64
echo "1. Cross-compiling Go binary for linux/amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o manager main.go

if [ ! -f manager ]; then
    echo "Error: Binary not created"
    exit 1
fi

echo "✅ Binary built successfully"
file manager
echo ""

# Step 2: Build container image
echo "2. Building container image..."
podman build -f Containerfile.simple --platform linux/amd64 -t "$IMAGE" .

echo ""
echo "✅ Container image built successfully: $IMAGE"
echo ""
echo "Next steps:"
echo "  Push image:    podman push $IMAGE"
echo "  Deploy:        kubectl apply -f config/manager/deployment.yaml"
echo ""
