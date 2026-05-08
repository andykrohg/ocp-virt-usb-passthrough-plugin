# USB Passthrough for OpenShift Virtualization

A Kubernetes-native solution for USB device passthrough to VMs using USB/IP protocol.

## Architecture

This project consists of three components:

### 1. USB Listener Operator (`usb-listener-operator/`)

Kubernetes operator running in the cluster that:
- Manages USB device connections via CustomResources
- Discovers available USB devices from workstation agents
- Proxies USB/IP traffic to target VMs
- Integrates with virtctl for VM attachment

### 2. Workstation Agent (`workstation-agent/`)

Lightweight Go binary (~5MB) running on user workstations that:
- Runs USB/IP server exposing local USB devices
- Registers available devices with the cluster
- Provides system tray UI showing connection status
- Auto-starts and runs in background

### 3. Console Plugin (`console-plugin/`)

OpenShift Console Dynamic Plugin that:
- Provides browser-based UI for USB passthrough
- Lists available USB devices from cluster
- Allows selecting VM and device for connection
- Creates USBConnection resources via Kubernetes API

## How It Works

```
User Workstation              OpenShift Cluster
┌──────────────┐             ┌────────────────────────┐
│ USB Devices  │             │  Console Plugin        │
│      │       │             │  (Browser)             │
│      ▼       │             └───────┬────────────────┘
│ Workstation  │                     │ Creates CR
│ Agent        │                     ▼
│ (USB/IP srv) │             ┌──────────────────────┐
│      │       │             │ USB Listener         │
│      │       │ USB/IP      │ Operator             │
│      └───────┼────────────►│ (Reconciles)         │
│              │ Protocol    └───────┬──────────────┘
│              │                     │
│              │                     │ virtctl attach
│              │                     ▼
│              │             ┌──────────────────────┐
│              │             │ VM with USB device   │
└──────────────┘             └──────────────────────┘
```

## Quick Start

### Install the Operator

```bash
cd usb-listener-operator
make install
make deploy
```

### Install Workstation Agent

```bash
cd workstation-agent
go build -o usb-agent
./usb-agent --kubeconfig ~/.kube/config
```

### Install Console Plugin

```bash
cd console-plugin
npm install
npm run build
oc apply -f manifests/plugin.yaml
```

## Usage

1. Start workstation agent on your machine
2. Log into OpenShift Console
3. Navigate to Virtualization → USB Passthrough
4. Select your VM
5. Select USB device from your workstation
6. Click "Connect"

The USB device will be redirected to the VM!

## Components

See individual component READMEs for detailed information:
- [USB Listener Operator](./usb-listener-operator/README.md)
- [Workstation Agent](./workstation-agent/README.md)
- [Console Plugin](./console-plugin/README.md)

## Benefits

- ✅ **Browser-based UI** - No desktop app needed
- ✅ **Kubernetes native** - Full RBAC, audit logs, GitOps support
- ✅ **Centralized management** - IT controls all USB connections
- ✅ **Multi-user** - Many users can connect simultaneously
- ✅ **Lightweight client** - 5MB agent vs 160MB desktop app
- ✅ **Auto-discovery** - Devices automatically appear in console
- ✅ **Enterprise ready** - Proper auth, encryption, monitoring

## License

MIT
