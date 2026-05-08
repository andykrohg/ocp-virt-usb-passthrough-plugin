//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// enumerateUSBDevices returns a list of USB devices on Windows
func enumerateUSBDevices() ([]USBDevice, error) {
	// Use PowerShell Get-PnpDevice to enumerate USB devices
	// This is more reliable than wmic and works on modern Windows
	cmd := exec.Command("powershell", "-Command",
		`Get-PnpDevice -Class USB | Where-Object {$_.Status -eq 'OK'} | Select-Object FriendlyName, DeviceID, InstanceId | ConvertTo-Json`)

	output, err := cmd.Output()
	if err != nil {
		// Fallback to wmic if PowerShell fails
		return enumerateUSBDevicesWMIC()
	}

	return parseWindowsUSBPowerShell(string(output))
}

// enumerateUSBDevicesWMIC is a fallback using wmic
func enumerateUSBDevicesWMIC() ([]USBDevice, error) {
	cmd := exec.Command("wmic", "path", "Win32_PnPEntity",
		"where", "DeviceID like 'USB%'",
		"get", "Caption,DeviceID", "/format:csv")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate USB devices: %w", err)
	}

	return parseWindowsUSBWMIC(string(output))
}

type windowsUSBDevice struct {
	FriendlyName string `json:"FriendlyName"`
	DeviceID     string `json:"DeviceID"`
	InstanceId   string `json:"InstanceId"`
}

func parseWindowsUSBPowerShell(output string) ([]USBDevice, error) {
	// PowerShell returns JSON, but we'll use regex to extract key info
	// DeviceID format: USB\VID_090C&PID_1000\0375420080001253

	var devices []USBDevice

	// Extract USB device IDs using regex
	vidPidRegex := regexp.MustCompile(`VID_([0-9A-F]{4})&PID_([0-9A-F]{4})`)
	nameRegex := regexp.MustCompile(`"FriendlyName":\s*"([^"]+)"`)
	instanceRegex := regexp.MustCompile(`"InstanceId":\s*"([^"]+)"`)

	// Split by device entries (JSON objects)
	entries := strings.Split(output, "}")

	for _, entry := range entries {
		if !strings.Contains(entry, "FriendlyName") {
			continue
		}

		// Extract friendly name
		nameMatches := nameRegex.FindStringSubmatch(entry)
		if len(nameMatches) < 2 {
			continue
		}
		name := nameMatches[1]

		// Extract instance ID (contains vendor/product)
		instanceMatches := instanceRegex.FindStringSubmatch(entry)
		if len(instanceMatches) < 2 {
			continue
		}
		instanceID := instanceMatches[1]

		// Extract VID and PID
		vidPidMatches := vidPidRegex.FindStringSubmatch(instanceID)
		if len(vidPidMatches) < 3 {
			continue
		}

		vendorID := strings.ToLower(vidPidMatches[1])
		productID := strings.ToLower(vidPidMatches[2])
		vendorProduct := fmt.Sprintf("%s:%s", vendorID, productID)

		// Extract serial number (last part of instance ID)
		parts := strings.Split(instanceID, "\\")
		serial := ""
		if len(parts) > 2 {
			serial = parts[2]
		}

		// Generate bus ID (simplified - Windows doesn't have direct bus/device numbers)
		busID := fmt.Sprintf("usb-%s", serial)

		device := USBDevice{
			Name:          name,
			VendorProduct: vendorProduct,
			Vendor:        extractVendor(name),
			Product:       name,
			Serial:        serial,
			BusID:         busID,
			IsCAC:         detectCACReader(vendorID),
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func parseWindowsUSBWMIC(output string) ([]USBDevice, error) {
	var devices []USBDevice

	lines := strings.Split(output, "\n")
	vidPidRegex := regexp.MustCompile(`VID_([0-9A-F]{4})&PID_([0-9A-F]{4})`)

	for _, line := range lines {
		if !strings.Contains(line, "USB") {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			continue
		}

		name := strings.TrimSpace(fields[1])
		deviceID := strings.TrimSpace(fields[2])

		// Extract VID and PID
		matches := vidPidRegex.FindStringSubmatch(deviceID)
		if len(matches) < 3 {
			continue
		}

		vendorID := strings.ToLower(matches[1])
		productID := strings.ToLower(matches[2])
		vendorProduct := fmt.Sprintf("%s:%s", vendorID, productID)

		device := USBDevice{
			Name:          name,
			VendorProduct: vendorProduct,
			Vendor:        extractVendor(name),
			Product:       name,
			Serial:        "",
			BusID:         "usb-unknown",
			IsCAC:         detectCACReader(vendorID),
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// extractVendor tries to extract vendor name from device friendly name
func extractVendor(name string) string {
	// Common patterns: "Vendor Product", "Vendor - Product", etc.
	parts := strings.Fields(name)
	if len(parts) > 0 {
		return parts[0]
	}
	return "Unknown"
}

// detectCACReader checks if a device is a CAC card reader based on vendor ID
func detectCACReader(vendorID string) bool {
	vendorID = strings.ToLower(strings.TrimSpace(vendorID))

	knownCACVendors := []string{
		"0529", // Aladdin Knowledge Systems (eToken)
		"04e6", // SCM Microsystems
		"0403", // FTDI (some CAC readers)
		"076b", // OmniKey CardMan
		"058f", // Alcor Micro
		"072f", // Advanced Card Systems
		"04e8", // Samsung (some models)
		"0b97", // O2 Micro
	}

	for _, known := range knownCACVendors {
		if vendorID == known {
			return true
		}
	}

	return false
}
