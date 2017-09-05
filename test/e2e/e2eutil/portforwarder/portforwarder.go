package portforwarder

import (
	"fmt"
	"net/http"
	"path"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwarder interface {
	// StartForwarding starts the client-go portforwarder to listen and forward ports to the specified pod
	// Each port string maps a local port to the target pod's port and is of the format: "<local-port>:<pod-port>"
	StartForwarding(podName, namespace string, ports []string) error
	// StopForwarding stops the client-go portforwarder from forwarding ports to the specified pod
	StopForwarding(podName, namespace string) error
	// StopForwardingAll stops portforwarding to all active pods
	StopForwardingAll()
}

// connection is used to keep track the info for an active portforwarding connection
type connection struct {
	podName   string
	namespace string
	ports     []string
	stopChan  chan struct{}
}

type portForwarder struct {
	activeConnections map[string]connection
	kubeClient        kubernetes.Interface
	transport         http.RoundTripper
	upgrader          spdy.Upgrader
}

// New returns a PortForwarder that uses client-go's implementation of the httpstream.Dialer interface
// See https://github.com/kubernetes/client-go/blob/master/transport/spdy/spdy.go
func New(kubeClient kubernetes.Interface, config *restclient.Config) (PortForwarder, error) {
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get roundtripper: %v", err)
	}
	return &portForwarder{
		activeConnections: map[string]connection{},
		kubeClient:        kubeClient,
		transport:         transport,
		upgrader:          upgrader,
	}, nil
}

func (pf *portForwarder) StartForwarding(podName, namespace string, ports []string) error {
	podFullName := getNamespacedName(podName, namespace)
	if conn, ok := pf.activeConnections[podFullName]; ok {
		return fmt.Errorf("Already forwarding ports (%v) to the pod (%v). Stop that first", conn.ports, podFullName)
	}

	url := pf.kubeClient.CoreV1().RESTClient().Post().Resource("pods").Namespace(namespace).Name(podName).SubResource("portforward").URL()

	dialer := spdy.NewDialer(pf.upgrader, &http.Client{Transport: pf.transport}, "POST", url)
	stopChan := make(chan struct{})
	readyChan := make(chan struct{})
	errChan := make(chan error)

	k8sPF, err := portforward.New(dialer, ports, stopChan, readyChan, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create a : %v", err)
	}

	go func() {
		errChan <- k8sPF.ForwardPorts()
	}()

	select {
	case err = <-errChan:
		return fmt.Errorf("failed to forward ports: %v", err)
	case <-readyChan:
	}

	pf.activeConnections[podFullName] = connection{
		podName:   podName,
		namespace: namespace,
		ports:     ports,
		stopChan:  stopChan,
	}
	return nil
}

func (pf *portForwarder) StopForwarding(podName, namespace string) error {
	podFullName := getNamespacedName(podName, namespace)
	conn, ok := pf.activeConnections[podFullName]
	if !ok {
		return fmt.Errorf("No ports being forwarded to the pod (%v)", podFullName)
	}

	// Stop the client-go port forwarder for this pod
	close(conn.stopChan)
	delete(pf.activeConnections, podFullName)
	return nil
}

// StopForwardingAll stops portforwarding to all active pods
func (pf *portForwarder) StopForwardingAll() {
	for _, conn := range pf.activeConnections {
		close(conn.stopChan)
	}
	pf.activeConnections = map[string]connection{}
}

func getNamespacedName(name, namespace string) string {
	return path.Join(namespace, name)
}
