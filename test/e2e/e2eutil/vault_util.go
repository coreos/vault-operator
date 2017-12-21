package e2eutil

import (
	"fmt"
	"testing"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos-inc/vault-operator/pkg/generated/clientset/versioned"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	vaultapi "github.com/hashicorp/vault/api"
	"k8s.io/client-go/kubernetes"
)

// WaitForCluster waits for all available nodes of a cluster to appear in the vault CR status
// Returns the updated vault cluster and the TLS configuration to use for vault clients interacting with the cluster
func WaitForCluster(t *testing.T, kubeClient kubernetes.Interface, vaultsCRClient versioned.Interface, vaultCR *api.VaultService) (*api.VaultService, *vaultapi.TLSConfig) {
	// Based on local testing, it took about ~50s for a normal deployment to finish.
	vaultCR, err := WaitAvailableVaultsUp(t, vaultsCRClient, int(vaultCR.Spec.Nodes), 10, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for cluster nodes to become available: %v", err)
	}

	tlsConfig, err := k8sutil.VaultTLSFromSecret(kubeClient, vaultCR)
	if err != nil {
		t.Fatalf("failed to read TLS config for vault client: %v", err)
	}
	return vaultCR, tlsConfig
}

// InitializeVault initializes the specified vault cluster and waits for all available nodes to appear as sealed.
// Requires established portforwarded connections to the vault pods
// Returns the updated vault cluster and the initialization response which includes the unseal key
func InitializeVault(t *testing.T, vaultsCRClient versioned.Interface, vault *api.VaultService, conn *Connection) (*api.VaultService, *vaultapi.InitResponse) {
	initOpts := &vaultapi.InitRequest{SecretShares: 1, SecretThreshold: 1}
	initResp, err := conn.VClient.Sys().Init(initOpts)
	if err != nil {
		t.Fatalf("failed to initialize vault: %v", err)
	}
	// Wait until initialized nodes to be reflected on status.nodes.Sealed
	vault, err = WaitSealedVaultsUp(t, vaultsCRClient, int(vault.Spec.Nodes), 6, vault)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become sealed: %v", err)
	}
	return vault, initResp
}

// UnsealVaultNode unseals the specified vault pod by portforwarding to it via its vault client
func UnsealVaultNode(unsealKey string, conn *Connection) error {
	unsealResp, err := conn.VClient.Sys().Unseal(unsealKey)
	if err != nil {
		return fmt.Errorf("failed to unseal vault: %v", err)
	}
	if unsealResp.Sealed {
		return fmt.Errorf("failed to unseal vault: unseal response still shows vault as sealed")
	}
	return nil
}
