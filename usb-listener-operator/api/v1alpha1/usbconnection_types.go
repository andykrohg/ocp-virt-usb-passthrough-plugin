package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// USBConnectionSpec defines the desired state of USBConnection
type USBConnectionSpec struct {
	// WorkstationAddress is the IP:port of the workstation running USB/IP server
	// Example: "192.168.1.100:3240"
	WorkstationAddress string `json:"workstationAddress"`

	// DeviceID is the USB vendor:product ID
	// Example: "090c:1000"
	DeviceID string `json:"deviceID"`

	// VMName is the name of the VirtualMachineInstance to attach the device to
	VMName string `json:"vmName"`

	// Namespace is the namespace containing the VM
	Namespace string `json:"namespace"`

	// DeviceName is a human-readable name for the device
	// Example: "Samsung USB Drive" or "CAC Card Reader"
	// +optional
	DeviceName string `json:"deviceName,omitempty"`
}

// USBConnectionStatus defines the observed state of USBConnection
type USBConnectionStatus struct {
	// Phase represents the current phase of the USB connection
	// Possible values: Pending, Connecting, Connected, Disconnecting, Failed
	Phase ConnectionPhase `json:"phase,omitempty"`

	// Message provides additional details about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// ConnectedAt is the timestamp when the connection was established
	// +optional
	ConnectedAt *metav1.Time `json:"connectedAt,omitempty"`

	// LastError contains the last error message if connection failed
	// +optional
	LastError string `json:"lastError,omitempty"`
}

// ConnectionPhase represents the phase of a USB connection
// +kubebuilder:validation:Enum=Pending;Connecting;Connected;Disconnecting;Failed
type ConnectionPhase string

const (
	ConnectionPhasePending       ConnectionPhase = "Pending"
	ConnectionPhaseConnecting    ConnectionPhase = "Connecting"
	ConnectionPhaseConnected     ConnectionPhase = "Connected"
	ConnectionPhaseDisconnecting ConnectionPhase = "Disconnecting"
	ConnectionPhaseFailed        ConnectionPhase = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=usbconn
// +kubebuilder:printcolumn:name="Device",type=string,JSONPath=`.spec.deviceID`
// +kubebuilder:printcolumn:name="VM",type=string,JSONPath=`.spec.vmName`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// USBConnection is the Schema for the usbconnections API
type USBConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   USBConnectionSpec   `json:"spec,omitempty"`
	Status USBConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// USBConnectionList contains a list of USBConnection
type USBConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []USBConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&USBConnection{}, &USBConnectionList{})
}
