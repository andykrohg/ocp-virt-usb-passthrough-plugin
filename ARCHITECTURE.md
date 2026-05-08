# USB Passthrough Architecture

## Overview

This system enables USB device passthrough from user workstations to OpenShift virtualization VMs using a Kubernetes-native approach with USB/IP protocol.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     User Workstation                            │
│                                                                 │
│  ┌──────────────┐                                              │
│  │ USB Devices  │                                              │
│  │  • CAC Card  │                                              │
│  │  • USB Drive │                                              │
│  │  • etc.      │                                              │
│  └──────┬───────┘                                              │
│         │                                                       │
│         ▼                                                       │
│  ┌──────────────────────────────────────┐                     │
│  │   USB Workstation Agent              │                     │
│  │   • Enumerates local USB devices     │                     │
│  │   • Runs USB/IP server (port 3240)   │                     │
│  │   • Registers devices with cluster   │                     │
│  │   • System tray UI                   │                     │
│  └──────────────┬───────────────────────┘                     │
│                 │                                               │
└─────────────────┼───────────────────────────────────────────────┘
                  │
                  │ 1. Register USBDevice CRs
                  │ 2. USB/IP Protocol (TCP:3240)
                  │
┌─────────────────┼───────────────────────────────────────────────┐
│   OpenShift     ▼                                               │
│  ┌───────────────────────────────────────────────────────┐     │
│  │              Kubernetes API Server                    │     │
│  │  • USBDevice CRs                                      │     │
│  │  • USBConnection CRs                                  │     │
│  │  • VirtualMachineInstance resources                  │     │
│  └─────────┬─────────────────────────────┬───────────────┘     │
│            │                             │                     │
│            │                             │                     │
│  ┌─────────▼──────────────┐    ┌────────▼──────────────┐     │
│  │  Console Plugin        │    │  USB Listener         │     │
│  │  (Browser UI)          │    │  Operator             │     │
│  │                        │    │                       │     │
│  │  • Lists USB devices   │    │  • Watches            │     │
│  │  • Lists VMs           │    │    USBConnection CRs  │     │
│  │  • Creates             │    │  • Connects to        │     │
│  │    USBConnection CRs   │    │    workstation        │     │
│  │                        │    │  • Attaches USB       │     │
│  └────────────────────────┘    │    to VM via virtctl  │     │
│                                 └────────┬──────────────┘     │
│                                          │                     │
│                                          │                     │
│                                 ┌────────▼──────────────┐     │
│                                 │  Virtual Machine      │     │
│                                 │  • Receives USB       │     │
│                                 │    device             │     │
│                                 │  • User logs in       │     │
│                                 │    with CAC           │     │
│                                 └───────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Workstation Agent (~5MB Go binary)

**Purpose**: Lightweight client running on user workstations

**Responsibilities**:
- Enumerate local USB devices (system_profiler/lsusb)
- Run USB/IP server on port 3240
- Register USBDevice CRs in the cluster
- Send heartbeats to keep device status current
- Provide system tray UI showing connection status

**Technology**: Go 1.23, USB/IP protocol

**Platform Support**: macOS, Linux, Windows

### 2. USB Listener Operator

**Purpose**: Kubernetes operator managing USB connections

**Responsibilities**:
- Watch USBConnection custom resources
- Connect to workstation's USB/IP server
- Attach USB devices to VMs using virtctl
- Update connection status
- Handle disconnections gracefully

**Technology**: Go 1.23, controller-runtime, Kubebuilder

**Custom Resources**:
- `USBConnection` - Represents active USB passthrough
- `USBDevice` - Represents available USB device from workstation

### 3. Console Plugin

**Purpose**: Browser-based UI in OpenShift Console

**Responsibilities**:
- Display available USB devices (from USBDevice CRs)
- Display running VMs
- Allow user selection
- Create USBConnection CRs
- Show active connections and status

**Technology**: React 18, TypeScript, PatternFly 5, OpenShift Console SDK

## Data Flow

### Device Registration Flow

```
1. User starts workstation agent
   └─> Agent enumerates USB devices
   └─> Agent starts USB/IP server on port 3240
   └─> Agent creates USBDevice CRs in cluster
   └─> Agent sends periodic heartbeats (30s interval)
```

### Connection Creation Flow

```
1. User opens OpenShift Console
2. Navigate to "USB Passthrough" page
3. Console plugin loads:
   ├─> USBDevice CRs (available devices)
   └─> VirtualMachineInstance resources (running VMs)
4. User selects USB device
5. User selects target VM
6. User clicks "Connect"
7. Console plugin creates USBConnection CR:
   apiVersion: usb.openshift.io/v1alpha1
   kind: USBConnection
   spec:
     workstationAddress: "192.168.1.100:3240"
     deviceID: "090c:1000"
     vmName: "windows-vm"
     namespace: "user-vms"
8. Operator reconciles USBConnection:
   ├─> Connects to workstation USB/IP server
   ├─> Attaches device to VM via virtctl
   └─> Updates status to "Connected"
9. Device is now usable in the VM
```

### Disconnection Flow

```
1. User deletes USBConnection CR
2. Operator reconciles:
   ├─> Detaches device from VM
   ├─> Closes USB/IP connection
   └─> Updates USBDevice status to "available"
3. Device returns to workstation
```

## Custom Resource Definitions

### USBDevice

Represents a USB device advertised by a workstation.

```yaml
apiVersion: usb.openshift.io/v1alpha1
kind: USBDevice
metadata:
  name: samsung-usb-090c-1000
  namespace: default
spec:
  workstationAddress: "192.168.1.100:3240"
  deviceID: "090c:1000"
  deviceName: "Samsung USB Drive"
  vendorName: "Samsung"
  serial: "0375420080001253"
  isCAC: false
  owner: "alice"
status:
  available: true
  connectedTo: ""  # "namespace/vmname" when connected
  lastSeen: "2026-05-08T10:30:00Z"
```

### USBConnection

Represents an active USB passthrough connection.

```yaml
apiVersion: usb.openshift.io/v1alpha1
kind: USBConnection
metadata:
  name: cac-to-windows-vm
  namespace: user-vms
spec:
  workstationAddress: "192.168.1.100:3240"
  deviceID: "090c:1000"
  deviceName: "CAC Card Reader"
  vmName: "windows-vm"
  namespace: "user-vms"
status:
  phase: Connected  # Pending|Connecting|Connected|Failed
  message: "USB device 090c:1000 connected to VM user-vms/windows-vm"
  connectedAt: "2026-05-08T10:15:30Z"
  lastError: ""
```

## Security Considerations

### Authentication & Authorization

- Workstation agent uses kubeconfig for cluster authentication
- RBAC controls who can create USBConnection resources
- Namespace isolation prevents unauthorized VM access

### Network Security

- USB/IP traffic should use TLS wrapper (future enhancement)
- Firewall rules should restrict USB/IP port to cluster nodes
- Consider VPN for workstation-to-cluster connectivity

### Privilege Requirements

- Workstation agent needs root/admin for USB access
- Operator runs with ServiceAccount permissions
- No privileged containers required in cluster

## Advantages Over Desktop App

| Aspect | Desktop App | Console Plugin + Operator |
|--------|-------------|---------------------------|
| **Installation** | Every user downloads 160MB app | 5MB agent + cluster-side deployment |
| **Updates** | Per-user manual update | Centralized operator update |
| **UI** | Desktop window | Browser-based console integration |
| **Multi-user** | Independent instances | Shared infrastructure |
| **Management** | Decentralized | IT controls via Kubernetes |
| **Audit** | Local logs only | K8s audit logs, RBAC |
| **Discoverability** | Users must know about app | Integrated in console navigation |

## Future Enhancements

### Phase 1 (Current)
- [x] Basic operator with CRDs
- [x] Workstation agent skeleton
- [x] Console plugin UI
- [x] USB/IP client implementation in operator
- [x] Platform-specific USB enumeration (macOS, Linux, Windows)

### Phase 2 (Next)
- [ ] Device health monitoring and connection keepalive
- [ ] TLS encryption for USB/IP traffic
- [ ] Enhanced error handling and recovery
- [ ] Integration tests for end-to-end flow

### Phase 3
- [ ] Multi-device simultaneous connections
- [ ] Connection metrics and monitoring
- [ ] Auto-discovery via mDNS/Avahi
- [ ] Windows VM USB/IP client installer

### Phase 4
- [ ] Connection templates/profiles
- [ ] Device favorites
- [ ] Connection history
- [ ] Advanced filtering and search

## Comparison with virtctl Approach

### Desktop App + virtctl (Previous)
```
User → Desktop App → Local virtctl → VM
  ✓ Simple implementation
  ✗ Large binary distribution
  ✗ No centralized control
  ✗ Per-user installation
```

### Console Plugin + Operator (Current)
```
User → Browser → Operator → virtctl → VM
  ✓ Browser-based UI
  ✓ Centralized management
  ✓ Small agent footprint
  ✓ Kubernetes-native
  ✗ More complex architecture
```

## Deployment Guide

See individual component READMEs:
- [Operator Deployment](./usb-listener-operator/README.md)
- [Agent Installation](./workstation-agent/README.md)
- [Console Plugin Setup](./console-plugin/README.md)

## License

MIT
