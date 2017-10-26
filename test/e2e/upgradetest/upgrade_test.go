package upgradetest

import (
	"fmt"
	"math/rand"
	"testing"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"
	"github.com/coreos-inc/vault-operator/test/e2e/upgradetest/framework"
)

func newOperatorName() string {
	suffix := fmt.Sprintf("-%d", rand.Uint64())
	return "vault-operator" + suffix
}

func TestUpgradeAndScaleVault(t *testing.T) {
	f := framework.Global
	name := newOperatorName()
	err := f.CreateOperatorDeployment(name)
	if err != nil {
		t.Fatalf("failed to create vault operator: %v", err)
	}
	defer func() {
		err := f.DeleteOperatorDeployment(name)
		if err != nil {
			t.Fatalf("failed to delete vault operator: %v", err)
		}
	}()
	if err = e2eutil.WaitUntilOperatorReady(f.KubeClient, f.Namespace, name); err != nil {
		t.Fatalf("failed to wait for operator to become ready: %v", err)
	}

	vaultCR, err := e2eutil.CreateCluster(t, f.VaultsCRClient, e2eutil.NewCluster("upgradetest-vault-", f.Namespace, 1))
	if err != nil {
		t.Fatalf("failed to create vault cluster: %v", err)
	}
	defer func(vaultCR *api.VaultService) {
		if err := e2eutil.DeleteCluster(t, f.VaultsCRClient, vaultCR); err != nil {
			t.Fatalf("failed to delete vault cluster: %v", err)
		}
	}(vaultCR)
	vaultCR, tlsConfig := e2eutil.WaitForCluster(t, f.KubeClient, f.VaultsCRClient, vaultCR)

	startingConns, err := e2eutil.PortForwardVaultClients(f.KubeClient, f.Config, f.Namespace, tlsConfig, vaultCR.Status.Nodes.Available...)
	if err != nil {
		t.Fatalf("failed to portforward and create vault clients: %v", err)
	}
	defer e2eutil.CleanupConnections(t, f.Namespace, startingConns)

	// Init vault via the first available node
	podName := vaultCR.Status.Nodes.Available[0]
	conn := e2eutil.GetConnOrFail(t, podName, startingConns)
	vaultCR, initResp := e2eutil.InitializeVault(t, f.VaultsCRClient, vaultCR, conn)

	// Unseal the vault node and wait for it to become active
	podName = vaultCR.Status.Nodes.Sealed[0]
	conn = e2eutil.GetConnOrFail(t, podName, startingConns)
	if err := e2eutil.UnsealVaultNode(initResp.Keys[0], conn); err != nil {
		t.Fatalf("failed to unseal vault node(%v): %v", podName, err)
	}
	vaultCR, err = e2eutil.WaitActiveVaultsUp(t, f.VaultsCRClient, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for any node to become active: %v", err)
	}

	if err = f.UpgradeOperator(name); err != nil {
		t.Fatalf("failed to upgrade operator: %v", err)
	}

	// Resize cluster to 2 nodes
	vaultCR, err = e2eutil.ResizeCluster(t, f.VaultsCRClient, vaultCR, 2)
	if err != nil {
		t.Fatalf("failed to resize vault cluster: %v", err)
	}

	// Wait for 1 unsealed node and create a vault client for it
	vaultCR, err = e2eutil.WaitSealedVaultsUp(t, f.VaultsCRClient, 1, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become sealed: %v", err)
	}
	podName = vaultCR.Status.Nodes.Sealed[0]
	scaledConns, err := e2eutil.PortForwardVaultClients(f.KubeClient, f.Config, f.Namespace, tlsConfig, podName)
	if err != nil {
		t.Fatalf("failed to portforward and create vault clients: %v", err)
	}
	defer e2eutil.CleanupConnections(t, f.Namespace, scaledConns)

	// Unseal the new node and wait for it to become standby
	conn = e2eutil.GetConnOrFail(t, podName, scaledConns)
	if err := e2eutil.UnsealVaultNode(initResp.Keys[0], conn); err != nil {
		t.Fatalf("failed to unseal vault node(%v): %v", podName, err)
	}
	vaultCR, err = e2eutil.WaitStandbyVaultsUp(t, f.VaultsCRClient, 1, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become standby: %v", err)
	}
}
