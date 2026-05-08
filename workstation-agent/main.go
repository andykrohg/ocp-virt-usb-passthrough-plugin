package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// USB/IP default port
	DefaultUSBIPPort = 3240

	// Heartbeat interval for device registration
	HeartbeatInterval = 30 * time.Second
)

type Config struct {
	Kubeconfig  string
	USBIPPort   int
	Namespace   string
	Owner       string
	ClusterAddr string
}

func main() {
	config := parseFlags()

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	// Start USB/IP server
	log.Printf("Starting USB/IP server on port %d...\n", config.USBIPPort)
	if err := startUSBIPServer(ctx, config.USBIPPort); err != nil {
		log.Printf("Warning: Failed to start USB/IP server: %v\n", err)
		log.Println("Continuing without USB/IP server (device registration only mode)")
		log.Println("USB connections will not work, but devices will be registered in the cluster")
	}

	// Get local devices
	devices, err := enumerateUSBDevices()
	if err != nil {
		log.Fatalf("Failed to enumerate USB devices: %v", err)
	}

	log.Printf("Found %d USB devices\n", len(devices))
	for _, dev := range devices {
		log.Printf("  - %s (%s)\n", dev.Name, dev.VendorProduct)
	}

	// Connect to cluster
	if config.Kubeconfig != "" {
		log.Println("Connecting to cluster...")
		if err := registerDevices(ctx, config, devices); err != nil {
			log.Printf("Warning: Failed to register devices with cluster: %v\n", err)
			log.Println("Running in standalone mode...")
		} else {
			log.Println("Devices registered with cluster")
			// Start heartbeat to keep device registrations fresh
			go heartbeat(ctx, config, devices)
		}
	}

	// System tray icon (platform-specific)
	// For now, just print status
	printStatus(devices, config)

	// Wait for shutdown
	<-ctx.Done()
	log.Println("Shutdown complete")
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.Kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"),
		"Path to kubeconfig file (defaults to $KUBECONFIG or ~/.kube/config)")
	flag.IntVar(&config.USBIPPort, "port", DefaultUSBIPPort,
		"Port for USB/IP server")
	flag.StringVar(&config.Namespace, "namespace", "default",
		"Namespace to register USBDevice resources")
	flag.StringVar(&config.Owner, "owner", os.Getenv("USER"),
		"Owner name for device registration")
	flag.StringVar(&config.ClusterAddr, "cluster", "",
		"Cluster address to advertise (auto-detected if not specified)")

	flag.Parse()

	// Default kubeconfig location
	if config.Kubeconfig == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			config.Kubeconfig = fmt.Sprintf("%s/.kube/config", homeDir)
		}
	}

	// Auto-detect cluster address
	if config.ClusterAddr == "" {
		config.ClusterAddr = getLocalIP()
	}

	return config
}

func startUSBIPServer(ctx context.Context, port int) error {
	// Check if usbipd is installed
	usbipd, err := exec.LookPath("usbipd")
	if err != nil {
		return fmt.Errorf("usbipd not found in PATH. Please install USB/IP:\n"+
			"  Linux: sudo apt install usbip (or yum install usbip-utils)\n"+
			"  Windows: Install USB/IP for Windows from https://github.com/cezanne/usbip-win\n"+
			"  macOS: brew install usbip (if available)")
	}

	log.Printf("Found usbipd at: %s\n", usbipd)

	// Start usbipd daemon
	// Platform-specific command construction
	var cmd *exec.Cmd

	if isWindows() {
		// Windows: usbipd.exe -d (debug mode for console output)
		cmd = exec.CommandContext(ctx, usbipd, "-d")
	} else if isMacOS() {
		// macOS: usbipd daemon (homebrew version uses subcommand)
		cmd = exec.CommandContext(ctx, "sudo", usbipd, "daemon")
	} else {
		// Linux: sudo usbipd -D (daemon mode)
		cmd = exec.CommandContext(ctx, "sudo", usbipd, "-D")
	}

	// Capture output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Starting USB/IP server (requires elevated privileges)...")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start usbipd: %w\nMake sure you have permission to run with elevated privileges", err)
	}

	// Wait for server to be ready
	time.Sleep(1 * time.Second)

	// Verify server is listening
	if err := checkUSBIPServer(port); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("usbipd started but not listening: %w", err)
	}

	log.Printf("USB/IP server listening on port %d\n", port)

	// Keep server running in background
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("usbipd exited: %v\n", err)
		}
	}()

	return nil
}

func checkUSBIPServer(port int) error {
	// Try to connect to the USB/IP port
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		return fmt.Errorf("server not responding on port %d: %w", port, err)
	}
	conn.Close()
	return nil
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func isMacOS() bool {
	return runtime.GOOS == "darwin"
}

// USBDevice represents a USB device discovered on the workstation
type USBDevice struct {
	Name          string
	VendorProduct string // Format: "090c:1000"
	Vendor        string
	Product       string
	Serial        string
	BusID         string
	IsCAC         bool
}

// enumerateUSBDevices is implemented in platform-specific files:
// - usb_darwin.go for macOS
// - usb_linux.go for Linux
// - usb_windows.go for Windows

func registerDevices(ctx context.Context, config *Config, devices []USBDevice) error {
	// Load kubeconfig
	restConfig, err := clientcmd.BuildConfigFromFlags("", config.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Define USBDevice GVR (Group/Version/Resource)
	usbDeviceGVR := schema.GroupVersionResource{
		Group:    "usb.openshift.io",
		Version:  "v1alpha1",
		Resource: "usbdevices",
	}

	workstationAddr := fmt.Sprintf("%s:%d", config.ClusterAddr, config.USBIPPort)
	now := metav1.Now()

	// Create or update each device
	for _, device := range devices {
		// Generate consistent name from device ID
		deviceName := fmt.Sprintf("usb-%s-%s",
			config.Owner,
			sanitizeName(device.VendorProduct))

		// Build USBDevice resource
		usbDevice := map[string]interface{}{
			"apiVersion": "usb.openshift.io/v1alpha1",
			"kind":       "USBDevice",
			"metadata": map[string]interface{}{
				"name":      deviceName,
				"namespace": config.Namespace,
			},
			"spec": map[string]interface{}{
				"workstationAddress": workstationAddr,
				"deviceID":           device.VendorProduct,
				"deviceName":         device.Name,
				"vendorName":         device.Vendor,
				"serial":             device.Serial,
				"isCAC":              device.IsCAC,
				"owner":              config.Owner,
			},
			"status": map[string]interface{}{
				"available":   true,
				"connectedTo": "",
				"lastSeen":    now.Format(time.RFC3339),
			},
		}

		unstructuredDevice := &unstructured.Unstructured{}
		unstructuredDevice.SetUnstructuredContent(usbDevice)

		// Create or update the resource
		_, err := dynamicClient.Resource(usbDeviceGVR).
			Namespace(config.Namespace).
			Create(ctx, unstructuredDevice, metav1.CreateOptions{})

		if err != nil {
			// If already exists, update it
			if isAlreadyExistsError(err) {
				_, err = dynamicClient.Resource(usbDeviceGVR).
					Namespace(config.Namespace).
					Update(ctx, unstructuredDevice, metav1.UpdateOptions{})
				if err != nil {
					log.Printf("Warning: failed to update device %s: %v\n", deviceName, err)
				}
			} else {
				log.Printf("Warning: failed to create device %s: %v\n", deviceName, err)
			}
		}
	}

	log.Printf("Registered %d devices with cluster\n", len(devices))
	return nil
}

func heartbeat(ctx context.Context, config *Config, devices []USBDevice) {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Re-enumerate devices in case USB devices were plugged/unplugged
			currentDevices, err := enumerateUSBDevices()
			if err != nil {
				log.Printf("Heartbeat: failed to enumerate devices: %v\n", err)
				continue
			}

			// Update device registrations
			if err := registerDevices(ctx, config, currentDevices); err != nil {
				log.Printf("Heartbeat: failed to update devices: %v\n", err)
			} else {
				log.Printf("Heartbeat: updated %d devices\n", len(currentDevices))
			}
		}
	}
}

func sanitizeName(s string) string {
	// Replace invalid Kubernetes name characters
	// Valid: lowercase alphanumeric, -, .
	result := ""
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			result += string(ch)
		} else if ch == ':' || ch == '_' || ch == ' ' {
			result += "-"
		}
	}
	return result
}

func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "already exists")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && anySubstring(s, substr))
}

func anySubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func printStatus(devices []USBDevice, config *Config) {
	fmt.Println("\n╔════════════════════════════════════════════════════════╗")
	fmt.Println("║    USB Workstation Agent - Running                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Printf("\n  Port: %d\n", config.USBIPPort)
	fmt.Printf("  Owner: %s\n", config.Owner)
	fmt.Printf("  Devices: %d\n\n", len(devices))

	for i, dev := range devices {
		fmt.Printf("  %d. %s (%s)\n", i+1, dev.Name, dev.VendorProduct)
		if dev.IsCAC {
			fmt.Println("     🔒 CAC Reader")
		}
	}

	fmt.Println("\n  Status: 🟢 Running")
	fmt.Println("  Press Ctrl+C to stop")
	fmt.Println()
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
