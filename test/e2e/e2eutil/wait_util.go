package e2eutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/coreos-inc/vault-operator/pkg/client"
	"github.com/coreos-inc/vault-operator/pkg/spec"

	"github.com/coreos/etcd-operator/pkg/util/k8sutil"
	"github.com/coreos/etcd-operator/pkg/util/retryutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Retry interval used for all retries in wait related functions
var retryInterval = 10 * time.Second

// checkConditionFunc is used to check if a condition for the vault CR is true
type checkConditionFunc func(*spec.Vault) bool

// WaitUntilOperatorReady will wait until the first pod with the label name=<name> is ready.
func WaitUntilOperatorReady(kubecli kubernetes.Interface, namespace, name string) error {
	var podName string
	lo := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(PodLabelForOperator(name)).String(),
	}
	err := retryutil.Retry(retryInterval, 6, func() (bool, error) {
		podList, err := kubecli.CoreV1().Pods(namespace).List(lo)
		if err != nil {
			return false, err
		}
		if len(podList.Items) > 0 {
			podName = podList.Items[0].Name
			if k8sutil.IsPodReady(&podList.Items[0]) {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for pod (%v) to become ready: %v", podName, err)
	}
	return nil
}

// WaitUntilVaultConditionTrue retries until the specified condition check becomes true for the vault CR
func WaitUntilVaultConditionTrue(t *testing.T, vaultsCRClient client.Vaults, retries int, cl *spec.Vault, checkCondition checkConditionFunc) (*spec.Vault, error) {
	var vault *spec.Vault
	var err error
	err = retryutil.Retry(retryInterval, 6, func() (bool, error) {
		vault, err = vaultsCRClient.Get(context.TODO(), cl.Namespace, cl.Name)
		if err != nil {
			return false, fmt.Errorf("failed to get CR: %v", err)
		}
		return checkCondition(vault), nil
	})
	if err != nil {
		return nil, err
	}
	return vault, nil
}

// WaitAvailableVaultsUp retries until the desired number of vault nodes are shown as available in the CR status
func WaitAvailableVaultsUp(t *testing.T, vaultsCRClient client.Vaults, size, retries int, cl *spec.Vault) (*spec.Vault, error) {
	vault, err := WaitUntilVaultConditionTrue(t, vaultsCRClient, retries, cl, func(v *spec.Vault) bool {
		LogfWithTimestamp(t, "available nodes: (%v)", v.Status.AvailableNodes)
		return len(v.Status.AvailableNodes) == size
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for available size to become (%v): %v", size, err)
	}
	return vault, nil
}

// WaitSealedVaultsUp retries until the desired number of vault nodes are shown as sealed in the CR status
func WaitSealedVaultsUp(t *testing.T, vaultsCRClient client.Vaults, size, retries int, cl *spec.Vault) (*spec.Vault, error) {
	vault, err := WaitUntilVaultConditionTrue(t, vaultsCRClient, retries, cl, func(v *spec.Vault) bool {
		LogfWithTimestamp(t, "sealed nodes: (%v)", v.Status.SealedNodes)
		return len(v.Status.SealedNodes) == size
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for sealed size to become (%v): %v", size, err)
	}
	return vault, nil
}

// WaitStandbyVaultsUp retries until the desired number of vault nodes are shown as standby in the CR status
func WaitStandbyVaultsUp(t *testing.T, vaultsCRClient client.Vaults, size, retries int, cl *spec.Vault) (*spec.Vault, error) {
	vault, err := WaitUntilVaultConditionTrue(t, vaultsCRClient, retries, cl, func(v *spec.Vault) bool {
		LogfWithTimestamp(t, "standby nodes: (%v)", v.Status.StandbyNodes)
		return len(v.Status.StandbyNodes) == size
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for standby size to become (%v): %v", size, err)
	}
	return vault, nil
}

// WaitActiveVaultsUp retries until there is 1 active node in the CR status
func WaitActiveVaultsUp(t *testing.T, vaultsCRClient client.Vaults, retries int, cl *spec.Vault) (*spec.Vault, error) {
	vault, err := WaitUntilVaultConditionTrue(t, vaultsCRClient, retries, cl, func(v *spec.Vault) bool {
		LogfWithTimestamp(t, "active node: (%v)", v.Status.ActiveNode)
		return len(v.Status.ActiveNode) != 0
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for any node to become active: %v", err)
	}
	return vault, nil
}
