# USB Passthrough Console Plugin

OpenShift Console Dynamic Plugin for managing USB device passthrough to VMs.

## Features

- 📱 Browser-based UI integrated into OpenShift Console
- 🔌 View available USB devices from local workstation
- 🖥️ Attach/detach USB devices to/from running VMs
- ⚡ One-click USB connection
- 📊 Monitor active connections
- 🎨 PatternFly-based UI matching OpenShift design
- 🔒 CAC reader detection and highlighting

## Installation

### Prerequisites

- OpenShift 4.12+ cluster with OpenShift Virtualization
- Console operator with dynamic plugin support enabled
- Workstation agent running on your local machine

### Deploy to Cluster

```bash
# Build and push container image
cd console-plugin
# Deploy to cluster
oc apply -f manifests/deployment.yaml

# Enable the plugin in OpenShift Console
oc patch consoles.operator.openshift.io cluster \
  --type json \
  -p '[{"op": "add", "path": "/spec/plugins/-", "value": "usb-passthrough-plugin"}]'

# Verify deployment
oc get pods -n usb-passthrough-plugin
oc get consoleplugin usb-passthrough-plugin
```

The console will restart automatically to load the plugin.

## Development

### Local Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Output in dist/
```

### Project Structure

```
console-plugin/
├── src/
│   └── components/
│       └── VMUSBTab.tsx            # USB Devices tab component
├── console-extensions.json         # Plugin extension definitions
├── nginx.conf                      # nginx config for serving plugin
├── manifests/
│   └── deployment.yaml             # Kubernetes deployment
├── Containerfile                   # Container build definition
├── package.json
├── tsconfig.json
└── webpack.config.js
```

## Console Extension

### USB Devices Tab

Adds a "USB Devices" tab to VirtualMachine detail pages showing:

- **Connected USB Devices**: Currently attached devices with status
  - Device name and ID
  - Connection status (Connecting, Connected, Failed)
  - Detach button

- **Attach USB Device**: Interface for attaching new devices
  - Dropdown of available devices from workstation agent
  - CAC readers highlighted with 🔒 icon
  - Attach button (disabled when VM is not running)

- **VM Status Check**: Prevents attaching to stopped VMs with helpful warning

## API Integration

The plugin communicates with the **workstation agent** running on localhost:

- `GET http://localhost:8080/devices` - Fetch available USB devices
- `GET http://localhost:8080/connections` - Fetch active connections
- `POST http://localhost:8080/attach` - Attach device to VM
- `DELETE http://localhost:8080/detach/{id}` - Detach device from VM

Polls every 3 seconds for real-time updates.

## User Workflow

1. **Start workstation agent** on your local machine
2. **Navigate to Virtualization → VirtualMachines** in OpenShift Console
3. **Select a VM** (must be running)
4. **Click "USB Devices" tab**
5. **Select USB device** from dropdown (CAC readers shown with 🔒)
6. **Click "Attach Device"**
7. Device appears in "Connected USB Devices" section with status
8. **Click "Detach"** to disconnect when done

## Features

### CAC Reader Detection

CAC card readers are automatically detected and highlighted with 🔒 icon.

Known CAC reader vendor IDs:
- 0x0529 - Aladdin Knowledge Systems
- 0x04e6 - SCM Microsystems
- 0x0403 - FTDI
- 0x076b - OmniKey CardMan
- 0x058f - Alcor Micro

### VM Status Validation

Plugin prevents attaching USB devices to stopped VMs and displays:
- Current VM status
- Warning message
- Instructions to start the VM first

### Error Handling

- **Agent not running**: Shows installation instructions with GitHub link
- **Connection failures**: Displays error messages from agent/virtctl
- **No devices available**: Helpful troubleshooting tips

### Auto-refresh

Plugin automatically detects when the workstation agent starts - no page refresh needed.

## Styling

The plugin uses:
- **PatternFly 4** components for UI consistency
- **OpenShift Console SDK** for platform integration
- Responsive grid layout

## Testing

```bash
# Type checking
npm run typecheck

# Linting
npm run lint
```

## Troubleshooting

**Plugin not appearing in console:**
- Check plugin is enabled: `oc get consoles.operator.openshift.io cluster -o jsonpath='{.spec.plugins}'`
- Verify plugin pod is running: `oc get pods -n usb-passthrough-plugin`
- Check console pod logs: `oc logs -n openshift-console -l component=ui`

**No USB devices showing:**
- Ensure workstation agent is running on localhost:8080
- Check browser console for CORS errors (F12 → Console)
- Verify agent responds: `curl http://localhost:8080/devices`

**USB Devices tab missing:**
- Tab only appears on VirtualMachine resources
- Verify you're viewing a VM details page
- Check plugin extension loaded: look for "USB Devices" in nav tabs

**Attach fails:**
- Check VM is running (tab shows warning if stopped)
- Verify workstation agent has elevated privileges (sudo)
- Check agent logs for virtctl errors
- Ensure VM has `clientPassthrough: {}` enabled

## Building the Container Image

```bash
# Build plugin
npm run build

# Build container (ARM Mac)
podman build -t quay.io/youruser/usb-passthrough-plugin:latest -f Containerfile .

# Push to registry
podman push quay.io/youruser/usb-passthrough-plugin:latest
```

## License

MIT
