# USB/IP Client Library

This package implements a USB/IP protocol client for connecting to remote USB/IP servers (workstation agents) and managing USB device attachments.

## Architecture

```
┌─────────────────────┐         USB/IP Protocol        ┌─────────────────────┐
│  Operator Pod       │◄──────── TCP Port 3240 ────────│  Workstation Agent  │
│  (This Library)     │                                 │  (USB/IP Server)    │
└─────────────────────┘                                 └─────────────────────┘
         │                                                        │
         │ virtctl usbredir                                      │ Local USB
         ▼                                                        ▼
┌─────────────────────┐                                 ┌─────────────────────┐
│  VirtualMachine     │                                 │  USB Devices        │
│  (QEMU/KubeVirt)    │                                 │  • CAC Readers      │
└─────────────────────┘                                 │  • USB Drives       │
                                                        └─────────────────────┘
```

## Components

### 1. Protocol (`protocol.go`)

Implements the USB/IP wire protocol - a binary protocol for remote USB device sharing.

**Key Operations:**
- `OpReqDevlist (0x8005)` - Request list of available USB devices
- `OpRepDevlist (0x0005)` - Reply with device list
- `OpReqImport (0x8003)` - Request to attach/import a device
- `OpRepImport (0x0003)` - Reply with attachment confirmation

**Binary Format:**
- Big-endian encoding
- Fixed-size headers (8 bytes for requests, variable for responses)
- Device info includes vendor/product IDs, bus numbers, speed, etc.

**Functions:**
- `NewDeviceListRequest()` - Creates binary request to list devices
- `NewImportRequest(busID)` - Creates binary request to attach device
- `ParseDeviceListReply(data)` - Parses binary device list response
- `ParseImportReply(data)` - Parses binary attachment response

### 2. Client (`client.go`)

TCP client for communicating with USB/IP servers.

**Usage:**
```go
// Connect to workstation's USB/IP server
client := usbip.NewClient("192.168.1.100:3240")
if err := client.Connect(); err != nil {
    return err
}
defer client.Close()

// List available devices
devices, err := client.ListDevices()
if err != nil {
    return err
}

// Find specific device by vendor:product
device, err := client.FindDevice("090c:1000")
if err != nil {
    return err
}

// Attach device by bus ID
info, err := client.AttachDevice(device.BusID)
if err != nil {
    return err
}
```

**Features:**
- 10-second timeout for all operations
- Automatic connection management
- Error handling for network and protocol errors

### 3. VirtctlBridge (`virtctl_bridge.go`)

Integration layer between USB/IP client and virtctl (OpenShift Virtualization CLI).

**Methods:**

#### `AttachUSBDevice(ctx, workstationAddress, vendorProduct, vmName, namespace)`

High-level method using virtctl's built-in USB/IP support.

```go
bridge := usbip.NewVirtctlBridge("")
err := bridge.AttachUSBDevice(
    ctx,
    "192.168.1.100:3240",
    "067b:2303",  // CAC reader
    "windows-vm",
    "production",
)
```

**How it works:**
- Executes `virtctl usbredir <vendor>:<product> <vm-name> -n <namespace>`
- Sets `USBIP_SERVER` environment variable to point to workstation
- virtctl handles the USB/IP connection and QEMU integration

#### `AttachUSBDeviceViaProxy(...)` (Placeholder)

Future method for more control over USB/IP connection:
1. Connect to workstation's USB/IP server
2. Find and attach device via USB/IP protocol
3. Keep connection alive and bridge to VM
4. Allows custom error handling and monitoring

#### `CheckVirtctlVersion()`

Verifies virtctl is available and returns version string.

## Integration with Operator

The operator controller uses this library in the reconciliation loop:

```go
// Step 1: Connect to workstation USB/IP server
usbipClient := usbip.NewClient(workstationAddress)
if err := usbipClient.Connect(); err != nil {
    // Handle connection error
}

// Step 2: Find requested device
device, err := usbipClient.FindDevice(deviceID)
if err != nil {
    // Device not found
}

// Step 3: Attach device via USB/IP
if _, err := usbipClient.AttachDevice(device.BusID); err != nil {
    // Attachment failed
}

// Step 4: Redirect to VM via virtctl
bridge := usbip.NewVirtctlBridge("")
if err := bridge.AttachUSBDevice(ctx, workstationAddr, deviceID, vmName, namespace); err != nil {
    // VM attachment failed
}

// Store client for later cleanup
r.connections[key] = usbipClient
```

## USB/IP Protocol Details

### Device List Request

```
Offset  Size  Field
------  ----  -----
0       2     Version (0x0111)
2       2     Command (0x8005)
4       4     Status (0x00000000)
```

### Device List Reply

```
Offset  Size  Field
------  ----  -----
0       2     Version (0x0111)
2       2     Command (0x0005)
4       4     Status
8       4     Number of exported devices
12+     var   Device entries (312+ bytes each)
```

Each device entry contains:
- Path (256 bytes, null-terminated)
- Bus ID (32 bytes, null-terminated)
- Bus/Device numbers (4 bytes each)
- Speed (4 bytes)
- Vendor/Product IDs (2 bytes each)
- Device class/subclass/protocol
- Configuration and interface info

### Import Request

```
Offset  Size  Field
------  ----  -----
0       2     Version (0x0111)
2       2     Command (0x8003)
4       4     Status (0x00000000)
8       32    Bus ID (null-terminated)
```

### Import Reply

```
Offset  Size  Field
------  ----  -----
0       2     Version (0x0111)
2       2     Command (0x0003)
4       4     Status
8       312   Device info (same format as device list entry)
```

## Error Handling

The library handles several error conditions:

- **Connection failures** - Network timeouts, refused connections
- **Protocol errors** - Invalid version, unexpected commands
- **Device not found** - Requested vendor:product not available
- **Attachment failures** - Device already in use, permission denied

All errors are wrapped with context for debugging.

## Platform Support

Currently supports:
- **Linux** - Native USB/IP kernel module support
- **macOS** - Requires USB/IP server on workstation (custom implementation)
- **Windows** - Requires UsbDk driver and USB/IP server

## Future Enhancements

- [ ] TLS encryption for USB/IP traffic
- [ ] Connection keepalive and reconnection logic
- [ ] Metrics collection (bytes transferred, latency)
- [ ] Support for USB/IP protocol version negotiation
- [ ] Batch device operations
- [ ] Device filtering by capabilities
