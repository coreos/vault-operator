package e2eutil

import (
	"fmt"
	"strings"
	"testing"
	"time"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos-inc/vault-operator/pkg/generated/clientset/versioned"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"github.com/coreos/etcd-operator/pkg/util/retryutil"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Retry interval used for all retries in wait related functions
var retryInterval = 10 * time.Second

// checkConditionFunc is used to check if a condition for the vault CR is true
type checkConditionFunc func(*api.VaultService) bool

// filterFunc returns true if the pod matches some condition defined by filterFunc
type filterFunc func(*v1.Pod) bool

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
			if k8sutil.IsPodReady(podList.Items[0]) {
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
func WaitUntilVaultConditionTrue(t *testing.T, vaultsCRClient versioned.Interface, retries int, vs *api.VaultService, checkCondition checkConditionFunc) (*api.VaultService, error) {
	var vault *api.VaultService
	var err error
	err = retryutil.Retry(retryInterval, retries, func() (bool, error) {
		vault, err = vaultsCRClient.VaultV1alpha1().VaultServices(vs.Namespace).Get(vs.Name, metav1.GetOptions{})
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
func WaitAvailableVaultsUp(t *testing.T, vaultsCRClient versioned.Interface, size, retries int, vs *api.VaultService) (*api.VaultService, error) {
	vault, err := WaitUntilVaultConditionTrue(t, vaultsCRClient, retries, vs, func(v *api.VaultService) bool {
		available := getAvailableNodes(v.Status.VaultStatus)
		LogfWithTimestamp(t, "available nodes: (%v)", available)
		return len(available) == size
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for available size to become (%v): %v", size, err)
	}
	return vault, nil
}

// WaitSealedVaultsUp retries until the desired number of vault nodes are shown as sealed in the CR status
func WaitSealedVaultsUp(t *testing.T, vaultsCRClient versioned.Interface, size, retries int, vs *api.VaultService) (*api.VaultService, error) {
	vault, err := WaitUntilVaultConditionTrue(t, vaultsCRClient, retries, vs, func(v *api.VaultService) bool {
		LogfWithTimestamp(t, "sealed nodes: (%v)", v.Status.VaultStatus.Sealed)
		return len(v.Status.VaultStatus.Sealed) == size
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for sealed size to become (%v): %v", size, err)
	}
	return vault, nil
}

// WaitStandbyVaultsUp retries until the desired number of vault nodes are shown as standby in the CR status
func WaitStandbyVaultsUp(t *testing.T, vaultsCRClient versioned.Interface, size, retries int, vs *api.VaultService) (*api.VaultService, error) {
	vault, err := WaitUntilVaultConditionTrue(t, vaultsCRClient, retries, vs, func(v *api.VaultService) bool {
		LogfWithTimestamp(t, "standby nodes: (%v)", v.Status.VaultStatus.Standby)
		return len(v.Status.VaultStatus.Standby) == size
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for standby size to become (%v): %v", size, err)
	}
	return vault, nil
}

// WaitActiveVaultsUp retries until there is 1 active node in the CR status
func WaitActiveVaultsUp(t *testing.T, vaultsCRClient versioned.Interface, retries int, vs *api.VaultService) (*api.VaultService, error) {
	vault, err := WaitUntilVaultConditionTrue(t, vaultsCRClient, retries, vs, func(v *api.VaultService) bool {
		LogfWithTimestamp(t, "active node: (%v)", v.Status.VaultStatus.Active)
		return len(v.Status.VaultStatus.Active) != 0
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for any node to become active: %v", err)
	}
	return vault, nil
}

// WaitPodsDeletedCompletely waits until the pods are completely removed(not just terminating) for the given label selector
func WaitPodsDeletedCompletely(kubecli kubernetes.Interface, namespace string, retries int, lo metav1.ListOptions) ([]*v1.Pod, error) {
	return waitPodsDeleted(kubecli, namespace, retries, lo)
}

// WaitPodsWithImageDeleted waits until the pods with the specified image and labels are removed
func WaitPodsWithImageDeleted(kubecli kubernetes.Interface, namespace, image string, retries int, lo metav1.ListOptions) ([]*v1.Pod, error) {
	return waitPodsDeleted(kubecli, namespace, retries, lo, func(p *v1.Pod) bool {
		for _, c := range p.Spec.Containers {
			if c.Image == image {
				return false
			}
		}
		return true
	})
}

// waitPodsDeleted waits until the pods selected by the desired label selector and passing the filter conditions are completely removed
func waitPodsDeleted(kubecli kubernetes.Interface, namespace string, retries int, lo metav1.ListOptions, filters ...filterFunc) ([]*v1.Pod, error) {
	var pods []*v1.Pod
	err := retryutil.Retry(retryInterval, retries, func() (bool, error) {
		podList, err := kubecli.CoreV1().Pods(namespace).List(lo)
		if err != nil {
			return false, fmt.Errorf("failed to list pods: %v", err)
		}
		pods = nil
		for i := range podList.Items {
			p := &podList.Items[i]
			filtered := false
			for _, filter := range filters {
				if filter(p) {
					filtered = true
				}
			}
			if !filtered {
				pods = append(pods, p)
			}
		}
		return len(pods) == 0, nil
	})
	return pods, err
}

// CheckVersionReached checks if all the targetVaultPods are of the specified version
func CheckVersionReached(t *testing.T, kubeClient kubernetes.Interface, version string, retries int, vs *api.VaultService, targetVaultPods ...string) error {
	size := len(targetVaultPods)
	var names []string
	sel := k8sutil.LabelsForVault(vs.Name)
	opt := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(sel).String()}
	podList, err := kubeClient.Core().Pods(vs.Namespace).List(opt)
	if err != nil {
		return err
	}
	names = nil
	for i := range podList.Items {
		pod := &podList.Items[i]
		if !presentIn(pod.Name, targetVaultPods...) {
			continue
		}

		containerVersion := getVersionFromImage(pod.Spec.Containers[0].Image)
		if containerVersion != version {
			LogfWithTimestamp(t, "pod(%v): expected version(%v) current version(%v)", pod.Name, version, containerVersion)
			continue
		}

		names = append(names, pod.Name)
	}

	if len(names) != size {
		return fmt.Errorf("failed to see target pods(%v) update to version (%v): currently updated (%v)", targetVaultPods, size, names)
	}
	return nil
}

func presentIn(a string, list ...string) bool {
	for _, l := range list {
		if a == l {
			return true
		}
	}
	return false
}

func getVersionFromImage(image string) string {
	return strings.Split(image, ":")[1]
}

// WaitUntilActiveIsFrom waits until the active node is from one of the target pods
func WaitUntilActiveIsFrom(t *testing.T, vaultsCRClient versioned.Interface, retries int, vs *api.VaultService, targetVaultPods ...string) (*api.VaultService, error) {
	var vault *api.VaultService
	var err error
	// TODO: refactor WaitXXX func on VaultService to apply generic condition
	err = retryutil.Retry(retryInterval, retries, func() (bool, error) {
		vault, err = vaultsCRClient.VaultV1alpha1().VaultServices(vs.Namespace).Get(vs.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get CR: %v", err)
		}
		LogfWithTimestamp(t, "active node: (%v)", vault.Status.VaultStatus.Active)
		if !presentIn(vault.Status.VaultStatus.Active, targetVaultPods...) {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return vault, nil
}

// WaitUntilStandbyAreFrom waits until all the standby nodes are from the target pods
func WaitUntilStandbyAreFrom(t *testing.T, vaultsCRClient versioned.Interface, retries int, vs *api.VaultService, targetVaultPods ...string) (*api.VaultService, error) {
	var vault *api.VaultService
	var err error
	err = retryutil.Retry(retryInterval, retries, func() (bool, error) {
		vault, err = vaultsCRClient.VaultV1alpha1().VaultServices(vs.Namespace).Get(vs.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get CR: %v", err)
		}
		LogfWithTimestamp(t, "standby nodes: (%v)", vault.Status.VaultStatus.Standby)
		for _, standbyNode := range vault.Status.VaultStatus.Standby {
			if !presentIn(standbyNode, targetVaultPods...) {
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return vault, nil
}

// WaitUntilAvailableAreFrom waits until all the available nodes are from the target pods
func WaitUntilAvailableAreFrom(t *testing.T, vaultsCRClient versioned.Interface, retries int, vs *api.VaultService, targetVaultPods ...string) (*api.VaultService, error) {
	var vault *api.VaultService
	var err error
	err = retryutil.Retry(retryInterval, retries, func() (bool, error) {
		vault, err = vaultsCRClient.VaultV1alpha1().VaultServices(vs.Namespace).Get(vs.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get CR: %v", err)
		}
		available := getAvailableNodes(vault.Status.VaultStatus)
		LogfWithTimestamp(t, "available nodes: (%v)", available)
		for _, availableNode := range available {
			if !presentIn(availableNode, targetVaultPods...) {
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return vault, nil
}

func getAvailableNodes(vs api.VaultStatus) []string {
	var available []string
	if len(vs.Active) != 0 {
		available = append(available, vs.Active)
	}
	available = append(available, vs.Sealed...)
	available = append(available, vs.Standby...)
	return available
}
