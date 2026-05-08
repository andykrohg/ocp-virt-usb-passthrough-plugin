package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// USBDeviceSpec defines the desired state of USBDevice
type USBDeviceSpec struct {
	// WorkstationAddress is the IP:port of the workstation offering this device
	WorkstationAddress string `json:"workstationAddress"`

	// DeviceID is the USB vendor:product ID
	DeviceID string `json:"deviceID"`

	// DeviceName is a human-readable name
	DeviceName string `json:"deviceName"`

	// VendorName is the USB vendor name
	// +optional
	VendorName string `json:"vendorName,omitempty"`

	// Serial is the device serial number
	// +optional
	Serial string `json:"serial,omitempty"`

	// IsCAC indicates if this is a CAC card reader
	// +optional
	IsCAC bool `json:"isCAC,omitempty"`

	// Owner is the user who owns the workstation
	// +optional
	Owner string `json:"owner,omitempty"`
}

// USBDeviceStatus defines the observed state of USBDevice
type USBDeviceStatus struct {
	// Available indicates if the device is currently available for connection
	Available bool `json:"available"`

	// ConnectedTo indicates which VM this device is currently connected to
	// Format: "namespace/vmname"
	// +optional
	ConnectedTo string `json:"connectedTo,omitempty"`

	// LastSeen is when the workstation last advertised this device
	LastSeen metav1.Time `json:"lastSeen,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=usbdev
// +kubebuilder:printcolumn:name="Device",type=string,JSONPath=`.spec.deviceID`
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.deviceName`
// +kubebuilder:printcolumn:name="Available",type=boolean,JSONPath=`.status.available`
// +kubebuilder:printcolumn:name="Connected To",type=string,JSONPath=`.status.connectedTo`

// USBDevice represents a USB device advertised by a workstation
type USBDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   USBDeviceSpec   `json:"spec,omitempty"`
	Status USBDeviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// USBDeviceList contains a list of USBDevice
type USBDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []USBDevice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&USBDevice{}, &USBDeviceList{})
}
