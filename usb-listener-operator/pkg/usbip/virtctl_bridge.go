package usbip

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// VirtctlBridge provides integration between USB/IP and virtctl
type VirtctlBridge struct {
	virtctlPath string
}

// NewVirtctlBridge creates a new virtctl bridge
func NewVirtctlBridge(virtctlPath string) *VirtctlBridge {
	if virtctlPath == "" {
		virtctlPath = "virtctl" // Use PATH
	}
	return &VirtctlBridge{
		virtctlPath: virtctlPath,
	}
}

// AttachUSBDevice attaches a USB device to a VM using virtctl
// This is the high-level method that combines USB/IP client with virtctl
func (v *VirtctlBridge) AttachUSBDevice(
	ctx context.Context,
	workstationAddress string,
	vendorProduct string,
	vmName string,
	namespace string,
) error {
	// Method 1: Use virtctl usbredir directly
	// virtctl can connect to USB/IP servers and redirect to VMs
	// Format: virtctl usbredir <vendor>:<product> <vm-name> -n <namespace>

	cmd := exec.CommandContext(
		ctx,
		v.virtctlPath,
		"usbredir",
		vendorProduct,
		vmName,
		"-n",
		namespace,
	)

	// Set environment variable for USB/IP server address
	// Some virtctl versions support USBIP_SERVER env var
	cmd.Env = append(cmd.Env, fmt.Sprintf("USBIP_SERVER=%s", workstationAddress))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("virtctl usbredir failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// AttachUSBDeviceViaProxy attaches a USB device using our own USB/IP client as a proxy
// This method gives us more control over the USB/IP connection
func (v *VirtctlBridge) AttachUSBDeviceViaProxy(
	ctx context.Context,
	workstationAddress string,
	vendorProduct string,
	vmName string,
	namespace string,
) error {
	// Step 1: Connect to workstation's USB/IP server
	client := NewClient(workstationAddress)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to USB/IP server: %w", err)
	}
	defer client.Close()

	// Step 2: Find the device
	device, err := client.FindDevice(vendorProduct)
	if err != nil {
		return fmt.Errorf("failed to find device: %w", err)
	}

	// Step 3: Attach the device via USB/IP
	_, err = client.AttachDevice(device.BusID)
	if err != nil {
		return fmt.Errorf("failed to attach device: %w", err)
	}

	// Step 4: Use virtctl to redirect the now-attached device to the VM
	// At this point, the USB/IP connection is established
	// We need to keep the connection alive and bridge it to the VM

	// This is where we'd integrate more deeply with virtctl or QEMU
	// For now, we rely on virtctl's built-in USB/IP support

	return fmt.Errorf("proxy method not yet implemented - use AttachUSBDevice instead")
}

// DetachUSBDevice detaches a USB device from a VM
func (v *VirtctlBridge) DetachUSBDevice(
	ctx context.Context,
	vmName string,
	namespace string,
) error {
	// virtctl doesn't have a direct "detach" command
	// The device is detached when the usbredir process is terminated

	// In practice, we'd need to:
	// 1. Find the running usbredir process for this VM
	// 2. Terminate it gracefully

	// For now, return an error indicating manual intervention is needed
	return fmt.Errorf("detach must be done by terminating the virtctl usbredir process")
}

// CheckVirtctlVersion checks if virtctl is available and returns its version
func (v *VirtctlBridge) CheckVirtctlVersion() (string, error) {
	cmd := exec.Command(v.virtctlPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("virtctl not found or not working: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Platform-specific USB device handling

// GetUSBDevicePath returns the platform-specific USB device path
func GetUSBDevicePath(vendorProduct string) (string, error) {
	// This would be platform-specific
	// On Linux: /dev/bus/usb/001/002
	// On macOS: IOKit path
	// On Windows: Device instance path

	return "", fmt.Errorf("platform-specific device path lookup not implemented")
}
