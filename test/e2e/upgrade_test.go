package e2e

import (
	"testing"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"
	"github.com/coreos-inc/vault-operator/test/e2e/framework"
)

func TestUpgradeVault(t *testing.T) {
	f := framework.Global
	vaultCR := e2eutil.NewCluster("test-vault-", f.Namespace, 2)
	vaultCR.Spec.Version = "0.9.1-0"
	vaultCR, err := e2eutil.CreateCluster(t, f.VaultsCRClient, vaultCR)
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

	// Initialize vault via the 1st available node
	podName := vaultCR.Status.Nodes.Available[0]
	conn := e2eutil.GetConnOrFail(t, podName, startingConns)
	vaultCR, initResp := e2eutil.InitializeVault(t, f.VaultsCRClient, vaultCR, conn)

	// Unseal the 1st vault node and wait for it to become active
	podName = vaultCR.Status.Nodes.Sealed[0]
	conn = e2eutil.GetConnOrFail(t, podName, startingConns)
	if err := e2eutil.UnsealVaultNode(initResp.Keys[0], conn); err != nil {
		t.Fatalf("failed to unseal vault node(%v): %v", podName, err)
	}
	vaultCR, err = e2eutil.WaitActiveVaultsUp(t, f.VaultsCRClient, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for any node to become active: %v", err)
	}

	// Unseal the 2nd vault node and wait for it to become standby
	podName = vaultCR.Status.Nodes.Sealed[0]
	conn = e2eutil.GetConnOrFail(t, podName, startingConns)
	if err = e2eutil.UnsealVaultNode(initResp.Keys[0], conn); err != nil {
		t.Fatalf("failed to unseal vault node(%v): %v", podName, err)
	}
	vaultCR, err = e2eutil.WaitStandbyVaultsUp(t, f.VaultsCRClient, 1, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become standby: %v", err)
	}

	// Upgrade vault version
	newVersion := "0.9.1-1"
	vaultCR, err = e2eutil.UpdateVersion(t, f.VaultsCRClient, vaultCR, newVersion)
	if err != nil {
		t.Fatalf("failed to update vault version: %v", err)
	}

	// Check for 2 sealed nodes
	vaultCR, err = e2eutil.WaitSealedVaultsUp(t, f.VaultsCRClient, 2, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for updated sealed vault nodes: %v", err)
	}

	// Portforward to and unseal the sealed nodes
	connsUpgraded, err := e2eutil.PortForwardVaultClients(f.KubeClient, f.Config, f.Namespace, tlsConfig, vaultCR.Status.Nodes.Sealed...)
	if err != nil {
		t.Fatalf("failed to portforward and create vault clients: %v", err)
	}
	defer e2eutil.CleanupConnections(t, f.Namespace, connsUpgraded)

	podName = vaultCR.Status.Nodes.Sealed[0]
	conn = e2eutil.GetConnOrFail(t, podName, connsUpgraded)
	if err = e2eutil.UnsealVaultNode(initResp.Keys[0], conn); err != nil {
		t.Fatalf("failed to unseal vault node(%v): %v", podName, err)
	}
	podName = vaultCR.Status.Nodes.Sealed[1]
	conn = e2eutil.GetConnOrFail(t, podName, connsUpgraded)
	if err = e2eutil.UnsealVaultNode(initResp.Keys[0], conn); err != nil {
		t.Fatalf("failed to unseal vault node(%v): %v", podName, err)
	}

	upgradedNodes := vaultCR.Status.Nodes.Sealed

	// Check that the active node is one of the newly unsealed nodes
	vaultCR, err = e2eutil.WaitUntilActiveIsFrom(t, f.VaultsCRClient, 6, vaultCR, upgradedNodes...)
	if err != nil {
		t.Fatalf("failed to see the active node to be from the newly unsealed pods (%v): %v", upgradedNodes, err)
	}

	// Check that the standby node(s) are all from the newly unsealed nodes
	vaultCR, err = e2eutil.WaitUntilStandbyAreFrom(t, f.VaultsCRClient, 6, vaultCR, upgradedNodes...)
	if err != nil {
		t.Fatalf("failed to see all the standby nodes to be from the newly unsealed pods (%v): %v", upgradedNodes, err)
	}

	// Check that the available nodes are all from the newly unsealed nodes, i.e the old nodes are deleted
	vaultCR, err = e2eutil.WaitUntilAvailableAreFrom(t, f.VaultsCRClient, 6, vaultCR, upgradedNodes...)
	if err != nil {
		t.Fatalf("failed to see all available nodes to be from the newly unsealed pods (%v): %v", upgradedNodes, err)
	}

	// Check that 1 active and 1 standby are of the updated version
	err = e2eutil.CheckVersionReached(t, f.KubeClient, newVersion, 6, vaultCR, vaultCR.Status.Nodes.Active)
	if err != nil {
		t.Fatalf("failed to wait for active node to become updated: %v", err)
	}
	err = e2eutil.CheckVersionReached(t, f.KubeClient, newVersion, 6, vaultCR, vaultCR.Status.Nodes.Standby...)
	if err != nil {
		t.Fatalf("failed to wait for standby nodes to become updated: %v", err)
	}
}
