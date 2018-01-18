package e2e

import (
	"reflect"
	"testing"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"
	"github.com/coreos-inc/vault-operator/test/e2e/framework"
)

func TestCreateHAVault(t *testing.T) {
	f := framework.Global
	vaultCR, err := e2eutil.CreateCluster(t, f.VaultsCRClient, e2eutil.NewCluster("test-vault-", f.Namespace, 2))
	if err != nil {
		t.Fatalf("failed to create vault cluster: %v", err)
	}
	defer func(vaultCR *api.VaultService) {
		if err := e2eutil.DeleteCluster(t, f.VaultsCRClient, vaultCR); err != nil {
			t.Fatalf("failed to delete vault cluster: %v", err)
		}
	}(vaultCR)

	vaultCR, tlsConfig := e2eutil.WaitForCluster(t, f.KubeClient, f.VaultsCRClient, vaultCR)

	// Init vault via the first available node
	podName := vaultCR.Status.Nodes.Available[0]
	vClient := e2eutil.SetupVaultClient(t, f.KubeClient, f.Namespace, tlsConfig, podName)
	vaultCR, initResp := e2eutil.InitializeVault(t, f.VaultsCRClient, vaultCR, vClient)

	// Unseal the 1st vault node and wait for it to become active
	podName = vaultCR.Status.Nodes.Sealed[0]
	vClient = e2eutil.SetupVaultClient(t, f.KubeClient, f.Namespace, tlsConfig, podName)
	if err := e2eutil.UnsealVaultNode(initResp.Keys[0], vClient); err != nil {
		t.Fatalf("failed to unseal vault node(%v): %v", podName, err)
	}
	vaultCR, err = e2eutil.WaitActiveVaultsUp(t, f.VaultsCRClient, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for any node to become active: %v", err)
	}

	// Unseal the 2nd vault node(the remaining sealed node) and wait for it to become standby
	podName = vaultCR.Status.Nodes.Sealed[0]
	vClient = e2eutil.SetupVaultClient(t, f.KubeClient, f.Namespace, tlsConfig, podName)
	if err := e2eutil.UnsealVaultNode(initResp.Keys[0], vClient); err != nil {
		t.Fatalf("failed to unseal vault node(%v): %v", podName, err)
	}
	vaultCR, err = e2eutil.WaitStandbyVaultsUp(t, f.VaultsCRClient, 1, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become standby: %v", err)
	}

	// Write secret to active node
	podName = vaultCR.Status.Nodes.Active
	vClient = e2eutil.SetupVaultClient(t, f.KubeClient, f.Namespace, tlsConfig, podName)
	vClient.SetToken(initResp.RootToken)

	path := "secret/login"
	data := &e2eutil.SampleSecret{Username: "user", Password: "pass"}
	secretData, err := e2eutil.MapObjectToArbitraryData(data)
	if err != nil {
		t.Fatalf("failed to create secret data: %v", err)
	}

	_, err = vClient.Logical().Write(path, secretData)
	if err != nil {
		t.Fatalf("failed to write secret (%v) to vault node (%v): %v", path, podName, err)
	}

	// Read secret back from active node
	secret, err := vClient.Logical().Read(path)
	if err != nil {
		t.Fatalf("failed to read secret(%v) from vault node (%v): %v", path, podName, err)
	}

	if !reflect.DeepEqual(secret.Data, secretData) {
		// TODO: Print out secrets
		t.Fatal("Read secret data is not the same as write secret")
	}

}
