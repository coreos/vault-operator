package e2e

import (
	"testing"

	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"
	"github.com/coreos-inc/vault-operator/test/e2e/framework"

	vaultapi "github.com/hashicorp/vault/api"
)

func TestScaleUp(t *testing.T) {
	f := framework.Global
	testVault, err := e2eutil.CreateCluster(t, f.VaultsCRClient, e2eutil.NewCluster("test-vault-", f.Namespace, 1))
	if err != nil {
		t.Fatalf("failed to create vault cluster: %v", err)
	}
	defer func() {
		if err := e2eutil.DeleteCluster(t, f.VaultsCRClient, testVault); err != nil {
			t.Fatalf("failed to delete vault cluster: %v", err)
		}
	}()

	vault, err := e2eutil.WaitAvailableVaultsUp(t, f.VaultsCRClient, 1, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for cluster nodes to become available: %v", err)
	}

	tlsConfig, err := k8sutil.VaultTLSFromSecret(f.KubeClient, vault)
	if err != nil {
		t.Fatalf("failed to read TLS config for vault client: %v", err)
	}

	startingConns, err := e2eutil.PortForwardVaultClients(f.KubeClient, f.Config, f.Namespace, tlsConfig, vault.Status.AvailableNodes...)
	if err != nil {
		t.Fatalf("failed to portforward and create vault clients: %v", err)
	}
	defer e2eutil.CleanupConnections(t, f.Namespace, startingConns)

	// Init vault via the first available node
	initOpts := &vaultapi.InitRequest{SecretShares: 1, SecretThreshold: 1}
	podName := vault.Status.AvailableNodes[0]
	conn, ok := startingConns[podName]
	if !ok {
		t.Fatalf("failed to find vault client for pod (%v)", podName)
	}
	initResp, err := conn.VClient.Sys().Init(initOpts)
	if err != nil {
		t.Fatalf("failed to initialize vault: %v", err)
	}

	vault, err = e2eutil.WaitSealedVaultsUp(t, f.VaultsCRClient, 1, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become sealed: %v", err)
	}

	// Unseal the vault node and wait for it to become active
	unsealResp, err := conn.VClient.Sys().Unseal(initResp.Keys[0])
	if err != nil {
		t.Fatalf("failed to unseal vault: %v", err)
	}
	if unsealResp.Sealed {
		t.Fatal("failed to unseal vault: unseal response still shows vault as sealed")
	}
	vault, err = e2eutil.WaitActiveVaultsUp(t, f.VaultsCRClient, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for any node to become active: %v", err)
	}

	// TODO: Write secret to active node, read secret from new node later

	// Resize cluster to 2 nodes
	vault, err = e2eutil.ResizeCluster(t, f.VaultsCRClient, vault, 2)
	if err != nil {
		t.Fatalf("failed to resize vault cluster: %v", err)
	}

	// Wait for 1 unsealed node and create a vault client for it
	vault, err = e2eutil.WaitSealedVaultsUp(t, f.VaultsCRClient, 1, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become sealed: %v", err)
	}
	podName = vault.Status.SealedNodes[0]
	scaledConns, err := e2eutil.PortForwardVaultClients(f.KubeClient, f.Config, f.Namespace, tlsConfig, podName)
	if err != nil {
		t.Fatalf("failed to portforward and create vault clients: %v", err)
	}
	defer e2eutil.CleanupConnections(t, f.Namespace, scaledConns)

	// Unseal the new node and wait for it to become standby
	conn, ok = scaledConns[podName]
	if !ok {
		t.Fatalf("failed to find vault client for pod (%v)", podName)
	}
	unsealResp, err = conn.VClient.Sys().Unseal(initResp.Keys[0])
	if err != nil {
		t.Fatalf("failed to unseal vault: %v", err)
	}
	if unsealResp.Sealed {
		t.Fatal("failed to unseal vault: unseal response still shows vault as sealed")
	}
	vault, err = e2eutil.WaitStandbyVaultsUp(t, f.VaultsCRClient, 1, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become standby: %v", err)
	}

}
