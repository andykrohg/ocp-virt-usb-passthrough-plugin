import * as React from 'react';
import {
  Alert,
  AlertVariant,
  Button,
  ButtonVariant,
  Card,
  CardBody,
  CardTitle,
  EmptyState,
  EmptyStateBody,
  EmptyStateIcon,
  Grid,
  GridItem,
  List,
  ListItem,
  Modal,
  ModalVariant,
  Select,
  SelectOption,
  SelectVariant,
  Spinner,
  Title,
} from '@patternfly/react-core';
import {
  CheckCircleIcon,
  UsbIcon,
  ExclamationCircleIcon,
} from '@patternfly/react-icons';
import {
  useK8sWatchResource,
  k8sCreate,
  k8sDelete,
} from '@openshift-console/dynamic-plugin-sdk';

interface VMUSBTabProps {
  obj?: any; // VirtualMachine or VirtualMachineInstance object
}

const VMUSBTab: React.FC<VMUSBTabProps> = (props) => {
  const { obj } = props || {};
  const [isSelectOpen, setIsSelectOpen] = React.useState(false);
  const [selectedDeviceId, setSelectedDeviceId] = React.useState<string>('');
  const [isAttaching, setIsAttaching] = React.useState(false);
  const [error, setError] = React.useState<string>('');
  const [showDetachModal, setShowDetachModal] = React.useState(false);
  const [deviceToDetach, setDeviceToDetach] = React.useState<any>(null);

  const vmName = obj?.metadata?.name;
  const vmNamespace = obj?.metadata?.namespace;

  // Watch USBDevice resources
  const [usbDevices, devicesLoaded, devicesError] = useK8sWatchResource({
    groupVersionKind: {
      group: 'usb.openshift.io',
      version: 'v1alpha1',
      kind: 'USBDevice',
    },
    isList: true,
  });

  // Watch USBConnection resources for this VM
  const [connections, connectionsLoaded, connectionsError] = useK8sWatchResource({
    groupVersionKind: {
      group: 'usb.openshift.io',
      version: 'v1alpha1',
      kind: 'USBConnection',
    },
    isList: true,
    namespace: vmNamespace,
  });

  const availableDevices = React.useMemo(() => {
    return (usbDevices as any[])?.filter((dev) => dev.status?.available) || [];
  }, [usbDevices]);

  const vmConnections = React.useMemo(() => {
    return (connections as any[])?.filter(
      (conn) => conn.spec?.vmName === vmName && conn.spec?.namespace === vmNamespace
    ) || [];
  }, [connections, vmName, vmNamespace]);

  const handleAttach = async () => {
    if (!selectedDeviceId) return;

    setIsAttaching(true);
    setError('');

    try {
      const selectedDevice = availableDevices.find(
        (dev) => dev.spec?.deviceID === selectedDeviceId
      );

      if (!selectedDevice) {
        throw new Error('Selected device not found');
      }

      const connectionName = `${vmName}-${selectedDeviceId.replace(':', '-')}`;

      const usbConnection = {
        apiVersion: 'usb.openshift.io/v1alpha1',
        kind: 'USBConnection',
        metadata: {
          name: connectionName,
          namespace: vmNamespace,
        },
        spec: {
          workstationAddress: selectedDevice.spec?.workstationAddress,
          deviceID: selectedDeviceId,
          deviceName: selectedDevice.spec?.deviceName || selectedDeviceId,
          vmName: vmName,
          namespace: vmNamespace,
        },
      };

      await k8sCreate({
        model: {
          apiGroup: 'usb.openshift.io',
          apiVersion: 'v1alpha1',
          kind: 'USBConnection',
          plural: 'usbconnections',
          abbr: 'USBCONN',
          label: 'USBConnection',
          labelPlural: 'USBConnections',
        },
        data: usbConnection,
      });

      setSelectedDeviceId('');
      setIsSelectOpen(false);
    } catch (err: any) {
      setError(err.message || 'Failed to attach USB device');
    } finally {
      setIsAttaching(false);
    }
  };

  const handleDetachConfirm = async () => {
    if (!deviceToDetach) return;

    try {
      await k8sDelete({
        model: {
          apiGroup: 'usb.openshift.io',
          apiVersion: 'v1alpha1',
          kind: 'USBConnection',
          plural: 'usbconnections',
          abbr: 'USBCONN',
          label: 'USBConnection',
          labelPlural: 'USBConnections',
        },
        resource: deviceToDetach,
      });

      setShowDetachModal(false);
      setDeviceToDetach(null);
    } catch (err: any) {
      setError(err.message || 'Failed to detach USB device');
    }
  };

  const openDetachModal = (connection: any) => {
    setDeviceToDetach(connection);
    setShowDetachModal(true);
  };

  if (!devicesLoaded || !connectionsLoaded) {
    return (
      <EmptyState>
        <EmptyStateIcon variant="container" component={Spinner} />
        <Title headingLevel="h2" size="lg">
          Loading USB devices...
        </Title>
      </EmptyState>
    );
  }

  if (devicesError) {
    return (
      <Alert variant="danger" title="Error loading USB devices">
        {devicesError.message}
      </Alert>
    );
  }

  return (
    <>
      <Grid hasGutter>
        <GridItem span={12 as any}>
          {error && (
            <Alert
              variant="danger"
              title="Error"
              actionClose={<Button variant="plain" onClick={() => setError('')} />}
            >
              {error}
            </Alert>
          )}
        </GridItem>

        {/* Connected Devices Section */}
        <GridItem span={12 as any}>
          <Card>
            <CardTitle>
              <Title headingLevel="h3">Connected USB Devices</Title>
            </CardTitle>
            <CardBody>
              {vmConnections.length === 0 ? (
                <EmptyState variant="xs">
                  <EmptyStateIcon icon={UsbIcon} />
                  <Title headingLevel="h4" size="md">
                    No USB devices connected
                  </Title>
                  <EmptyStateBody>
                    Connect a USB device from your workstation to this VM using the form below.
                  </EmptyStateBody>
                </EmptyState>
              ) : (
                <List>
                  {vmConnections.map((conn) => {
                    const phase = conn.status?.phase;
                    const isConnected = phase === 'Connected';
                    const isFailed = phase === 'Failed';

                    return (
                      <ListItem key={conn.metadata?.name}>
                        <Grid hasGutter>
                          <GridItem span={6 as any}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                              {isConnected && <CheckCircleIcon color="green" />}
                              {isFailed && <ExclamationCircleIcon color="red" />}
                              <strong>{conn.spec?.deviceName || conn.spec?.deviceID}</strong>
                            </div>
                            <div style={{ fontSize: '0.875rem', color: '#6a6e73' }}>
                              Device ID: {conn.spec?.deviceID}
                            </div>
                            {conn.status?.message && (
                              <div style={{ fontSize: '0.875rem', color: isFailed ? '#c9190b' : '#6a6e73' }}>
                                {conn.status.message}
                              </div>
                            )}
                          </GridItem>
                          <GridItem span={3 as any}>
                            <span style={{ fontSize: '0.875rem' }}>
                              Status: <strong>{phase || 'Unknown'}</strong>
                            </span>
                          </GridItem>
                          <GridItem span={3 as any} style={{ textAlign: 'right' }}>
                            <Button
                              variant="danger"
                              onClick={() => openDetachModal(conn)}
                            >
                              Detach
                            </Button>
                          </GridItem>
                        </Grid>
                      </ListItem>
                    );
                  })}
                </List>
              )}
            </CardBody>
          </Card>
        </GridItem>

        {/* Attach Device Section */}
        <GridItem span={12 as any}>
          <Card>
            <CardTitle>
              <Title headingLevel="h3">Attach USB Device</Title>
            </CardTitle>
            <CardBody>
              {availableDevices.length === 0 ? (
                <Alert variant="info" title="No USB devices available" isInline>
                  <p>
                    Make sure the workstation agent is running on your computer and USB devices are
                    connected.
                  </p>
                  <p style={{ marginTop: '0.5rem' }}>
                    Visit the workstation agent documentation to learn how to install and run the
                    agent.
                  </p>
                </Alert>
              ) : (
                <Grid hasGutter>
                  <GridItem span={8 as any}>
                    <Select
                      variant={SelectVariant.single}
                      onToggle={setIsSelectOpen}
                      onSelect={(_, value) => {
                        setSelectedDeviceId(value as string);
                        setIsSelectOpen(false);
                      }}
                      selections={selectedDeviceId}
                      isOpen={isSelectOpen}
                      placeholderText="Select a USB device..."
                    >
                      {availableDevices.map((device) => {
                        const deviceId = device.spec?.deviceID;
                        const deviceName = device.spec?.deviceName || deviceId;
                        const isCAC = device.spec?.isCAC;

                        return (
                          <SelectOption
                            key={deviceId}
                            value={deviceId}
                            description={`ID: ${deviceId} | Owner: ${device.spec?.owner || 'Unknown'}`}
                          >
                            {isCAC && '🔒 '}
                            {deviceName}
                            {isCAC && ' (CAC Reader)'}
                          </SelectOption>
                        );
                      })}
                    </Select>
                  </GridItem>
                  <GridItem span={4 as any}>
                    <Button
                      variant="primary"
                      onClick={handleAttach}
                      isDisabled={!selectedDeviceId || isAttaching}
                      isLoading={isAttaching}
                    >
                      {isAttaching ? 'Attaching...' : 'Attach Device'}
                    </Button>
                  </GridItem>
                </Grid>
              )}
            </CardBody>
          </Card>
        </GridItem>
      </Grid>

      {/* Detach Confirmation Modal */}
      <Modal
        variant={ModalVariant.small}
        title="Detach USB Device"
        isOpen={showDetachModal}
        onClose={() => setShowDetachModal(false)}
        actions={[
          <Button key="confirm" variant="danger" onClick={handleDetachConfirm}>
            Detach
          </Button>,
          <Button key="cancel" variant="link" onClick={() => setShowDetachModal(false)}>
            Cancel
          </Button>,
        ]}
      >
        Are you sure you want to detach{' '}
        <strong>{deviceToDetach?.spec?.deviceName || deviceToDetach?.spec?.deviceID}</strong> from
        this VM?
      </Modal>
    </>
  );
};

export default VMUSBTab;
