package e2e

import (
	"testing"

	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"
	"github.com/coreos-inc/vault-operator/test/e2e/framework"
)

func TestCreateCluster(t *testing.T) {
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

	err = e2eutil.WaitAvailableVaultsUp(t, f.VaultsCRClient, 1, 6, testVault)
	if err != nil {
		t.Fatalf("failed to wait for cluster nodes to become available: %v", err)
	}

	// TODO: Init and unseal pod, then read/write to vault service
}
