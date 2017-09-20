package e2eutil

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/coreos-inc/vault-operator/pkg/util/vaultutil"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil/portforwarder"

	vaultapi "github.com/hashicorp/vault/api"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

const (
	// Ephemeral port ranges
	ephemeralPortLowerBound = 30000
	ephemeralPortUpperBound = 60000
	targetVaultPort         = "8200"
)

// Connection is used to pair a vault client to a portforwarding session
type Connection struct {
	VClient *vaultapi.Client
	PF      portforwarder.PortForwarder
}

var (
	// Atomic counter used to assign increasing port numbers
	portCounter int64
)

func init() {
	portCounter = ephemeralPortLowerBound
}

// NextPortNumber returns the next port number in an atomically increasing sequence in the ephemeral port range
func NextPortNumber() int64 {
	return atomic.AddInt64(&portCounter, 1)
}

// PortForwardVaultClients creates a port forwarding session and a vault client for each vault pod.
// The portforwarding is done on localhost X:8200 where X is some ephemeral port allocated for that pod's portforwarding session.
// For each pod the vault-client and it's respective portforward session are tracked via the map of Connections returned.
func PortForwardVaultClients(kubeClient kubernetes.Interface, config *restclient.Config, namespace string, tlsConfig *vaultapi.TLSConfig, vaultPods ...string) (map[string]*Connection, error) {
	connections := map[string]*Connection{}
	for _, podName := range vaultPods {
		pf, err := portforwarder.New(kubeClient, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create a portforwarder: %v", err)
		}
		port := strconv.FormatInt(NextPortNumber(), 10)
		portMapping := []string{port + ":" + targetVaultPort}
		// TODO: Retry with another port if it fails?
		if err := pf.StartForwarding(podName, namespace, portMapping); err != nil {
			return nil, fmt.Errorf("failed to forward port(%v) to pod(%v): %v", podName, port, err)
		}

		vClient, err := vaultutil.NewClient("localhost", port, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed creating vault client for (localhost:%v): %v", port, err)
		}

		connections[podName] = &Connection{
			VClient: vClient,
			PF:      pf,
		}
	}
	return connections, nil
}

// CleanupConnections stops forwarding all connections present in conns
func CleanupConnections(t *testing.T, namespace string, conns map[string]*Connection) {
	for podName, conn := range conns {
		if err := conn.PF.StopForwarding(podName, namespace); err != nil {
			t.Errorf("failed to stop port forwarding to pod(%v): %v", podName, err)
		}
	}
}
