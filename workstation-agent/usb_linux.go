//go:build linux

package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// enumerateUSBDevices returns a list of USB devices on Linux
func enumerateUSBDevices() ([]USBDevice, error) {
	// Use lsusb -v for detailed USB device information
	cmd := exec.Command("lsusb", "-v")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run lsusb: %w (make sure usbutils is installed)", err)
	}

	return parseLinuxUSB(string(output))
}

func parseLinuxUSB(output string) ([]USBDevice, error) {
	var devices []USBDevice

	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regex patterns for parsing lsusb output
	busDeviceRegex := regexp.MustCompile(`Bus (\d+) Device (\d+): ID ([0-9a-f]{4}):([0-9a-f]{4}) (.+)`)
	manufacturerRegex := regexp.MustCompile(`^\s+iManufacturer\s+\d+\s+(.+)$`)
	productRegex := regexp.MustCompile(`^\s+iProduct\s+\d+\s+(.+)$`)
	serialRegex := regexp.MustCompile(`^\s+iSerial\s+\d+\s+(.+)$`)

	var currentDevice *USBDevice

	for scanner.Scan() {
		line := scanner.Text()

		// Match main device line
		if matches := busDeviceRegex.FindStringSubmatch(line); matches != nil {
			// Save previous device if exists
			if currentDevice != nil {
				devices = append(devices, *currentDevice)
			}

			busNum := matches[1]
			devNum := matches[2]
			vendorID := matches[3]
			productID := matches[4]
			description := strings.TrimSpace(matches[5])

			currentDevice = &USBDevice{
				Name:          description,
				VendorProduct: fmt.Sprintf("%s:%s", vendorID, productID),
				Vendor:        "",
				Product:       description,
				Serial:        "",
				BusID:         fmt.Sprintf("%s-%s", busNum, devNum),
				IsCAC:         detectCACReader(vendorID),
			}
			continue
		}

		// Parse detail lines for current device
		if currentDevice != nil {
			if matches := manufacturerRegex.FindStringSubmatch(line); matches != nil {
				currentDevice.Vendor = strings.TrimSpace(matches[1])
			} else if matches := productRegex.FindStringSubmatch(line); matches != nil {
				currentDevice.Product = strings.TrimSpace(matches[1])
			} else if matches := serialRegex.FindStringSubmatch(line); matches != nil {
				currentDevice.Serial = strings.TrimSpace(matches[1])
			}
		}
	}

	// Add last device
	if currentDevice != nil {
		devices = append(devices, *currentDevice)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing lsusb output: %w", err)
	}

	return devices, nil
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
