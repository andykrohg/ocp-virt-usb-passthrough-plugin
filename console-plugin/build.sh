#!/bin/bash

set -e

IMAGE=${IMAGE:-"quay.io/andy_krohg/usb-passthrough-plugin:latest"}

echo "Building USB Passthrough Console Plugin..."
echo "Target image: $IMAGE"
echo ""

# Step 1: Clean install dependencies
echo "1. Installing Node dependencies..."
rm -f package-lock.json
npm install
echo ""

# Step 2: Build the plugin
echo "2. Building plugin with webpack..."
npm run build

if [ ! -d "dist" ]; then
    echo "Error: dist directory not created"
    exit 1
fi

echo "✅ Plugin built successfully"
ls -lh dist/
echo ""

# Step 3: Build container image
echo "3. Building container image..."
podman build --platform linux/amd64 -t "$IMAGE" .

echo ""
echo "✅ Container image built successfully: $IMAGE"
echo ""
echo "Next steps:"
echo "  Push image:    podman push $IMAGE"
echo "  Deploy:        kubectl apply -f manifests/deployment.yaml"
echo "  Enable plugin: kubectl patch consoles.operator.openshift.io cluster \\"
echo "                   --type json -p '[{\"op\": \"add\", \"path\": \"/spec/plugins/-\", \"value\": \"usb-passthrough-plugin\"}]'"
echo ""
