package usbip

import (
	"fmt"
	"io"
	"net"
	"time"
)

// Client represents a USB/IP client connection
type Client struct {
	conn    net.Conn
	address string
	timeout time.Duration
}

// NewClient creates a new USB/IP client
func NewClient(address string) *Client {
	return &Client{
		address: address,
		timeout: 10 * time.Second,
	}
}

// Connect establishes a connection to the USB/IP server
func (c *Client) Connect() error {
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.address, err)
	}
	c.conn = conn
	return nil
}

// Close closes the connection to the USB/IP server
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ListDevices retrieves the list of available USB devices from the server
func (c *Client) ListDevices() ([]USBDeviceInfo, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Send device list request
	request := NewDeviceListRequest()
	if _, err := c.conn.Write(request); err != nil {
		return nil, fmt.Errorf("failed to send device list request: %w", err)
	}

	// Read response header first
	header := make([]byte, 12)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, fmt.Errorf("failed to read response header: %w", err)
	}

	// The response contains the full device list, which can be large
	// We need to read enough to parse all devices
	// For now, read up to 64KB (should be enough for reasonable number of devices)
	response := make([]byte, 65536)
	copy(response, header)

	// Set a read deadline
	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	defer c.conn.SetReadDeadline(time.Time{})

	// Read remaining data
	n, err := c.conn.Read(response[12:])
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	totalRead := 12 + n

	// Parse the response
	devices, err := ParseDeviceListReply(response[:totalRead])
	if err != nil {
		return nil, fmt.Errorf("failed to parse device list: %w", err)
	}

	return devices, nil
}

// AttachDevice attaches a USB device by its bus ID
func (c *Client) AttachDevice(busID string) (*USBDeviceInfo, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Send import request
	request := NewImportRequest(busID)
	if _, err := c.conn.Write(request); err != nil {
		return nil, fmt.Errorf("failed to send import request: %w", err)
	}

	// Read response
	response := make([]byte, 320)

	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	defer c.conn.SetReadDeadline(time.Time{})

	n, err := io.ReadFull(c.conn, response)
	if err != nil {
		return nil, fmt.Errorf("failed to read import response: %w", err)
	}

	// Parse the response
	device, err := ParseImportReply(response[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to parse import response: %w", err)
	}

	return device, nil
}

// FindDevice finds a device by vendor:product ID
func (c *Client) FindDevice(vendorProduct string) (*USBDeviceInfo, error) {
	devices, err := c.ListDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.FormatDeviceID() == vendorProduct {
			return &device, nil
		}
	}

	return nil, fmt.Errorf("device %s not found", vendorProduct)
}

// DetachDevice detaches a USB device
// Note: USB/IP doesn't have a specific detach command; we just close the connection
func (c *Client) DetachDevice() error {
	return c.Close()
}
