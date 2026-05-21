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

### Platform-Specific Notes

**Linux:**
- No additional requirements

**macOS:**
- USB device access requires sudo/root privileges
- Agent will automatically request elevation on startup

**Windows:**
- Run as Administrator
- May need to allow through Windows Firewall

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
- Ensure you have a valid kubeconfig: `kubectl get nodes`
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
- Agent must run with elevated privileges (sudo)

**Permission denied errors:**
- Run agent with `sudo` (it will auto-elevate but you may need to start with sudo initially)
- macOS: Grant Terminal.app Full Disk Access in System Settings

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
