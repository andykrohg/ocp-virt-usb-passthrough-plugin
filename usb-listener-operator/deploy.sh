#!/bin/bash

set -e

echo "Deploying USB Passthrough Operator..."

# Install CRDs
echo "Installing CRDs..."
kubectl apply -f config/crd/usbdevice-crd.yaml
kubectl apply -f config/crd/usbconnection-crd.yaml

# Create namespace and RBAC
echo "Creating namespace and RBAC..."
kubectl apply -f config/manager/deployment.yaml  # Contains namespace
kubectl apply -f config/rbac/rbac.yaml

# Wait for CRDs to be established
echo "Waiting for CRDs to be established..."
kubectl wait --for condition=established --timeout=60s crd/usbdevices.usb.openshift.io
kubectl wait --for condition=established --timeout=60s crd/usbconnections.usb.openshift.io

echo ""
echo "✅ CRDs and RBAC installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Build the operator image:"
echo "     cd usb-listener-operator"
echo "     podman build -t quay.io/yourorg/usb-listener-operator:latest ."
echo "     podman push quay.io/yourorg/usb-listener-operator:latest"
echo ""
echo "  2. Update config/manager/deployment.yaml with your image"
echo ""
echo "  3. Deploy the operator:"
echo "     kubectl apply -f config/manager/deployment.yaml"
echo ""
echo "  4. Run the workstation agent:"
echo "     cd ../workstation-agent"
echo "     go build -o usb-agent"
echo "     sudo ./usb-agent -kubeconfig ~/.kube/config"
echo ""
