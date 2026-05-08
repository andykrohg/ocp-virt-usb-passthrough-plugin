# USB Workstation Agent

Lightweight agent that runs on user workstations to expose USB devices via USB/IP and register them with the cluster.

## Features

- 🔌 Runs USB/IP server to expose local USB devices
- 📡 Registers available devices with OpenShift cluster
- 💚 Heartbeat mechanism to keep device list current
- 🪶 Lightweight (~5MB binary, minimal resource usage)
- 🖥️ System tray integration (macOS/Windows/Linux)
- 🔒 Identifies CAC card readers automatically

## Installation

### From Binary

```bash
# Download latest release
curl -LO https://github.com/openshift/usb-workstation-agent/releases/latest/download/usb-agent-darwin-arm64

# Make executable
chmod +x usb-agent-darwin-arm64

# Move to PATH
sudo mv usb-agent-darwin-arm64 /usr/local/bin/usb-agent
```

### From Source

```bash
# Clone repo
git clone https://github.com/openshift/usb-workstation-agent
cd usb-workstation-agent

# Build
go build -o usb-agent main.go

# Install
sudo mv usb-agent /usr/local/bin/
```

## Prerequisites

### Linux

Install USB/IP tools:

```bash
# Ubuntu/Debian
sudo apt install usbip

# RHEL/CentOS/Fedora  
sudo yum install usbip-utils

# Arch
sudo pacman -S usbip
```

Load kernel modules:

```bash
sudo modprobe usbip-core
sudo modprobe usbip-host
```

### Windows

1. **Install UsbDk** (required driver):
   - Download from https://www.spice-space.org/download/windows/usbdk/
   - Run installer (requires admin privileges)
   - Reboot after installation

2. **Install USB/IP for Windows**:
   - Download from https://github.com/cezanne/usbip-win/releases
   - Extract `usbipd.exe` and `usbip.exe` to `C:\Program Files\usbip` (or add to PATH)

### macOS

```bash
# Install usbip (via Homebrew if available)
brew install usbip
```

**Note**: macOS USB/IP support is experimental. For production, use Linux or Windows workstations.

## Usage

### Basic Usage

```bash
# Run with default settings
usb-agent

# Specify kubeconfig
usb-agent --kubeconfig ~/.kube/config

# Custom port
usb-agent --port 3240

# Specify namespace for device registration
usb-agent --namespace usb-devices --owner alice
```

### Configuration

The agent accepts the following flags:

- `--kubeconfig` - Path to kubeconfig (default: $KUBECONFIG or ~/.kube/config)
- `--port` - USB/IP server port (default: 3240)
- `--namespace` - Namespace for USBDevice resources (default: default)
- `--owner` - Owner name for device registration (default: $USER)
- `--cluster` - Cluster address to advertise (auto-detected)

### Auto-Start on Login

#### macOS (LaunchAgent)

Create `~/Library/LaunchAgents/com.openshift.usb-agent.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.openshift.usb-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/usb-agent</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load it:
```bash
launchctl load ~/Library/LaunchAgents/com.openshift.usb-agent.plist
```

#### Linux (systemd)

Create `/etc/systemd/system/usb-agent.service`:

```ini
[Unit]
Description=USB Workstation Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/usb-agent
Restart=always
User=%u

[Install]
WantedBy=default.target
```

Enable it:
```bash
systemctl --user enable usb-agent
systemctl --user start usb-agent
```

## How It Works

1. **Starts USB/IP server** on port 3240 (configurable)
2. **Enumerates USB devices** using system tools:
   - macOS: `system_profiler SPUSBHostDataType`
   - Linux: `lsusb`
   - Windows: WMI queries
3. **Registers devices** as `USBDevice` CRs in the cluster
4. **Sends heartbeats** every 30 seconds to update device status
5. **Shows system tray icon** with connection status

## Example Output

```
╔════════════════════════════════════════════════════════╗
║    USB Workstation Agent - Running                     ║
╚════════════════════════════════════════════════════════╝

  Port: 3240
  Owner: alice
  Devices: 3

  1. Samsung USB Drive (090c:1000)
  2. CAC Card Reader (0529:0620)
     🔒 CAC Reader
  3. Apple Keyboard (05ac:024f)

  Status: 🟢 Running
  Press Ctrl+C to stop
```

## Device Registration

When running, the agent creates `USBDevice` resources in the cluster:

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
  lastSeen: "2026-05-08T10:30:00Z"
```

These resources are visible in the console plugin for users to select.

## Security

- Agent requires **root/admin privileges** to access USB hardware
- Recommend running with limited kubeconfig (read-only for devices)
- USB/IP traffic should be encrypted (TLS wrapper recommended)
- Firewall rules should restrict USB/IP port to cluster nodes only

## Troubleshooting

**Agent won't start:**
- Check if `usbip` is installed: `which usbipd`
- Ensure you have admin/root privileges
- Check port 3240 isn't already in use: `lsof -i :3240`

**No devices showing:**
- Run `lsusb` (Linux) or `system_profiler SPUSBDataType` (macOS) manually
- Check USB devices are properly connected
- Restart agent with verbose logging

**Can't connect to cluster:**
- Verify kubeconfig is valid: `kubectl get nodes`
- Check network connectivity to cluster
- Ensure RBAC permissions for creating USBDevice resources

## Building from Source

```bash
# Build for current platform
go build -o usb-agent main.go

# Cross-compile for all platforms
GOOS=darwin GOARCH=arm64 go build -o usb-agent-darwin-arm64
GOOS=darwin GOARCH=amd64 go build -o usb-agent-darwin-amd64
GOOS=linux GOARCH=amd64 go build -o usb-agent-linux-amd64
GOOS=windows GOARCH=amd64 go build -o usb-agent-windows-amd64.exe
```

## License

MIT
