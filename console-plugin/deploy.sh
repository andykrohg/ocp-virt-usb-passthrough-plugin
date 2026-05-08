#!/bin/bash

set -e

IMAGE=${IMAGE:-"quay.io/andy_krohg/usb-passthrough-plugin:latest"}

echo "Deploying USB Passthrough Console Plugin..."
echo ""

# Step 1: Build if needed
if [ ! -d "dist" ] || [ "$1" == "--rebuild" ]; then
    echo "Building plugin..."
    ./build.sh
    echo ""
fi

# Step 2: Deploy to cluster
echo "1. Deploying plugin to cluster..."
kubectl apply -f manifests/deployment.yaml

echo "Waiting for deployment to be ready..."
kubectl wait --for=condition=available --timeout=120s \
  deployment/usb-passthrough-plugin -n usb-passthrough-plugin || true

echo ""

# Step 3: Check if plugin is already enabled
PLUGIN_ENABLED=$(kubectl get consoles.operator.openshift.io cluster -o jsonpath='{.spec.plugins}' | grep -c "usb-passthrough-plugin" || echo "0")

if [ "$PLUGIN_ENABLED" == "0" ]; then
    echo "2. Enabling plugin in OpenShift Console..."
    kubectl patch consoles.operator.openshift.io cluster \
      --type json \
      -p '[{"op": "add", "path": "/spec/plugins/-", "value": "usb-passthrough-plugin"}]'

    echo ""
    echo "⚠️  Console pods will restart to load the plugin (may take 1-2 minutes)"
else
    echo "2. Plugin already enabled in console"
fi

echo ""
echo "✅ Deployment complete!"
echo ""
echo "Plugin status:"
kubectl get consoleplugin usb-passthrough-plugin
echo ""
kubectl get pods -n usb-passthrough-plugin
echo ""
echo "To access:"
echo "  1. Wait for console pods to restart (if first-time deployment)"
echo "  2. Navigate to Virtualization → VirtualMachines"
echo "  3. Click on a VM"
echo "  4. Look for 'USB Devices' tab"
echo ""
