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

interface VMUSBTabProps {
  obj?: any; // VirtualMachine or VirtualMachineInstance object
}

interface USBDevice {
  id: string;
  name: string;
  vendorProduct: string;
  vendor: string;
  product: string;
  serial: string;
  isCAC: boolean;
  owner: string;
}

interface USBConnection {
  id: string;
  deviceId: string;
  deviceName: string;
  vmName: string;
  namespace: string;
  status: string;
  message?: string;
  startedAt: string;
}

const AGENT_API_URL = 'http://localhost:8080';

const VMUSBTab: React.FC<VMUSBTabProps> = (props) => {
  const { obj } = props || {};
  const [isSelectOpen, setIsSelectOpen] = React.useState(false);
  const [selectedDeviceId, setSelectedDeviceId] = React.useState<string>('');
  const [isAttaching, setIsAttaching] = React.useState(false);
  const [error, setError] = React.useState<string>('');
  const [showDetachModal, setShowDetachModal] = React.useState(false);
  const [deviceToDetach, setDeviceToDetach] = React.useState<USBConnection | null>(null);

  const [devices, setDevices] = React.useState<USBDevice[]>([]);
  const [connections, setConnections] = React.useState<USBConnection[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [agentConnected, setAgentConnected] = React.useState(false);

  const vmName = obj?.metadata?.name;
  const vmNamespace = obj?.metadata?.namespace;
  const vmStatus = obj?.status?.printableStatus || obj?.status?.phase || 'Unknown';
  const isVMRunning = vmStatus === 'Running';

  // Fetch devices from local agent
  const fetchDevices = React.useCallback(async () => {
    try {
      const response = await fetch(`${AGENT_API_URL}/devices`);
      if (!response.ok) throw new Error('Failed to fetch devices');
      const data = await response.json();
      setDevices(data || []);
      setAgentConnected(true);
      setError('');
    } catch (err: any) {
      console.error('Failed to fetch devices:', err);
      setAgentConnected(false);
      if (!error) {
        setError('Cannot connect to workstation agent. Make sure it is running on localhost:8080');
      }
    }
  }, [error]);

  // Fetch connections from local agent
  const fetchConnections = React.useCallback(async () => {
    try {
      const response = await fetch(`${AGENT_API_URL}/connections`);
      if (!response.ok) throw new Error('Failed to fetch connections');
      const data = await response.json();
      setConnections(data || []);
    } catch (err: any) {
      console.error('Failed to fetch connections:', err);
    }
  }, []);

  // Initial fetch and polling
  React.useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      await Promise.all([fetchDevices(), fetchConnections()]);
      setLoading(false);
    };

    fetchData();

    // Poll every 3 seconds
    const interval = setInterval(() => {
      fetchDevices();
      fetchConnections();
    }, 3000);

    return () => clearInterval(interval);
  }, [fetchDevices, fetchConnections]);

  const vmConnections = React.useMemo(() => {
    return connections.filter(
      (conn) => conn.vmName === vmName && conn.namespace === vmNamespace
    );
  }, [connections, vmName, vmNamespace]);

  const handleAttach = async () => {
    if (!selectedDeviceId) return;

    setIsAttaching(true);
    setError('');

    try {
      const selectedDevice = devices.find((dev) => dev.id === selectedDeviceId);

      if (!selectedDevice) {
        throw new Error('Selected device not found');
      }

      const response = await fetch(`${AGENT_API_URL}/attach`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          deviceId: selectedDeviceId,
          deviceName: selectedDevice.name,
          vmName: vmName,
          namespace: vmNamespace,
        }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'Failed to attach device');
      }

      setSelectedDeviceId('');
      setIsSelectOpen(false);

      // Refresh connections
      await fetchConnections();
    } catch (err: any) {
      setError(err.message || 'Failed to attach USB device');
    } finally {
      setIsAttaching(false);
    }
  };

  const handleDetachConfirm = async () => {
    if (!deviceToDetach) return;

    try {
      const response = await fetch(`${AGENT_API_URL}/detach/${deviceToDetach.id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new Error('Failed to detach device');
      }

      setShowDetachModal(false);
      setDeviceToDetach(null);

      // Refresh connections
      await fetchConnections();
    } catch (err: any) {
      setError(err.message || 'Failed to detach USB device');
    }
  };

  const openDetachModal = (connection: USBConnection) => {
    setDeviceToDetach(connection);
    setShowDetachModal(true);
  };

  if (loading) {
    return (
      <EmptyState>
        <EmptyStateIcon variant="container" component={Spinner} />
        <Title headingLevel="h2" size="lg">
          Loading USB devices...
        </Title>
      </EmptyState>
    );
  }

  if (!agentConnected) {
    return (
      <Alert variant="warning" title="Workstation Agent Not Running" isInline>
        <p>
          The USB Passthrough workstation agent is not running on your computer.
        </p>
        <p style={{ marginTop: '0.5rem' }}>
          To use USB device passthrough, you need to:
        </p>
        <ol style={{ marginTop: '0.5rem', marginLeft: '1.5rem' }}>
          <li>
            <a
              href="https://github.com/andykrohg/ocp-virt-usb-passthrough-plugin"
              target="_blank"
              rel="noopener noreferrer"
            >
              Download the workstation agent
            </a> and run it on your local machine
          </li>
          <li>Ensure it's listening on port 8080</li>
        </ol>
        <p style={{ marginTop: '0.5rem', fontSize: '0.875rem', color: '#6a6e73' }}>
          This page will automatically detect the agent when it starts.
        </p>
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
                    const isConnected = conn.status === 'Connected';
                    const isFailed = conn.status === 'Failed';

                    return (
                      <ListItem key={conn.id}>
                        <Grid hasGutter>
                          <GridItem span={6 as any}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                              {isConnected && <CheckCircleIcon color="green" />}
                              {isFailed && <ExclamationCircleIcon color="red" />}
                              <strong>{conn.deviceName || conn.deviceId}</strong>
                            </div>
                            <div style={{ fontSize: '0.875rem', color: '#6a6e73' }}>
                              Device ID: {conn.deviceId}
                            </div>
                            {conn.message && (
                              <div style={{ fontSize: '0.875rem', color: isFailed ? '#c9190b' : '#6a6e73' }}>
                                {conn.message}
                              </div>
                            )}
                          </GridItem>
                          <GridItem span={3 as any}>
                            <span style={{ fontSize: '0.875rem' }}>
                              Status: <strong>{conn.status || 'Unknown'}</strong>
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
              {!isVMRunning ? (
                <Alert variant="warning" title="VM is not running" isInline>
                  <p>
                    USB devices can only be attached to running VMs.
                  </p>
                  <p style={{ marginTop: '0.5rem' }}>
                    Current VM status: <strong>{vmStatus}</strong>
                  </p>
                  <p style={{ marginTop: '0.5rem' }}>
                    Please start the VM before attaching USB devices.
                  </p>
                </Alert>
              ) : devices.length === 0 ? (
                <Alert variant="info" title="No USB devices available" isInline>
                  <p>
                    No USB devices detected on your workstation.
                  </p>
                  <p style={{ marginTop: '0.5rem' }}>
                    Make sure USB devices are connected to your computer and the workstation agent
                    is running.
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
                      isDisabled={!isVMRunning}
                      placeholderText="Select a USB device..."
                    >
                      {devices.map((device) => {
                        return (
                          <SelectOption
                            key={device.id}
                            value={device.id}
                            description={`ID: ${device.vendorProduct} | Owner: ${device.owner}`}
                          >
                            {device.isCAC && '🔒 '}
                            {device.name}
                            {device.isCAC && ' (CAC Reader)'}
                          </SelectOption>
                        );
                      })}
                    </Select>
                  </GridItem>
                  <GridItem span={4 as any}>
                    <Button
                      variant="primary"
                      onClick={handleAttach}
                      isDisabled={!selectedDeviceId || isAttaching || !isVMRunning}
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
        <strong>{deviceToDetach?.deviceName || deviceToDetach?.deviceId}</strong> from
        this VM?
      </Modal>
    </>
  );
};

export default VMUSBTab;
