# USB Passthrough for OpenShift Virtualization

Browser-based USB device passthrough to VMs using virtctl.

## Architecture

This project consists of two components:

### 1. Workstation Agent (`workstation-agent/`)

Lightweight Go binary running on user workstations that:
- Enumerates local USB devices (including CAC readers)
- Exposes HTTP API for browser console plugin
- Executes `virtctl usbredir` to attach devices to VMs
- Manages active USB passthrough connections

### 2. Console Plugin (`console-plugin/`)

OpenShift Console Dynamic Plugin that:
- Adds "USB Devices" tab to VM details pages
- Lists available USB devices from localhost agent
- Allows attaching/detaching devices with one click
- Shows connection status in real-time

## How It Works

```
User Workstation              OpenShift Cluster
┌──────────────┐             
│ USB Devices  │             
│      │       │             
│      ▼       │             
│ Workstation  │             ┌────────────────────────┐
│ Agent        │             │  Console Plugin        │
│ (HTTP API)   │◄────────────│  (Browser)             │
│      │       │ localhost   │  http://localhost:8080 │
│      │       │ API calls   └────────────────────────┘
│      │       │             
│      │       │                     
│      │ virtctl usbredir            
│      │ (connects to VM)            
│      └───────┼────────────►┌──────────────────────┐
│              │             │ VM with USB device   │
└──────────────┘             └──────────────────────┘
```

## Quick Start

### Install Workstation Agent

```bash
cd workstation-agent
go build -o usb-agent
./usb-agent --kubeconfig ~/.kube/config
```

The agent will start on `http://localhost:8080`.

### Install Console Plugin

```bash
cd console-plugin

# Deploy plugin to cluster
kubectl apply -f manifests/deployment.yaml

# Enable plugin in console
kubectl patch consoles.operator.openshift.io cluster \
  --type json \
  -p '[{"op": "add", "path": "/spec/plugins/-", "value": "usb-passthrough-plugin"}]'
```

### Prerequisites

- **virtctl** installed on workstation ([download](https://docs.openshift.com/container-platform/latest/virt/virt-using-the-cli-tools.html))
- **Valid kubeconfig** for OpenShift cluster
- **VMs must have `clientPassthrough: {}` enabled** (max 4 USB devices per VM)

## Usage

1. **Start workstation agent** on your local machine:
   ```bash
   ./usb-agent --kubeconfig ~/.kube/config
   ```

2. **Open OpenShift Console** in your browser

3. **Navigate to Virtualization → VirtualMachines**

4. **Click on a VM** → **USB Devices tab**

5. **Select USB device** from dropdown (CAC readers shown with 🔒)

6. **Click "Attach Device"**

The USB device will be redirected to the VM! Click "Detach" to disconnect.

## Components

See individual component READMEs for detailed information:
- [Workstation Agent](./workstation-agent/README.md)
- [Console Plugin](./console-plugin/README.md)

## Benefits

- ✅ **Browser-based UI** - Integrated into OpenShift Console
- ✅ **No cluster state** - Devices stay on user's workstation, not in CRs
- ✅ **Lightweight** - Small Go binary + virtctl
- ✅ **Secure** - Browser talks to localhost only, no network exposure
- ✅ **Simple** - No operator, no CRDs, just virtctl
- ✅ **CAC card support** - Auto-detects and highlights CAC readers

## Platform Support

- **macOS**: ✅ USB enumeration via `system_profiler`
- **Linux**: ✅ USB enumeration via `lsusb`
- **Windows**: ✅ USB enumeration via PowerShell

All platforms require `virtctl` for USB redirection.

## License

MIT
