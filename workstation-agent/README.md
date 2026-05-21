# USB Workstation Agent

Lightweight HTTP API server that exposes local USB devices to the OpenShift Console plugin.

## Features

- 🔌 Enumerates local USB devices
- 📡 HTTP API for console plugin integration
- 🚀 Executes virtctl usbredir for USB passthrough
- 🔐 Auto-elevation for USB device access
- 🔒 Identifies CAC card readers automatically
- 🖥️ Cross-platform (macOS, Linux, Windows)

## Installation

### Download Pre-built Binary

Visit the [releases page](https://github.com/andykrohg/ocp-virt-usb-passthrough-plugin/releases) and download the binary for your platform:

**macOS (Apple Silicon):**
```bash
curl -LO https://github.com/andykrohg/ocp-virt-usb-passthrough-plugin/releases/latest/download/usb-agent-darwin-arm64
chmod +x usb-agent-darwin-arm64
sudo mv usb-agent-darwin-arm64 /usr/local/bin/usb-agent
```

**macOS (Intel):**
```bash
curl -LO https://github.com/andykrohg/ocp-virt-usb-passthrough-plugin/releases/latest/download/usb-agent-darwin-amd64
chmod +x usb-agent-darwin-amd64
sudo mv usb-agent-darwin-amd64 /usr/local/bin/usb-agent
```

**Linux:**
```bash
curl -LO https://github.com/andykrohg/ocp-virt-usb-passthrough-plugin/releases/latest/download/usb-agent-linux-amd64
chmod +x usb-agent-linux-amd64
sudo mv usb-agent-linux-amd64 /usr/local/bin/usb-agent
```

**Windows:**
```powershell
# Download usb-agent-windows-amd64.exe from releases
# Move to C:\Program Files\usb-agent\usb-agent.exe
```

### Build from Source

```bash
# Clone repo
git clone https://github.com/andykrohg/ocp-virt-usb-passthrough-plugin
cd ocp-virt-usb-passthrough-plugin/workstation-agent

# Build
go build -o usb-agent

# Install
sudo mv usb-agent /usr/local/bin/
```

## Prerequisites

### All Platforms

**Install virtctl** (required for USB redirection):
- Download from [OpenShift Virtualization CLI Tools](https://docs.openshift.com/container-platform/latest/virt/virt-using-the-cli-tools.html)
- Or install via Homebrew: `brew install virtctl` (macOS)

**Valid kubeconfig** with access to your OpenShift cluster.

### Elevated Privileges Required

**All platforms require Administrator/root privileges** for USB device passthrough. The agent will automatically request elevation on startup.

**Why are elevated privileges needed?**

USB passthrough requires low-level access to the USB subsystem:
- **Unbind devices** from their current OS drivers (e.g., HID for keyboards, mass storage for USB drives)
- **Bind devices** to USB/IP redirection drivers that tunnel USB traffic over the network
- **Capture raw USB communication** to forward to remote VMs

Operating systems protect these operations to prevent malicious applications from:
- Hijacking security devices (CAC readers, security keys)
- Capturing keyboard/mouse input
- Accessing storage devices without permission

**Platform-Specific Behavior:**

**Linux:**
- Auto-elevates with `sudo` (prompts for password)
- Requires `virtctl` to be in PATH

**macOS:**
- Auto-elevates with `sudo` (prompts for password)
- May need to grant Terminal.app Full Disk Access in System Settings for full USB enumeration

**Windows:**
- Auto-elevates with UAC prompt (User Account Control)
- May need to allow through Windows Firewall for console plugin access

## Usage

### Basic Usage

```bash
# Run agent (will auto-elevate with sudo if needed)
./usb-agent --kubeconfig ~/.kube/config

# Specify custom port
./usb-agent --kubeconfig ~/.kube/config --port 8080
```

### Configuration Options

- `--kubeconfig` - Path to kubeconfig (default: $KUBECONFIG or ~/.kube/config)
- `--port` - HTTP API port (default: 8080)
- `--owner` - Owner name for device identification (default: $USER)

### HTTP API Endpoints

The agent exposes these endpoints on `localhost:8080`:

- `GET /devices` - List local USB devices
- `GET /connections` - List active USB passthrough connections
- `POST /attach` - Attach device to VM (starts virtctl usbredir)
- `DELETE /detach/{id}` - Detach device (stops virtctl)

## How It Works

1. **Enumerates USB devices** using system tools:
   - macOS: `system_profiler SPUSBDataType`
   - Linux: `lsusb`
   - Windows: PowerShell `Get-PnpDevice`

2. **Exposes HTTP API** on localhost:8080 with CORS enabled for console access

3. **Executes virtctl usbredir** when attach is requested:
   ```bash
   virtctl usbredir <vendor>:<product> <vm-name> -n <namespace>
   ```

4. **Manages processes** - tracks virtctl PIDs and handles cleanup

## Example Output

```
╔════════════════════════════════════════════════════════╗
║    USB Workstation Agent - Running                     ║
╚════════════════════════════════════════════════════════╝

  API Port: 8080
  Owner: akrohg
  Platform: darwin

  Endpoints:
    GET  http://localhost:8080/devices
    GET  http://localhost:8080/connections
    POST http://localhost:8080/attach
    DEL  http://localhost:8080/detach/{id}

  Status: 🟢 Running (elevated)
  Press Ctrl+C to stop
```

## Testing

Test the API directly:

```bash
# List USB devices
curl http://localhost:8080/devices | jq

# List active connections
curl http://localhost:8080/connections | jq

# Attach device (example)
curl -X POST http://localhost:8080/attach \
  -H "Content-Type: application/json" \
  -d '{
    "deviceId": "04e6:5816",
    "deviceName": "SCM CAC Reader",
    "vmName": "windows-vm",
    "namespace": "default"
  }'
```

## Troubleshooting

**Agent won't start:**
- Check if `virtctl` is installed: `which virtctl`
- Ensure you have a valid kubeconfig: `oc get nodes`
- Verify port 8080 isn't in use: `lsof -i :8080` (macOS/Linux)

**No devices showing:**
- Run enumeration command manually:
  - macOS: `system_profiler SPUSBDataType`
  - Linux: `lsusb`
  - Windows: `Get-PnpDevice -Class USB`
- Ensure USB devices are connected
- Check agent logs for parsing errors

**virtctl fails:**
- Verify VM is running (not stopped)
- Check VM has `clientPassthrough: {}` enabled in spec
- Ensure kubeconfig has permissions to access VMs
- **Agent must run with elevated privileges** - virtctl cannot access USB devices without root/Administrator

**Permission denied errors:**
- If auto-elevation fails, manually run with `sudo` on macOS/Linux or "Run as Administrator" on Windows
- The agent checks elevation on startup and will attempt to relaunch itself with privileges
- macOS: Grant Terminal.app Full Disk Access in System Settings for full USB device enumeration

**"Failed to open device" from virtctl:**
- This means the agent is not running with sufficient privileges
- USB passthrough requires root/Administrator to unbind devices from OS drivers
- Restart the agent and approve the sudo/UAC prompt

## Building from Source

```bash
# Build for current platform
go build -o usb-agent

# Cross-compile for all platforms
GOOS=darwin GOARCH=arm64 go build -o usb-agent-darwin-arm64
GOOS=darwin GOARCH=amd64 go build -o usb-agent-darwin-amd64
GOOS=linux GOARCH=amd64 go build -o usb-agent-linux-amd64
GOOS=windows GOARCH=amd64 go build -o usb-agent-windows-amd64.exe
```

## License

MIT
