package e2eutil

import (
	"fmt"
	"reflect"
	"testing"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos-inc/vault-operator/pkg/generated/clientset/versioned"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/pkg/util/vaultutil"

	vaultapi "github.com/hashicorp/vault/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const targetVaultPort = "8200"

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
func InitializeVault(t *testing.T, vaultsCRClient versioned.Interface, vault *api.VaultService, vClient *vaultapi.Client) (*api.VaultService, *vaultapi.InitResponse) {
	initOpts := &vaultapi.InitRequest{SecretShares: 1, SecretThreshold: 1}
	initResp, err := vClient.Sys().Init(initOpts)
	if err != nil {
		t.Fatalf("failed to initialize vault: %v", err)
	}
	// Wait until initialized nodes to be reflected on status.vaultStatus.Sealed
	vault, err = WaitSealedVaultsUp(t, vaultsCRClient, int(vault.Spec.Nodes), 6, vault)
	if err != nil {
		t.Fatalf("failed to wait for vault nodes to become sealed: %v", err)
	}
	return vault, initResp
}

// UnsealVaultNode unseals the specified vault pod by portforwarding to it via its vault client
func UnsealVaultNode(unsealKey string, vClient *vaultapi.Client) error {
	unsealResp, err := vClient.Sys().Unseal(unsealKey)
	if err != nil {
		return fmt.Errorf("failed to unseal vault: %v", err)
	}
	if unsealResp.Sealed {
		return fmt.Errorf("failed to unseal vault: unseal response still shows vault as sealed")
	}
	return nil
}

// SetupVaultClient creates a vault client for the specified pod
func SetupVaultClient(t *testing.T, kubeClient kubernetes.Interface, namespace string, tlsConfig *vaultapi.TLSConfig, podName string) *vaultapi.Client {
	pod, err := kubeClient.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("fail to get vault pod (%s): %v", podName, err)
	}
	vClient, err := vaultutil.NewClient(k8sutil.PodDNSName(*pod), targetVaultPort, tlsConfig)
	if err != nil {
		t.Fatalf("failed creating vault client for (localhost:%v): %v", targetVaultPort, err)
	}
	return vClient
}

// SetupUnsealedVaultCluster initializes a vault cluster and unseals the 1st vault node.
func SetupUnsealedVaultCluster(t *testing.T, kubeClient kubernetes.Interface, vaultsCRClient versioned.Interface, namespace string) (*api.VaultService, *vaultapi.TLSConfig, string) {
	vaultCR, err := CreateCluster(t, vaultsCRClient, NewCluster("test-vault-", namespace, 2))
	if err != nil {
		t.Fatalf("failed to create vault cluster: %v", err)
	}

	vaultCR, tlsConfig := WaitForCluster(t, kubeClient, vaultsCRClient, vaultCR)

	// Init vault via the first sealed node
	podName := vaultCR.Status.VaultStatus.Sealed[0]
	vClient := SetupVaultClient(t, kubeClient, namespace, tlsConfig, podName)
	vaultCR, initResp := InitializeVault(t, vaultsCRClient, vaultCR, vClient)

	// Unseal the 1st vault node and wait for it to become active
	podName = vaultCR.Status.VaultStatus.Sealed[0]
	vClient = SetupVaultClient(t, kubeClient, namespace, tlsConfig, podName)
	if err := UnsealVaultNode(initResp.Keys[0], vClient); err != nil {
		t.Fatalf("failed to unseal vault node(%v): %v", podName, err)
	}
	vaultCR, err = WaitActiveVaultsUp(t, vaultsCRClient, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for any node to become active: %v", err)
	}

	return vaultCR, tlsConfig, initResp.RootToken
}

// WriteSecretData writes secret data into vault.
func WriteSecretData(t *testing.T, vaultCR *api.VaultService, kubeClient kubernetes.Interface, tlsConfig *vaultapi.TLSConfig, rootToken, namespace string) (*vaultapi.Client, string, map[string]interface{}, string) {
	// Write secret to active node
	podName := vaultCR.Status.VaultStatus.Active
	vClient := SetupVaultClient(t, kubeClient, namespace, tlsConfig, podName)
	vClient.SetToken(rootToken)

	keyPath := "secret/login"
	data := &SampleSecret{Username: "user", Password: "pass"}
	secretData, err := MapObjectToArbitraryData(data)
	if err != nil {
		t.Fatalf("failed to create secret data (%+v): %v", data, err)
	}

	_, err = vClient.Logical().Write(keyPath, secretData)
	if err != nil {
		t.Fatalf("failed to write secret (%v) to vault node (%v): %v", keyPath, podName, err)
	}
	return vClient, keyPath, secretData, podName
}

// VerifySecretData gets secret of the "keyPath" and compares it against the given secretData.
func VerifySecretData(t *testing.T, vClient *vaultapi.Client, secretData map[string]interface{}, keyPath, podName string) {
	// Read secret back from active node
	secret, err := vClient.Logical().Read(keyPath)
	if err != nil {
		t.Fatalf("failed to read secret(%v) from vault node (%v): %v", keyPath, podName, err)
	}

	if !reflect.DeepEqual(secret.Data, secretData) {
		t.Fatalf("Read secret data (%+v) is not the same as written secret (%+v)", secret.Data, secretData)
	}
}
