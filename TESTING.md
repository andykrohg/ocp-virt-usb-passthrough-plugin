# Testing Guide

This guide walks through testing the USB passthrough system end-to-end.

## Prerequisites

- OpenShift cluster (4.12+) with OpenShift Virtualization installed
- At least one Windows VM with `clientPassthrough: {}` enabled in the spec
- Local workstation with USB devices
- Network connectivity from workstation to cluster
- `oc` or `oc` CLI configured

## Step 1: Enable USB Passthrough on VM

Edit your VirtualMachine to add `clientPassthrough`:

```bash
oc edit vm <vm-name> -n <namespace>
```

Add under `spec.template.spec.domain.devices`:

```yaml
spec:
  template:
    spec:
      domain:
        devices:
          clientPassthrough: {}  # Add this line
          disks:
            - name: root
              disk: {}
```

Restart the VM for changes to take effect:

```bash
oc delete vmi <vm-name> -n <namespace>
# VM controller will recreate it
```

## Step 2: Install CRDs and Operator

```bash
cd usb-listener-operator

# Install CRDs and RBAC
./deploy.sh
```

This creates:
- `usb-passthrough-system` namespace
- USBDevice and USBConnection CRDs
- ServiceAccount and RBAC permissions

## Step 3: Build and Deploy Operator

```bash
# Build operator image
podman build -t quay.io/yourorg/usb-listener-operator:latest .

# Push to registry
podman push quay.io/yourorg/usb-listener-operator:latest

# Update deployment with your image
# Edit config/manager/deployment.yaml and change image URL

# Deploy operator
oc apply -f config/manager/deployment.yaml

# Verify operator is running
oc get pods -n usb-passthrough-system
```

Expected output:
```
NAME                                     READY   STATUS    RESTARTS   AGE
usb-listener-operator-6b8f9d5c7b-xyz12   1/1     Running   0          30s
```

## Step 4: Install USB/IP Tools on Workstation

### Linux

```bash
# Ubuntu/Debian
sudo apt install usbip

# RHEL/Fedora
sudo yum install usbip-utils

# Load kernel modules
sudo modprobe usbip-core
sudo modprobe usbip-host
```

### macOS

```bash
# If available via Homebrew
brew install usbip
```

### Windows

1. Download and install UsbDk from https://www.spice-space.org/download/windows/usbdk/
2. Download USB/IP for Windows from https://github.com/cezanne/usbip-win/releases
3. Extract `usbipd.exe` to a directory in PATH

## Step 5: Run Workstation Agent

```bash
cd workstation-agent

# Build agent
go build -o usb-agent

# Run agent (requires sudo/admin for USB access)
sudo ./usb-agent -kubeconfig ~/.kube/config -owner $(whoami)
```

Expected output:
```
Starting USB/IP server on port 3240...
Found usbipd at: /usr/sbin/usbipd
Starting USB/IP server (requires elevated privileges)...
USB/IP server listening on port 3240
Found 3 USB devices
  - CAC Card Reader (076b:2303)
  - Samsung USB Drive (090c:1000)
  - Logitech Keyboard (046d:c52b)
Connecting to cluster...
Registered 3 devices with cluster

╔════════════════════════════════════════════════════════╗
║    USB Workstation Agent - Running                     ║
╚════════════════════════════════════════════════════════╝

  Port: 3240
  Owner: yourname
  Devices: 3

  1. CAC Card Reader (076b:2303)
     🔒 CAC Reader
  2. Samsung USB Drive (090c:1000)
  3. Logitech Keyboard (046d:c52b)

  Status: 🟢 Running
  Press Ctrl+C to stop
```

## Step 6: Verify USBDevice CRs

```bash
# List registered USB devices
oc get usbdevices

# Detailed view
oc get usbdevices -o yaml
```

Expected output:
```
NAME                      DEVICE ID     DEVICE NAME         OWNER     AVAILABLE   CONNECTED TO
usb-yourname-076b-2303    076b:2303     CAC Card Reader     yourname  true        
usb-yourname-090c-1000    090c:1000     Samsung USB Drive   yourname  true        
usb-yourname-046d-c52b    046d:c52b     Logitech Keyboard   yourname  true        
```

## Step 7: Deploy Console Plugin (Optional)

If you want to test the UI:

```bash
cd console-plugin

# Install dependencies
npm install

# Build
npm run build

# Build and push plugin image
podman build -t quay.io/yourorg/usb-passthrough-plugin:latest .
podman push quay.io/yourorg/usb-passthrough-plugin:latest

# Deploy plugin (create deployment manifests)
# oc apply -f manifests/
```

## Step 8: Test USB Connection via CLI

Create a USBConnection manually:

```bash
cat <<EOF | oc apply -f -
apiVersion: usb.openshift.io/v1alpha1
kind: USBConnection
metadata:
  name: test-usb-connection
  namespace: default
spec:
  workstationAddress: "YOUR_WORKSTATION_IP:3240"
  deviceID: "090c:1000"
  deviceName: "Samsung USB Drive"
  vmName: "windows-vm"
  namespace: "default"
EOF
```

Replace:
- `YOUR_WORKSTATION_IP` with your workstation's IP (shown in agent output)
- `deviceID` with actual device ID from `oc get usbdevices`
- `vmName` and `namespace` with your VM details

## Step 9: Monitor Connection

```bash
# Watch USBConnection status
oc get usbconnection test-usb-connection -o yaml -w

# Check operator logs
oc logs -n usb-passthrough-system -l app=usb-listener-operator -f
```

Expected status progression:
```
status:
  phase: Pending
  message: "Initiating USB connection"
```

Then:
```
status:
  phase: Connecting
  message: "Connecting to USB/IP server..."
```

Finally:
```
status:
  phase: Connected
  message: "USB device 090c:1000 connected to VM default/windows-vm"
  connectedAt: "2026-05-08T15:30:00Z"
```

## Step 10: Verify Device in VM

1. Connect to the VM console (via OpenShift Console or RDP)
2. On Windows: Open Device Manager
3. Check for the USB device under "Disk drives" or "USB controllers"
4. Device should show as connected and usable

## Step 11: Test Detachment

```bash
# Delete the USBConnection
oc delete usbconnection test-usb-connection

# Verify device is removed from VM
# Check Device Manager in Windows - device should disappear
```

## Step 12: Test via Console Plugin

If plugin is deployed:

1. Navigate to **Virtualization → VirtualMachines**
2. Click on your VM
3. Go to **USB Devices** tab
4. You should see:
   - Available USB devices from your workstation
   - Dropdown to select a device
   - "Attach Device" button
5. Select a device and click "Attach Device"
6. Device should appear in "Connected USB Devices" section
7. Click "Detach" to remove

## Troubleshooting

### Agent: "usbipd not found"

Install USB/IP tools (see Step 4).

### Agent: "Failed to start usbipd: permission denied"

Run agent with `sudo` (Linux/macOS) or as Administrator (Windows).

### No devices showing in `oc get usbdevices`

Check agent logs:
- Verify agent is running
- Check network connectivity to cluster
- Verify RBAC permissions: `oc auth can-i create usbdevices.usb.openshift.io`

### USBConnection stuck in "Connecting"

Check operator logs:
```bash
oc logs -n usb-passthrough-system -l app=usb-listener-operator
```

Common issues:
- Workstation firewall blocking port 3240
- USB/IP server not running on workstation
- Incorrect workstation address in spec

### Device not showing in VM

1. Verify VM has `clientPassthrough: {}` enabled
2. Check VM was restarted after adding clientPassthrough
3. Verify USBConnection status is "Connected"
4. Check virtctl is installed in operator pod
5. Look for errors in operator logs

### "Failed to connect to USB/IP server"

Ensure:
- Workstation agent is running
- Firewall allows TCP connections to port 3240
- Workstation IP is correct and reachable from cluster

Test connectivity from operator pod:
```bash
POD=$(oc get pods -n usb-passthrough-system -l app=usb-listener-operator -o name | head -1)
oc exec -n usb-passthrough-system $POD -- nc -zv YOUR_WORKSTATION_IP 3240
```

## Clean Up

```bash
# Delete test connection
oc delete usbconnection --all

# Delete USB devices
oc delete usbdevices --all

# Stop workstation agent (Ctrl+C)

# Remove operator
oc delete -f config/manager/deployment.yaml
oc delete -f config/rbac/rbac.yaml

# Remove CRDs (this deletes all USBDevice and USBConnection resources)
oc delete -f config/crd/
```
