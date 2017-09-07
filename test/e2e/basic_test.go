package e2e

import (
	"testing"

	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"
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

	tlsConfig, err := k8sutil.VaultTLSFromSecret(f.KubeClient, vault)
	if err != nil {
		t.Fatalf("failed to read TLS config for vault client: %v", err)
	}

	// TODO: Run e2e tests in a container to avoid port conflicts between concurrent test runs
	conns := map[string]*e2eutil.Connection{}
	if err = e2eutil.PortForwardVaultClients(f.KubeClient, f.Config, f.Namespace, conns, tlsConfig, vault.Status.AvailableNodes...); err != nil {
		t.Fatalf("failed to portforward and create vault clients: %v", err)
	}
	defer func() {
		for podName, conn := range conns {
			if err = conn.PF.StopForwarding(podName, f.Namespace); err != nil {
				t.Errorf("failed to stop port forwarding to pod(%v): %v", podName, err)
			}
		}
	}()

	initOpts := &vaultapi.InitRequest{
		SecretShares:    1,
		SecretThreshold: 1,
	}

	// Init vault via the first available node
	podName := vault.Status.AvailableNodes[0]
	conn, ok := conns[podName]
	if !ok {
		t.Fatalf("failed to find vault client for pod (%v)", podName)
	}
	_, err = conn.VClient.Sys().Init(initOpts)
	if err != nil {
		t.Fatalf("failed to initialize vault: %v", err)
	}

	vault, err = e2eutil.WaitSealedVaultsUp(t, f.VaultsCRClient, 2, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become sealed: %v", err)
	}

	// TODO: Unseal both vault nodes and wait for 1 active and 1 standby node
}
