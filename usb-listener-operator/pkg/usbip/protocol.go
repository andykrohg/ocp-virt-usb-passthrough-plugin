package usbip

import (
	"encoding/binary"
	"fmt"
)

// USB/IP Protocol Version
const (
	USBIPVersion = 0x0111 // Version 1.1.1
)

// Operation codes
const (
	OpReqDevlist  = 0x8005 // Request device list
	OpRepDevlist  = 0x0005 // Reply device list
	OpReqImport   = 0x8003 // Request to import (attach) device
	OpRepImport   = 0x0003 // Reply to import request
	OpCmdSubmit   = 0x0001 // Submit URB
	OpRetSubmit   = 0x0003 // Return of submitted URB
	OpCmdUnlink   = 0x0002 // Unlink URB
	OpRetUnlink   = 0x0004 // Return of unlinked URB
)

// Status codes
const (
	StatusSuccess = 0x00000000
	StatusError   = 0x00000001
)

// USBIPHeader represents the common header for all USB/IP messages
type USBIPHeader struct {
	Version   uint16
	Command   uint16
	Status    uint32
	SeqNum    uint32
	DevID     uint32
	Direction uint32
	EP        uint32
}

// USBDeviceInfo represents information about a USB device
type USBDeviceInfo struct {
	Path         string // USB device path (e.g., "1-1")
	BusID        string // Bus ID (e.g., "1-1")
	BusNum       uint32
	DevNum       uint32
	Speed        uint32
	IDVendor     uint16
	IDProduct    uint16
	BCDDevice    uint16
	DeviceClass  uint8
	DeviceSubclass uint8
	DeviceProtocol uint8
	ConfigurationValue uint8
	NumConfigurations  uint8
	NumInterfaces      uint8
}

// USBInterfaceInfo represents information about a USB interface
type USBInterfaceInfo struct {
	InterfaceClass    uint8
	InterfaceSubclass uint8
	InterfaceProtocol uint8
	Padding           uint8 // Alignment padding
}

// DeviceListRequest creates a request to list available USB devices
func NewDeviceListRequest() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint16(buf[0:2], USBIPVersion)
	binary.BigEndian.PutUint16(buf[2:4], OpReqDevlist)
	binary.BigEndian.PutUint32(buf[4:8], StatusSuccess)
	return buf
}

// ImportRequest creates a request to import (attach) a USB device
func NewImportRequest(busID string) []byte {
	buf := make([]byte, 40)
	binary.BigEndian.PutUint16(buf[0:2], USBIPVersion)
	binary.BigEndian.PutUint16(buf[2:4], OpReqImport)
	binary.BigEndian.PutUint32(buf[4:8], StatusSuccess)

	// BusID is a 32-byte null-terminated string
	copy(buf[8:], []byte(busID))

	return buf
}

// ParseDeviceListReply parses the reply to a device list request
func ParseDeviceListReply(data []byte) ([]USBDeviceInfo, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("response too short: %d bytes", len(data))
	}

	version := binary.BigEndian.Uint16(data[0:2])
	command := binary.BigEndian.Uint16(data[2:4])
	status := binary.BigEndian.Uint32(data[4:8])

	if version != USBIPVersion {
		return nil, fmt.Errorf("unsupported protocol version: 0x%04x", version)
	}

	if command != OpRepDevlist {
		return nil, fmt.Errorf("unexpected command: 0x%04x", command)
	}

	if status != StatusSuccess {
		return nil, fmt.Errorf("server returned error status: 0x%08x", status)
	}

	if len(data) < 12 {
		return nil, fmt.Errorf("response missing device count")
	}

	numExportedDevices := binary.BigEndian.Uint32(data[8:12])

	devices := make([]USBDeviceInfo, 0, numExportedDevices)
	offset := 12

	for i := uint32(0); i < numExportedDevices; i++ {
		if len(data) < offset+312 { // Minimum size for device info
			return nil, fmt.Errorf("incomplete device info at offset %d", offset)
		}

		device := USBDeviceInfo{}

		// Parse device path (256 bytes, null-terminated)
		pathBytes := data[offset : offset+256]
		pathEnd := 0
		for pathEnd < len(pathBytes) && pathBytes[pathEnd] != 0 {
			pathEnd++
		}
		device.Path = string(pathBytes[:pathEnd])
		offset += 256

		// Parse bus ID (32 bytes, null-terminated)
		busIDBytes := data[offset : offset+32]
		busIDEnd := 0
		for busIDEnd < len(busIDBytes) && busIDBytes[busIDEnd] != 0 {
			busIDEnd++
		}
		device.BusID = string(busIDBytes[:busIDEnd])
		offset += 32

		// Parse device numbers
		device.BusNum = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
		device.DevNum = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
		device.Speed = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Parse USB descriptor fields
		device.IDVendor = binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2
		device.IDProduct = binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2
		device.BCDDevice = binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2

		device.DeviceClass = data[offset]
		offset++
		device.DeviceSubclass = data[offset]
		offset++
		device.DeviceProtocol = data[offset]
		offset++
		device.ConfigurationValue = data[offset]
		offset++
		device.NumConfigurations = data[offset]
		offset++
		device.NumInterfaces = data[offset]
		offset++

		// Skip interface info for now (we can parse it if needed)
		// Each interface is 4 bytes, so skip NumInterfaces * 4
		interfaceBytes := int(device.NumInterfaces) * 4
		if len(data) < offset+interfaceBytes {
			return nil, fmt.Errorf("incomplete interface info at offset %d", offset)
		}
		offset += interfaceBytes

		devices = append(devices, device)
	}

	return devices, nil
}

// ParseImportReply parses the reply to an import request
func ParseImportReply(data []byte) (*USBDeviceInfo, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("response too short: %d bytes", len(data))
	}

	version := binary.BigEndian.Uint16(data[0:2])
	command := binary.BigEndian.Uint16(data[2:4])
	status := binary.BigEndian.Uint32(data[4:8])

	if version != USBIPVersion {
		return nil, fmt.Errorf("unsupported protocol version: 0x%04x", version)
	}

	if command != OpRepImport {
		return nil, fmt.Errorf("unexpected command: 0x%04x", command)
	}

	if status != StatusSuccess {
		return nil, fmt.Errorf("import failed with status: 0x%08x", status)
	}

	if len(data) < 320 {
		return nil, fmt.Errorf("import reply too short: %d bytes", len(data))
	}

	device := &USBDeviceInfo{}
	offset := 8

	// Parse device path (256 bytes)
	pathBytes := data[offset : offset+256]
	pathEnd := 0
	for pathEnd < len(pathBytes) && pathBytes[pathEnd] != 0 {
		pathEnd++
	}
	device.Path = string(pathBytes[:pathEnd])
	offset += 256

	// Parse bus ID (32 bytes)
	busIDBytes := data[offset : offset+32]
	busIDEnd := 0
	for busIDEnd < len(busIDBytes) && busIDBytes[busIDEnd] != 0 {
		busIDEnd++
	}
	device.BusID = string(busIDBytes[:busIDEnd])
	offset += 32

	// Parse device info
	device.BusNum = binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4
	device.DevNum = binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4
	device.Speed = binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	device.IDVendor = binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	device.IDProduct = binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	device.BCDDevice = binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	return device, nil
}

// FormatDeviceID formats vendor and product IDs as "vendor:product"
func (d *USBDeviceInfo) FormatDeviceID() string {
	return fmt.Sprintf("%04x:%04x", d.IDVendor, d.IDProduct)
}

// FormatSpeed returns a human-readable speed string
func (d *USBDeviceInfo) FormatSpeed() string {
	switch d.Speed {
	case 1:
		return "1.5 Mb/s (Low Speed)"
	case 2:
		return "12 Mb/s (Full Speed)"
	case 3:
		return "480 Mb/s (High Speed)"
	case 4:
		return "5 Gb/s (Super Speed)"
	case 5:
		return "10 Gb/s (Super Speed+)"
	default:
		return fmt.Sprintf("Unknown (%d)", d.Speed)
	}
}
