package controllers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	usbv1alpha1 "github.com/openshift/usb-listener-operator/api/v1alpha1"
	"github.com/openshift/usb-listener-operator/pkg/usbip"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// USBConnectionReconciler reconciles a USBConnection object
type USBConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// Track active USB/IP connections
	connections map[string]*usbip.Client
	connMutex   sync.RWMutex
}

// +kubebuilder:rbac:groups=usb.openshift.io,resources=usbconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=usb.openshift.io,resources=usbconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=usb.openshift.io,resources=usbconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups=kubevirt.io,resources=virtualmachineinstances,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *USBConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the USBConnection instance
	usbConn := &usbv1alpha1.USBConnection{}
	err := r.Get(ctx, req.NamespacedName, usbConn)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !usbConn.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, usbConn)
	}

	// Update status based on current phase
	switch usbConn.Status.Phase {
	case "", usbv1alpha1.ConnectionPhasePending:
		return r.handlePending(ctx, usbConn)
	case usbv1alpha1.ConnectionPhaseConnecting:
		return r.handleConnecting(ctx, usbConn)
	case usbv1alpha1.ConnectionPhaseConnected:
		return r.handleConnected(ctx, usbConn)
	case usbv1alpha1.ConnectionPhaseFailed:
		// Stay in failed state unless spec changes
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *USBConnectionReconciler) handlePending(ctx context.Context, usbConn *usbv1alpha1.USBConnection) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting USB connection", "device", usbConn.Spec.DeviceID, "vm", usbConn.Spec.VMName)

	// Update status to Connecting
	usbConn.Status.Phase = usbv1alpha1.ConnectionPhaseConnecting
	usbConn.Status.Message = "Initiating USB connection"
	if err := r.Status().Update(ctx, usbConn); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue to handle the Connecting phase
	return ctrl.Result{Requeue: true}, nil
}

func (r *USBConnectionReconciler) handleConnecting(ctx context.Context, usbConn *usbv1alpha1.USBConnection) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	deviceID := usbConn.Spec.DeviceID
	vmName := usbConn.Spec.VMName
	namespace := usbConn.Spec.Namespace
	workstationAddr := usbConn.Spec.WorkstationAddress

	// Step 1: Connect to workstation's USB/IP server
	log.Info("Connecting to workstation USB/IP server", "address", workstationAddr)
	usbipClient := usbip.NewClient(workstationAddr)
	if err := usbipClient.Connect(); err != nil {
		log.Error(err, "Failed to connect to USB/IP server", "address", workstationAddr)
		usbConn.Status.Phase = usbv1alpha1.ConnectionPhaseFailed
		usbConn.Status.LastError = err.Error()
		usbConn.Status.Message = fmt.Sprintf("Failed to connect to USB/IP server: %v", err)
		if updateErr := r.Status().Update(ctx, usbConn); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Step 2: Find the requested device
	log.Info("Finding USB device", "deviceID", deviceID)
	device, err := usbipClient.FindDevice(deviceID)
	if err != nil {
		usbipClient.Close()
		log.Error(err, "Device not found", "deviceID", deviceID)
		usbConn.Status.Phase = usbv1alpha1.ConnectionPhaseFailed
		usbConn.Status.LastError = err.Error()
		usbConn.Status.Message = fmt.Sprintf("Device %s not found: %v", deviceID, err)
		if updateErr := r.Status().Update(ctx, usbConn); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Step 3: Attach device via USB/IP protocol
	log.Info("Attaching USB device via USB/IP", "busID", device.BusID, "deviceID", deviceID)
	if _, err := usbipClient.AttachDevice(device.BusID); err != nil {
		usbipClient.Close()
		log.Error(err, "Failed to attach device", "busID", device.BusID)
		usbConn.Status.Phase = usbv1alpha1.ConnectionPhaseFailed
		usbConn.Status.LastError = err.Error()
		usbConn.Status.Message = fmt.Sprintf("Failed to attach device: %v", err)
		if updateErr := r.Status().Update(ctx, usbConn); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Step 4: Use virtctl to redirect to VM
	log.Info("Redirecting USB device to VM via virtctl", "vm", vmName, "namespace", namespace)
	bridge := usbip.NewVirtctlBridge("")
	if err := bridge.AttachUSBDevice(ctx, workstationAddr, deviceID, vmName, namespace); err != nil {
		usbipClient.Close()
		log.Error(err, "Failed to attach device to VM", "vm", vmName)
		usbConn.Status.Phase = usbv1alpha1.ConnectionPhaseFailed
		usbConn.Status.LastError = err.Error()
		usbConn.Status.Message = fmt.Sprintf("Failed to attach to VM: %v", err)
		if updateErr := r.Status().Update(ctx, usbConn); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Store the connection for later cleanup
	connKey := fmt.Sprintf("%s/%s", usbConn.Namespace, usbConn.Name)
	r.connMutex.Lock()
	if r.connections == nil {
		r.connections = make(map[string]*usbip.Client)
	}
	r.connections[connKey] = usbipClient
	r.connMutex.Unlock()

	// Update status to Connected
	now := metav1.Now()
	usbConn.Status.Phase = usbv1alpha1.ConnectionPhaseConnected
	usbConn.Status.ConnectedAt = &now
	usbConn.Status.Message = fmt.Sprintf("USB device %s connected to VM %s/%s", deviceID, namespace, vmName)
	usbConn.Status.LastError = ""
	if err := r.Status().Update(ctx, usbConn); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("USB connection established", "device", deviceID, "vm", vmName)

	// Requeue after a while to check connection health
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *USBConnectionReconciler) handleConnected(ctx context.Context, usbConn *usbv1alpha1.USBConnection) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	connKey := fmt.Sprintf("%s/%s", usbConn.Namespace, usbConn.Name)

	// Check if connection still exists
	r.connMutex.RLock()
	_, exists := r.connections[connKey]
	r.connMutex.RUnlock()

	if !exists {
		log.Info("USB/IP connection lost", "connection", connKey)
		usbConn.Status.Phase = usbv1alpha1.ConnectionPhaseFailed
		usbConn.Status.Message = "USB/IP connection lost"
		usbConn.Status.LastError = "Connection terminated unexpectedly"
		if err := r.Status().Update(ctx, usbConn); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// TODO: Additional health checks:
	// - Verify VM is still running via VMI API
	// - Check USB/IP connection is responsive
	// - Monitor for connection errors

	// Requeue to check again later
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *USBConnectionReconciler) handleDeletion(ctx context.Context, usbConn *usbv1alpha1.USBConnection) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Disconnecting USB device", "device", usbConn.Spec.DeviceID, "vm", usbConn.Spec.VMName)

	connKey := fmt.Sprintf("%s/%s", usbConn.Namespace, usbConn.Name)

	// Close the USB/IP connection if it exists
	r.connMutex.Lock()
	if client, exists := r.connections[connKey]; exists {
		log.Info("Closing USB/IP connection", "connection", connKey)
		if err := client.Close(); err != nil {
			log.Error(err, "Error closing USB/IP connection", "connection", connKey)
		}
		delete(r.connections, connKey)
	}
	r.connMutex.Unlock()

	// TODO: Additional cleanup:
	// - Terminate virtctl usbredir process if running
	// - Verify device is detached from VM
	// - Update USBDevice CR to mark as available

	log.Info("USB connection cleaned up", "connection", connKey)

	// Connection cleaned up, allow deletion
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *USBConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&usbv1alpha1.USBConnection{}).
		Complete(r)
}
