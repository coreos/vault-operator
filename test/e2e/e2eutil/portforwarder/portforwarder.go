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
}

type portForwarder struct {
	kubeClient kubernetes.Interface
	transport  http.RoundTripper
	upgrader   spdy.Upgrader
}

// New returns a PortForwarder that uses client-go's implementation of the httpstream.Dialer interface
// See https://github.com/kubernetes/client-go/blob/master/transport/spdy/spdy.go
func New(kubeClient kubernetes.Interface, config *restclient.Config) (PortForwarder, error) {
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get roundtripper: %v", err)
	}
	return &portForwarder{
		kubeClient: kubeClient,
		transport:  transport,
		upgrader:   upgrader,
	}, nil
}

func (pf *portForwarder) StartForwarding(podName, namespace string, ports []string) error {
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

	return nil
}

func getNamespacedName(name, namespace string) string {
	return path.Join(namespace, name)
}
