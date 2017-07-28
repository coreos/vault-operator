package k8sutil

import (
	"path/filepath"

	"github.com/coreos-inc/vault-operator/pkg/spec"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	vaultImage         = "vault"
	vaultConfigVolName = "vault-config"
	vaultConfigPath    = "/run/vault-config/vault.hcl"
)

// DeployVault deploys a vault service.
// DeployVault is a multi-steps process. It creates the deployment, the service and
// other related Kubernetes objects for Vault. Any intermediate step can fail.
//
// DeployVault is idempotent. If an object already exists, this function will ignore creating
// it and return no error. It is safe to retry on this function.
func DeployVault(kubecli kubernetes.Interface, v *spec.Vault) error {
	// TODO: set owner ref.

	selector := map[string]string{"app": "vault", "name": v.GetName()}

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
					"-config=" + vaultConfigPath,
				},
				VolumeMounts: []v1.VolumeMount{{
					Name:      vaultConfigVolName,
					MountPath: filepath.Dir(vaultConfigPath),
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
							Name: v.Spec.ConfigMapName,
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
