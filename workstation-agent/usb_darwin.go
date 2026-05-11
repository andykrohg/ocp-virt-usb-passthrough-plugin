//go:build darwin

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// enumerateUSBDevices returns a list of USB devices on macOS
func enumerateUSBDevices() ([]USBDevice, error) {
	// Try both new and old system_profiler data types
	dataTypes := []string{"SPUSBHostDataType", "SPUSBDataType"}

	for _, dataType := range dataTypes {
		cmd := exec.Command("system_profiler", dataType, "-json")
		output, err := cmd.Output()
		if err != nil {
			continue // Try next data type
		}

		devices, err := parseMacOSUSB(string(output), dataType)
		if err == nil {
			// Success - return devices (even if empty list)
			return devices, nil
		}
	}

	return nil, fmt.Errorf("failed to enumerate USB devices with system_profiler")
}

type macOSUSBData struct {
	SPUSBDataType     []macOSUSBController `json:"SPUSBDataType,omitempty"`
	SPUSBHostDataType []macOSUSBController `json:"SPUSBHostDataType,omitempty"`
}

type macOSUSBController struct {
	Name  string           `json:"_name"`
	Items []macOSUSBDevice `json:"_items,omitempty"`
}

type macOSUSBDevice struct {
	Name string `json:"_name"`

	// Old field names (SPUSBDataType)
	VendorID       string `json:"vendor_id,omitempty"`
	ProductID      string `json:"product_id,omitempty"`
	SerialNum      string `json:"serial_num,omitempty"`
	LocationID     string `json:"location_id,omitempty"`
	Manufacturer   string `json:"manufacturer,omitempty"`

	// New field names (SPUSBHostDataType) - macOS 26+
	USBVendorID    string `json:"USBDeviceKeyVendorID,omitempty"`
	USBProductID   string `json:"USBDeviceKeyProductID,omitempty"`
	USBSerialNum   string `json:"USBDeviceKeySerialNumber,omitempty"`
	USBLocationID  string `json:"USBDeviceKeyLocationID,omitempty"`
	USBManufacturer string `json:"USBDeviceKeyManufacturerString,omitempty"`

	Items []macOSUSBDevice `json:"_items,omitempty"`
}

func parseMacOSUSB(jsonOutput string, dataType string) ([]USBDevice, error) {
	var data macOSUSBData
	if err := json.Unmarshal([]byte(jsonOutput), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var controllers []macOSUSBController
	if dataType == "SPUSBHostDataType" {
		controllers = data.SPUSBHostDataType
	} else {
		controllers = data.SPUSBDataType
	}

	var devices []USBDevice
	for _, controller := range controllers {
		devices = append(devices, extractDevices(controller.Items)...)
	}

	return devices, nil
}

func extractDevices(items []macOSUSBDevice) []USBDevice {
	var devices []USBDevice

	for _, item := range items {
		// Get vendor and product IDs (try both old and new field names)
		vendorID := item.VendorID
		if vendorID == "" {
			vendorID = item.USBVendorID
		}
		productID := item.ProductID
		if productID == "" {
			productID = item.USBProductID
		}

		// Skip if no vendor/product ID (likely a hub or controller)
		if vendorID == "" || productID == "" {
			// Recursively process nested devices
			if len(item.Items) > 0 {
				devices = append(devices, extractDevices(item.Items)...)
			}
			continue
		}

		// Get other fields
		serial := item.SerialNum
		if serial == "" {
			serial = item.USBSerialNum
		}
		manufacturer := item.Manufacturer
		if manufacturer == "" {
			manufacturer = item.USBManufacturer
		}
		locationID := item.LocationID
		if locationID == "" {
			locationID = item.USBLocationID
		}

		// Clean up vendor/product IDs - remove 0x prefix if present
		vendorID = strings.TrimPrefix(strings.TrimSpace(vendorID), "0x")
		productID = strings.TrimPrefix(strings.TrimSpace(productID), "0x")

		// Ensure 4-digit hex format
		vendorID = fmt.Sprintf("%04s", vendorID)
		productID = fmt.Sprintf("%04s", productID)

		vendorProduct := fmt.Sprintf("%s:%s",
			strings.ToLower(vendorID),
			strings.ToLower(productID))

		// Generate bus ID from location ID
		busID := fmt.Sprintf("1-%s", strings.TrimPrefix(locationID, "0x"))

		device := USBDevice{
			Name:          item.Name,
			VendorProduct: vendorProduct,
			Vendor:        manufacturer,
			Product:       item.Name,
			Serial:        serial,
			BusID:         busID,
			IsCAC:         detectCACReader(vendorID),
		}

		devices = append(devices, device)

		// Recursively process nested devices
		if len(item.Items) > 0 {
			devices = append(devices, extractDevices(item.Items)...)
		}
	}

	return devices
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
