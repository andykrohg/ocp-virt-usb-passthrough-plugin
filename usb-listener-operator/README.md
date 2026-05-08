```markdown
# USB Listener Operator

Kubernetes operator for managing USB device passthrough to VMs using USB/IP protocol.

## Overview

This operator watches for `USBConnection` custom resources and manages the lifecycle of USB device connections between workstations and VMs.

## Custom Resources

### USBConnection

Represents a connection between a USB device and a VM.

```yaml
apiVersion: usb.openshift.io/v1alpha1
kind: USBConnection
metadata:
  name: cac-to-windows-vm
  namespace: default
spec:
  workstationAddress: "192.168.1.100:3240"  # Workstation USB/IP server
  deviceID: "090c:1000"                      # USB vendor:product
  deviceName: "Samsung USB Drive"            # Human-readable name
  vmName: "windows-vm"                       # Target VM
  namespace: "user-vms"                      # VM namespace
status:
  phase: Connected
  message: "USB device 090c:1000 connected to VM user-vms/windows-vm"
  connectedAt: "2026-05-08T10:15:30Z"
```

### USBDevice

Represents a USB device advertised by a workstation agent.

```yaml
apiVersion: usb.openshift.io/v1alpha1
kind: USBDevice
metadata:
  name: samsung-usb-drive
  namespace: default
spec:
  workstationAddress: "192.168.1.100:3240"
  deviceID: "090c:1000"
  deviceName: "Samsung USB Drive"
  vendorName: "Samsung"
  serial: "0375420080001253"
  isCAC: false
  owner: "akrohg"
status:
  available: true
  connectedTo: ""
  lastSeen: "2026-05-08T10:15:30Z"
```

## Development

### Prerequisites

- Go 1.23+
- Kubernetes cluster
- kubectl configured

### Build

```bash
go mod download
go build -o bin/usb-listener-operator main.go
```

### Run Locally

```bash
# Install CRDs
make install

# Run operator locally
go run main.go
```

### Deploy to Cluster

```bash
# Build and push image
make docker-build docker-push IMG=quay.io/yourorg/usb-listener-operator:latest

# Deploy
make deploy IMG=quay.io/yourorg/usb-listener-operator:latest
```

## Testing

Create a test USBConnection:

```bash
kubectl apply -f - <<EOF
apiVersion: usb.openshift.io/v1alpha1
kind: USBConnection
metadata:
  name: test-connection
spec:
  workstationAddress: "192.168.1.100:3240"
  deviceID: "090c:1000"
  deviceName: "Test Device"
  vmName: "test-vm"
  namespace: "default"
EOF
```

Watch the connection status:

```bash
kubectl get usbconn test-connection -w
```

## How It Works

1. **User creates USBConnection CR** via console plugin or kubectl
2. **Operator reconciles** the resource
3. **Operator connects** to workstation's USB/IP server
4. **Operator attaches** USB device to target VM using virtctl
5. **Status updated** to reflect connection state

## Architecture

```
┌──────────────────┐
│  USBConnection   │
│  Custom Resource │
└────────┬─────────┘
         │
         │ Watches
         ▼
┌──────────────────────┐
│  Operator Controller │
│  • Reconcile loop    │
│  • Connect USB/IP    │
│  • Attach to VM      │
└────────┬─────────────┘
         │
         ├─────────────►┌──────────────────┐
         │              │ Workstation      │
         │              │ USB/IP Server    │
         │              └──────────────────┘
         │
         └─────────────►┌──────────────────┐
                        │ VM (virtctl)     │
                        └──────────────────┘
```

## Next Steps

- [ ] Implement actual USB/IP client connection
- [ ] Add device health checking
- [ ] Support multiple simultaneous connections
- [ ] Add metrics and monitoring
- [ ] Implement graceful connection cleanup
- [ ] Add TLS for USB/IP traffic encryption

## License

MIT
```
