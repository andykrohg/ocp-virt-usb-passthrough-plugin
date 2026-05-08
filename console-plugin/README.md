# USB Passthrough Console Plugin

OpenShift Console Dynamic Plugin for managing USB device passthrough to VMs.

## Features

- 📱 Browser-based UI integrated into OpenShift Console
- 🔌 View available USB devices from workstations
- 🖥️ Select target VMs for USB passthrough
- ⚡ One-click USB connection creation
- 📊 Monitor active connections
- 🎨 PatternFly-based UI matching OpenShift design

## Screenshots

### Main Page
![USB Passthrough main view showing available devices and VMs]

### Connection Wizard
![Wizard dialog for creating USB connection]

## Installation

### Prerequisites

- OpenShift 4.12+ cluster
- Console operator with dynamic plugin support enabled
- USB Listener Operator deployed

### Deploy to Cluster

**Quick Deploy:**

```bash
# Build and push (updates image name in manifests/deployment.yaml first)
./build.sh
podman push quay.io/akrohg/usb-passthrough-plugin:latest

# Deploy to cluster
kubectl apply -f manifests/deployment.yaml

# Enable the plugin in OpenShift Console
kubectl patch consoles.operator.openshift.io cluster \
  --type json \
  -p '[{"op": "add", "path": "/spec/plugins/-", "value": "usb-passthrough-plugin"}]'

# Verify deployment
kubectl get pods -n usb-passthrough-plugin
kubectl get consoleplugin usb-passthrough-plugin
```

**Manual Build:**

```bash
# Install dependencies and build
npm install
npm run build

# Build and push container image
podman build --platform linux/amd64 -t quay.io/akrohg/usb-passthrough-plugin:latest .
podman push quay.io/akrohg/usb-passthrough-plugin:latest
```

### Enable Plugin

The plugin is automatically registered with the console when deployed. To manually enable:

```bash
oc patch consoles.operator.openshift.io cluster \
  --type json \
  --patch '[{"op": "add", "path": "/spec/plugins/-", "value": "usb-passthrough-plugin"}]'
```

## Development

### Local Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# In another terminal, run console with plugin
oc console --plugins usb-passthrough-plugin=http://localhost:9001
```

Navigate to http://localhost:9000 to see the console with your plugin.

### Project Structure

```
console-plugin/
├── src/
│   ├── index.ts                    # Plugin entry point
│   └── components/
│       ├── USBPassthroughPage.tsx  # Main page component
│       ├── USBDeviceList.tsx       # USB device list
│       ├── VMSelector.tsx          # VM selection dropdown
│       └── USBConnectionWizard.tsx # Connection creation wizard
├── console-extensions.json         # Plugin extension definitions
├── package.json
├── tsconfig.json
└── webpack.config.js
```

## Console Extensions

This plugin provides:

### VM Details Tab

- **VMUSBTab**: Adds a "USB Devices" tab to VirtualMachineInstance detail pages
  - Shows currently connected USB devices for the VM
  - Lists available USB devices from workstations
  - Allows attaching/detaching devices with one click
  - Displays connection status in real-time

## API Integration

The plugin watches these Kubernetes resources:

### USBDevice (usb.openshift.io/v1alpha1)

Lists available USB devices advertised by workstation agents.

### USBConnection (usb.openshift.io/v1alpha1)

Manages active USB passthrough connections. The plugin creates these resources when users click "Connect".

### VirtualMachineInstance (kubevirt.io/v1)

Lists running VMs available for USB passthrough.

## User Workflow

1. User navigates to **Virtualization → VirtualMachines** and selects a running VM
2. User clicks on the **USB Devices** tab in the VM details page
3. Tab shows:
   - Currently connected USB devices (if any)
   - Available USB devices from workstations running the agent
4. User selects a USB device from the dropdown
5. User clicks "Attach Device"
6. Plugin creates USBConnection CR
7. Operator reconciles and establishes connection via USB/IP
8. Device appears in the "Connected USB Devices" section with status
9. User can detach by clicking "Detach" button next to the connected device

## Styling

The plugin uses:
- **PatternFly 5** components for UI consistency
- **OpenShift Console design tokens** for colors and spacing
- Responsive grid layout for device/VM selection

## Testing

```bash
# Type checking
npm run typecheck

# Linting
npm run lint

# Unit tests (when added)
npm test
```

## Building for Production

```bash
# Production build
npm run build

# Output in dist/
```

## Troubleshooting

**Plugin not appearing in console:**
- Check plugin is enabled: `oc get consoles.operator.openshift.io cluster -o yaml`
- Verify plugin pod is running: `oc get pods -n openshift-console`
- Check console pod logs: `oc logs -n openshift-console <console-pod>`

**No USB devices showing:**
- Ensure workstation agent is running and registered devices
- Check USBDevice CRs exist: `oc get usbdevice`
- Verify RBAC permissions for console ServiceAccount

**VMs not appearing:**
- Check VirtualMachineInstances exist: `oc get vmi`
- Ensure VMs are in "Running" state
- Verify kubevirt is installed and working

## Future Enhancements

- [ ] Real-time connection status updates
- [ ] Connection health monitoring
- [ ] Multi-device selection
- [ ] Connection history/logs
- [ ] Device filtering and search
- [ ] VM favorites/recent connections
- [ ] Connection templates/profiles

## License

MIT
