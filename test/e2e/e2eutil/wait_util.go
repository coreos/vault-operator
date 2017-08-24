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

// WaitAvailableVaultsUp waits until the desired number of vault nodes are shown as available in the CR status
func WaitAvailableVaultsUp(t *testing.T, vaultsCRClient client.Vaults, size, retries int, cl *spec.Vault) error {
	err := retryutil.Retry(retryInterval, 6, func() (bool, error) {
		vault, err := vaultsCRClient.Get(context.TODO(), cl.Namespace, cl.Name)
		if err != nil {
			return false, fmt.Errorf("failed to get CR: %v", err)
		}

		LogfWithTimestamp(t, "available nodes: (%v)", vault.Status.AvailableNodes)

		return len(vault.Status.AvailableNodes) == size, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for available size to become (%v): %v", size, err)
	}
	return nil
}
