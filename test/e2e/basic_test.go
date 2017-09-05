package e2e

import (
	"context"
	"testing"

	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/pkg/util/vaultutil"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil/portforwarder"
	"github.com/coreos-inc/vault-operator/test/e2e/framework"

	vaultapi "github.com/hashicorp/vault/api"
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

	vault, err := f.VaultsCRClient.Get(context.TODO(), testVault.Namespace, testVault.Name)
	if err != nil {
		t.Fatalf("failed to get CR: %v", err)
	}

	pf, err := portforwarder.New(f.KubeClient, f.Config)
	if err != nil {
		t.Fatalf("failed to create a portforwarder: %v", err)
	}

	// TODO: Run e2e tests in a container to avoid port conflicts between concurrent test runs
	ports := []string{"8200:8200"}
	podName := vault.Status.AvailableNodes[0]
	if err = pf.StartForwarding(podName, f.Namespace, ports); err != nil {
		t.Fatalf("failed to forward ports to pod(%v): %v", podName, err)
	}

	defer func() {
		if err = pf.StopForwarding(podName, f.Namespace); err != nil {
			t.Errorf("failed to stop forwarding ports to pod(%v): %v", podName, err)
		}
	}()

	tlsConfig, err := k8sutil.VaultTLSFromSecret(f.KubeClient, vault)
	if err != nil {
		t.Fatalf("failed to read TLS config for vault client: %v", err)
	}

	vapi, err := vaultutil.NewClient("localhost", tlsConfig)
	if err != nil {
		t.Fatalf("failed creating client for the vault pod (%s/%s): %v", f.Namespace, podName, err)
	}

	initOpts := &vaultapi.InitRequest{
		SecretShares:    1,
		SecretThreshold: 1,
	}

	_, err = vapi.Sys().Init(initOpts)
	if err != nil {
		t.Fatalf("failed to initialize vault: %v", err)
	}

	// TODO: Wait until node shows up as sealed, then unseal it.
}
