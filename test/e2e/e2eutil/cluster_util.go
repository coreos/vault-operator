package e2eutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/coreos-inc/vault-operator/pkg/client"
	"github.com/coreos-inc/vault-operator/pkg/spec"
)

// CreateCluster creates a vault CR with the desired spec
func CreateCluster(t *testing.T, crClient client.Vaults, cl *spec.Vault) (*spec.Vault, error) {
	vault, err := crClient.Create(context.TODO(), cl)
	if err != nil {
		return nil, fmt.Errorf("failed to create CR: %v", err)
	}
	LogfWithTimestamp(t, "created vault cluster: %s", vault.Name)
	return vault, nil
}

// ResizeCluster updates the Nodes field of the vault CR
func ResizeCluster(t *testing.T, crClient client.Vaults, cl *spec.Vault, size int) (*spec.Vault, error) {
	vault, err := crClient.Get(context.TODO(), cl.Namespace, cl.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get CR: %v", err)
	}
	vault.Spec.Nodes = int32(size)
	vault, err = crClient.Update(context.TODO(), vault)
	if err != nil {
		return nil, fmt.Errorf("failed to update CR: %v", err)
	}
	LogfWithTimestamp(t, "updated vault cluster(%v) to size(%v)", vault.Name, size)
	return vault, nil
}

// DeleteCluster deletes the vault CR specified by cluster spec
func DeleteCluster(t *testing.T, crClient client.Vaults, cl *spec.Vault) error {
	t.Logf("deleting vault cluster: %v", cl.Name)
	err := crClient.Delete(context.TODO(), cl.Namespace, cl.Name)
	if err != nil {
		return fmt.Errorf("failed to delete CR: %v", err)
	}
	// TODO: Wait for cluster resources to be deleted
	return nil
}
