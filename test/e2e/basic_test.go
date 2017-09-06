package e2e

import (
	"fmt"
	"testing"

	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil/portforwarder"
	"github.com/coreos-inc/vault-operator/test/e2e/framework"

	vaultapi "github.com/hashicorp/vault/api"
)

func TestCreateHAVault(t *testing.T) {
	f := framework.Global
	testVault, err := e2eutil.CreateCluster(t, f.VaultsCRClient, e2eutil.NewCluster("test-vault-", f.Namespace, 2))
	if err != nil {
		t.Fatalf("failed to create vault cluster: %v", err)
	}
	defer func() {
		if err := e2eutil.DeleteCluster(t, f.VaultsCRClient, testVault); err != nil {
			t.Fatalf("failed to delete vault cluster: %v", err)
		}
	}()

	vault, err := e2eutil.WaitAvailableVaultsUp(t, f.VaultsCRClient, 2, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for cluster nodes to become available: %v", err)
	}

	pf, err := portforwarder.New(f.KubeClient, f.Config)
	if err != nil {
		t.Fatalf("failed to create a portforwarder: %v", err)
	}

	tlsConfig, err := k8sutil.VaultTLSFromSecret(f.KubeClient, vault)
	if err != nil {
		t.Fatalf("failed to read TLS config for vault client: %v", err)
	}

	// TODO: Run e2e tests in a container to avoid port conflicts between concurrent test runs
	// Portforward to all available vault nodes, and create vault clients for each pod
	vClients := map[string]*vaultapi.Client{}
	if err = portForwardVaultClients(f.Namespace, vClients, tlsConfig, pf, vault.Status.AvailableNodes...); err != nil {
		t.Fatalf("failed to portforward and create vault client: %v", err)
	}
	defer func() {
		pf.StopForwardingAll()
	}()

	// Init vault and wait for sealed nodes
	podName := vault.Status.AvailableNodes[0]
	vClient, ok := vClients[podName]
	if !ok {
		t.Fatalf("failed to find vault client for pod (%v)", podName)
	}
	initResp, err := vClient.Sys().Init(e2eutil.SingleKeyInitOptions())
	if err != nil {
		t.Fatalf("failed to initialize vault: %v", err)
	}
	vault, err = e2eutil.WaitSealedVaultsUp(t, f.VaultsCRClient, 2, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become sealed: %v", err)
	}

	// Unseal the 1st vault node and wait for it to become active
	resp, err := vClient.Sys().Unseal(initResp.Keys[0])
	if err != nil {
		t.Fatalf("failed to unseal vault: %v", err)
	}
	if resp.Sealed {
		t.Fatal("failed to unseal vault: unseal response still shows vault as sealed")
	}
	vault, err = e2eutil.WaitActiveVaultsUp(t, f.VaultsCRClient, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for any node to become active: %v", err)
	}

	// Unseal the 2nd vault node(the remaining sealed node) and wait for it to become standby
	podName = vault.Status.SealedNodes[0]
	vClient, ok = vClients[podName]
	if !ok {
		t.Fatalf("failed to find vault client for pod (%v)", podName)
	}
	resp, err = vClient.Sys().Unseal(initResp.Keys[0])
	if err != nil {
		t.Fatalf("failed to unseal vault: %v", err)
	}
	if resp.Sealed {
		t.Fatal("failed to unseal vault: unseal response still shows vault as sealed")
	}
	vault, err = e2eutil.WaitStandbyVaultsUp(t, f.VaultsCRClient, 1, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become standby: %v", err)
	}

	// TODO: read write to vault cluster

}

// portForwardVaultClients creates a port forwarding session and a vault client for each vault pod.
// The portforwarding is done on localhost X:8200 where X is some ephemeral port allocated for that pod's portforwarding session.
// The vault clients are tracked via the vClients map passsed in.
func portForwardVaultClients(namespace string, vClients map[string]*vaultapi.Client, tlsConfig *vaultapi.TLSConfig, pf portforwarder.PortForwarder, vaultPods ...string) error {
	for _, podName := range vaultPods {
		port := e2eutil.NextPortNumber()
		// TODO: Retry with another port if it fails?
		if err := pf.StartForwarding(podName, namespace, e2eutil.GetPortMapping(port)); err != nil {
			return fmt.Errorf("failed to forward port(%v) to pod(%v): %v", podName, port, err)
		}

		vClient, err := e2eutil.NewVaultClient("localhost", port, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed creating vault client for (localhost:%v): %v", port, err)
		}
		vClients[podName] = vClient
	}
	return nil
}
