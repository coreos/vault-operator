package k8sutil

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/coreos-inc/vault-operator/pkg/spec"

	etcdCRClient "github.com/coreos/etcd-operator/pkg/client"
	etcdCRAPI "github.com/coreos/etcd-operator/pkg/spec"
	"github.com/coreos/etcd-operator/pkg/util/retryutil"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	// VaultConfigPath is the path that vault pod uses to read config from
	VaultConfigPath = "/run/vault-config/vault.hcl"

	vaultImage         = "vault"
	vaultConfigVolName = "vault-config"
)

// DeployEtcdCluster creates an etcd cluster for the given vault's name via etcd operator and
// waits for all of its members to be ready.
func DeployEtcdCluster(etcdCRCli etcdCRClient.EtcdClusterCR, v *spec.Vault) error {
	size := 3
	etcdCluster := &etcdCRAPI.EtcdCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       etcdCRAPI.CRDResourceKind,
			APIVersion: etcdCRAPI.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcdNameForVault(v.Name),
			Namespace: v.Namespace,
		},
		Spec: etcdCRAPI.ClusterSpec{
			Size: size,
		},
	}
	_, err := etcdCRCli.Create(context.TODO(), etcdCluster)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("deploy etcd cluster failed: %v", err)
	}

	err = retryutil.Retry(10*time.Second, 10, func() (bool, error) {
		er, err := etcdCRCli.Get(context.TODO(), v.Namespace, etcdNameForVault(v.Name))
		if err != nil {
			return false, err
		}
		if len(er.Status.Members.Ready) < size {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("deploy etcd cluster failed: %v", err)
	}
	return nil
}

// DeleteEtcdCluster deletes the etcd cluster for the given vault
func DeleteEtcdCluster(etcdCRCli etcdCRClient.EtcdClusterCR, v *spec.Vault) error {
	err := etcdCRCli.Delete(context.TODO(), v.Namespace, etcdNameForVault(v.Name))
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// DeployVault deploys a vault service.
// DeployVault is a multi-steps process. It creates the deployment, the service and
// other related Kubernetes objects for Vault. Any intermediate step can fail.
//
// DeployVault is idempotent. If an object already exists, this function will ignore creating
// it and return no error. It is safe to retry on this function.
func DeployVault(kubecli kubernetes.Interface, v *spec.Vault) error {
	// TODO: set owner ref.

	selector := PodsLabelsForVault(v.GetName())

	podTempl := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:   v.GetName(),
			Labels: selector,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  "vault",
				Image: vaultImage,
				Command: []string{
					"/bin/vault",
					"server",
					"-config=" + VaultConfigPath,
				},
				VolumeMounts: []v1.VolumeMount{{
					Name:      vaultConfigVolName,
					MountPath: filepath.Dir(VaultConfigPath),
				}},
				SecurityContext: &v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						// Vault requires mlock syscall to work.
						// Without this it would fail "Error initializing core: Failed to lock memory: cannot allocate memory"
						Add: []v1.Capability{"IPC_LOCK"},
					},
				},
				Ports: []v1.ContainerPort{{
					ContainerPort: int32(8200),
				}},
			}},
			Volumes: []v1.Volume{{
				Name: vaultConfigVolName,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: ConfigMapCopyName(v.Spec.ConfigMapName),
						},
					},
				},
			}},
		},
	}

	d := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   v.GetName(),
			Labels: selector,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: podTempl,
			Strategy: appsv1beta1.DeploymentStrategy{
				Type: appsv1beta1.RecreateDeploymentStrategyType,
			},
		},
	}
	_, err := kubecli.AppsV1beta1().Deployments(v.Namespace).Create(d)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   v.Name,
			Labels: selector,
		},
		Spec: v1.ServiceSpec{
			Selector: selector,
			Ports: []v1.ServicePort{{
				Name:     "vault",
				Protocol: v1.ProtocolTCP,
				Port:     8200,
			}},
		},
	}

	_, err = kubecli.CoreV1().Services(v.Namespace).Create(svc)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// ConfigMapCopyName is the configmap name to use by vault pod.
// It's a copy of user given configmap because we modify user config.
func ConfigMapCopyName(n string) string {
	return n + "-copy"
}

// VaultServiceAddr returns the DNS record of the vault service in the given namespace.
func VaultServiceAddr(name, namespace string) string {
	// TODO: change this to https
	return "http://" + name + "." + namespace + ":8200"
}

// DestroVault destroys a vault service.
// TODO: remove this function when CRD GC is enabled.
func DestroyVault(kubecli kubernetes.Interface, v *spec.Vault) error {
	bg := metav1.DeletePropagationBackground
	do := &metav1.DeleteOptions{PropagationPolicy: &bg}

	ns, n := v.GetNamespace(), v.GetName()
	err := kubecli.AppsV1beta1().
		Deployments(ns).
		Delete(n, do)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = kubecli.CoreV1().Services(ns).Delete(n, do)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// etcdNameForVault returns the etcd cluster's name for the given vault's name
func etcdNameForVault(name string) string {
	return name + "-etcd"
}

// EtcdURLForVault returns the URL to talk to etcd cluster for the given vault's name
func EtcdURLForVault(name string) string {
	return fmt.Sprintf("http://%s-client:2379", etcdNameForVault(name))
}

// PodsLabelsForVault returns the labels for selecting the pods belongs to the given vault
// name.
func PodsLabelsForVault(name string) map[string]string {
	return map[string]string{"app": "vault", "name": name}
}
