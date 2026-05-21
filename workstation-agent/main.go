package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

const (
	// Default HTTP API port
	DefaultAPIPort = 8080
)

type Config struct {
	APIPort    int
	Owner      string
	Kubeconfig string
}

// USBDevice represents a USB device discovered on the workstation
type USBDevice struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	VendorProduct string `json:"vendorProduct"` // Format: "090c:1000"
	Vendor        string `json:"vendor"`
	Product       string `json:"product"`
	Serial        string `json:"serial"`
	BusID         string `json:"busID"`
	IsCAC         bool   `json:"isCAC"`
	Owner         string `json:"owner"`
}

// USBConnection represents an active USB passthrough connection
type USBConnection struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"deviceId"`
	DeviceName string    `json:"deviceName"`
	VMName     string    `json:"vmName"`
	Namespace  string    `json:"namespace"`
	Status     string    `json:"status"` // "Connecting", "Connected", "Failed"
	Message    string    `json:"message,omitempty"`
	StartedAt  time.Time `json:"startedAt"`
	cmd        *exec.Cmd
	cancel     context.CancelFunc
}

// Server manages the HTTP API and active connections
type Server struct {
	config      *Config
	connections map[string]*USBConnection
	mu          sync.RWMutex
}

func main() {
	// Check if we need to elevate privileges
	if !isElevated() {
		log.Println("USB passthrough requires elevated privileges. Re-launching with sudo...")
		if err := relaunchElevated(); err != nil {
			log.Fatalf("Failed to elevate privileges: %v\n"+
				"Please run with sudo: sudo ./usb-agent --kubeconfig ~/.kube/config", err)
		}
		return
	}

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

	// Check for virtctl
	virtctlPath, err := exec.LookPath("virtctl")
	if err != nil {
		log.Fatalf("virtctl not found in PATH. Please install OpenShift Virtualization CLI:\n"+
			"  Download from: https://docs.openshift.com/container-platform/latest/virt/virt-using-the-cli-tools.html\n"+
			"  Or install with: brew install virtctl (macOS)")
	}
	log.Printf("Found virtctl at: %s\n", virtctlPath)

	// Verify kubeconfig
	if config.Kubeconfig == "" {
		log.Println("Warning: No kubeconfig specified. virtctl will use default kubeconfig location.")
	} else {
		log.Printf("Using kubeconfig: %s\n", config.Kubeconfig)
	}

	// Initialize server
	server := &Server{
		config:      config,
		connections: make(map[string]*USBConnection),
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/devices", server.handleDevices)
	mux.HandleFunc("/connections", server.handleConnections)
	mux.HandleFunc("/attach", server.handleAttach)
	mux.HandleFunc("/detach/", server.handleDetach)

	// Add CORS middleware
	handler := corsMiddleware(mux)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.APIPort),
		Handler: handler,
	}

	go func() {
		log.Printf("Starting HTTP API server on port %d...\n", config.APIPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	printStatus(config)

	// Wait for shutdown
	<-ctx.Done()

	// Graceful shutdown
	log.Println("Stopping all USB connections...")
	server.mu.Lock()
	for _, conn := range server.connections {
		if conn.cancel != nil {
			conn.cancel()
		}
	}
	server.mu.Unlock()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)

	log.Println("Shutdown complete")
}

func parseFlags() *Config {
	config := &Config{}

	flag.IntVar(&config.APIPort, "port", DefaultAPIPort,
		"Port for HTTP API server")
	flag.StringVar(&config.Owner, "owner", getDefaultOwner(),
		"Owner name for device registration")
	flag.StringVar(&config.Kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"),
		"Path to kubeconfig file (defaults to $KUBECONFIG or ~/.kube/config)")

	flag.Parse()

	// Default kubeconfig location
	if config.Kubeconfig == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			config.Kubeconfig = fmt.Sprintf("%s/.kube/config", homeDir)
		}
	}

	return config
}

func getDefaultOwner() string {
	if owner := os.Getenv("USER"); owner != "" {
		return owner
	}
	if owner := os.Getenv("USERNAME"); owner != "" {
		return owner
	}
	hostname, _ := os.Hostname()
	return hostname
}

// HTTP Handlers

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devices, err := enumerateUSBDevices()
	if err != nil {
		log.Printf("Error enumerating USB devices: %v", err)
		http.Error(w, fmt.Sprintf("Failed to enumerate USB devices: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to API format
	apiDevices := make([]USBDevice, len(devices))
	for i, dev := range devices {
		apiDevices[i] = USBDevice{
			ID:            dev.VendorProduct, // Use vendor:product as ID
			Name:          dev.Name,
			VendorProduct: dev.VendorProduct,
			Vendor:        dev.Vendor,
			Product:       dev.Product,
			Serial:        dev.Serial,
			BusID:         dev.BusID,
			IsCAC:         dev.IsCAC,
			Owner:         s.config.Owner,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiDevices)
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	connections := make([]*USBConnection, 0, len(s.connections))
	for _, conn := range s.connections {
		// Don't expose internal fields
		connections = append(connections, &USBConnection{
			ID:         conn.ID,
			DeviceID:   conn.DeviceID,
			DeviceName: conn.DeviceName,
			VMName:     conn.VMName,
			Namespace:  conn.Namespace,
			Status:     conn.Status,
			Message:    conn.Message,
			StartedAt:  conn.StartedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(connections)
}

func (s *Server) handleAttach(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DeviceID   string `json:"deviceId"`
		DeviceName string `json:"deviceName"`
		VMName     string `json:"vmName"`
		Namespace  string `json:"namespace"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" || req.VMName == "" || req.Namespace == "" {
		http.Error(w, "Missing required fields: deviceId, vmName, namespace", http.StatusBadRequest)
		return
	}

	// Create connection
	connID := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())

	conn := &USBConnection{
		ID:         connID,
		DeviceID:   req.DeviceID,
		DeviceName: req.DeviceName,
		VMName:     req.VMName,
		Namespace:  req.Namespace,
		Status:     "Connecting",
		StartedAt:  time.Now(),
		cancel:     cancel,
	}

	s.mu.Lock()
	s.connections[connID] = conn
	s.mu.Unlock()

	// Start virtctl in background
	go s.runVirtctl(ctx, conn)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":      connID,
		"status":  "Connecting",
		"message": "Starting USB redirection...",
	})
}

func (s *Server) handleDetach(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract connection ID from path: /detach/{id}
	connID := r.URL.Path[len("/detach/"):]
	if connID == "" {
		http.Error(w, "Missing connection ID", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	conn, exists := s.connections[connID]
	if !exists {
		s.mu.Unlock()
		http.Error(w, "Connection not found", http.StatusNotFound)
		return
	}

	// Cancel the context to stop virtctl
	if conn.cancel != nil {
		conn.cancel()
	}

	delete(s.connections, connID)
	s.mu.Unlock()

	log.Printf("Detached connection %s (device %s from VM %s/%s)\n",
		connID, conn.DeviceID, conn.Namespace, conn.VMName)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) runVirtctl(ctx context.Context, conn *USBConnection) {
	log.Printf("Starting virtctl usbredir for device %s to VM %s/%s\n",
		conn.DeviceID, conn.Namespace, conn.VMName)

	// Build virtctl command
	// virtctl usbredir <vendor>:<product> <vm-name> -n <namespace>
	args := []string{
		"usbredir",
		conn.DeviceID,
		conn.VMName,
		"-n", conn.Namespace,
	}

	// Add kubeconfig if specified
	if s.config.Kubeconfig != "" {
		args = append([]string{"--kubeconfig", s.config.Kubeconfig}, args...)
	}

	cmd := exec.CommandContext(ctx, "virtctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	conn.cmd = cmd

	// Run virtctl
	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("Failed to start virtctl: %v", err)
		log.Printf("%s\n", errMsg)
		s.mu.Lock()
		conn.Status = "Failed"
		conn.Message = errMsg
		s.mu.Unlock()
		return
	}

	log.Printf("virtctl started (PID %d) for device %s\n", cmd.Process.Pid, conn.DeviceID)

	// Wait a moment for virtctl to initialize and potentially fail fast
	time.Sleep(500 * time.Millisecond)

	// Check if process is still running
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		errMsg := "virtctl exited immediately - device may not be accessible (try running agent with sudo)"
		log.Printf("%s\n", errMsg)
		s.mu.Lock()
		conn.Status = "Failed"
		conn.Message = errMsg
		s.mu.Unlock()
		return
	}

	// Mark as connected
	s.mu.Lock()
	conn.Status = "Connected"
	conn.Message = "USB device redirected to VM"
	s.mu.Unlock()

	// Wait for completion
	err := cmd.Wait()
	if err != nil && ctx.Err() == nil {
		// Only log error if context wasn't cancelled (i.e., not a normal detach)
		errMsg := fmt.Sprintf("virtctl exited with error: %v", err)
		log.Printf("%s\n", errMsg)
		s.mu.Lock()
		conn.Status = "Failed"
		conn.Message = errMsg
		s.mu.Unlock()
	} else {
		log.Printf("virtctl stopped for device %s\n", conn.DeviceID)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from OpenShift Console
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func printStatus(config *Config) {
	fmt.Println("\n╔════════════════════════════════════════════════════════╗")
	fmt.Println("║    USB Workstation Agent - Running                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Printf("\n  API Port: %d\n", config.APIPort)
	fmt.Printf("  Owner: %s\n", config.Owner)
	fmt.Printf("  Platform: %s\n\n", runtime.GOOS)

	fmt.Println("  Endpoints:")
	fmt.Printf("    GET  http://localhost:%d/devices\n", config.APIPort)
	fmt.Printf("    GET  http://localhost:%d/connections\n", config.APIPort)
	fmt.Printf("    POST http://localhost:%d/attach\n", config.APIPort)
	fmt.Printf("    DEL  http://localhost:%d/detach/{id}\n\n", config.APIPort)

	fmt.Println("  Status: 🟢 Running (elevated)")
	fmt.Println("  Press Ctrl+C to stop")
	fmt.Println()
}

// Platform-specific privilege check and elevation functions
// These are set by init() functions in privilege_windows.go and privilege_unix.go
var (
	isElevatedFunc       func() bool
	relaunchElevatedFunc func() error
)

// isElevated checks if the process is running with elevated privileges
// Platform-specific implementations in privilege_*.go files
func isElevated() bool {
	if isElevatedFunc == nil {
		// Fallback for platforms without specific implementation
		return runtime.GOOS == "windows" || os.Geteuid() == 0
	}
	return isElevatedFunc()
}

// relaunchElevated re-launches the current process with elevated privileges
// Platform-specific implementations in privilege_*.go files
func relaunchElevated() error {
	if relaunchElevatedFunc == nil {
		return fmt.Errorf("privilege elevation not implemented for platform %s", runtime.GOOS)
	}
	return relaunchElevatedFunc()
}
