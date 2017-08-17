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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var (
	// VaultConfigPath is the path that vault pod uses to read config from
	VaultConfigPath = "/run/vault/config/vault.hcl"
	// VaultTLSAssetDir is the dir where vault's server TLS and etcd TLS assets sits
	VaultTLSAssetDir = "/run/vault/tls/"

	// vault image format is "<upstream-version>-<our-version"
	vaultImage           = "quay.io/coreos/vault:0.8.0-0"
	vaultTLSAssetVolume  = "vault-tls-secret"
	vaultConfigVolName   = "vault-config"
	evnVaultRedirectAddr = "VAULT_REDIRECT_ADDR"
)

// DefaultVaultClientTLSSecretName returns the name of the default vault client TLS secret
func DefaultVaultClientTLSSecretName(vaultName string) string {
	return vaultName + "-default-vault-client-tls"
}

// DefaultVaultServerTLSSecretName returns the name of the default vault server TLS secret
func DefaultVaultServerTLSSecretName(vaultName string) string {
	return vaultName + "-default-vault-server-tls"
}

// EtcdClientTLSSecretName returns the name of etcd client TLS secret for the given vault name
func EtcdClientTLSSecretName(vaultName string) string {
	return vaultName + "-etcd-client-tls"
}

// EtcdServerTLSSecretName returns the name of etcd server TLS secret for the given vault name
func EtcdServerTLSSecretName(vaultName string) string {
	return vaultName + "-etcd-server-tls"
}

// EtcdPeerTLSSecretName returns the name of etcd peer TLS secret for the given vault name
func EtcdPeerTLSSecretName(vaultName string) string {
	return vaultName + "-etcd-peer-tls"
}

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
			Name:      EtcdNameForVault(v.Name),
			Namespace: v.Namespace,
		},
		Spec: etcdCRAPI.ClusterSpec{
			Size: size,
			TLS: &etcdCRAPI.TLSPolicy{
				Static: &etcdCRAPI.StaticTLS{
					Member: &etcdCRAPI.MemberSecret{
						PeerSecret:   EtcdPeerTLSSecretName(v.Name),
						ServerSecret: EtcdServerTLSSecretName(v.Name),
					},
					OperatorSecret: EtcdClientTLSSecretName(v.Name),
				},
			},
		},
	}
	_, err := etcdCRCli.Create(context.TODO(), etcdCluster)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("deploy etcd cluster failed: %v", err)
	}

	err = retryutil.Retry(10*time.Second, 10, func() (bool, error) {
		er, err := etcdCRCli.Get(context.TODO(), v.Namespace, EtcdNameForVault(v.Name))
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
	err := etcdCRCli.Delete(context.TODO(), v.Namespace, EtcdNameForVault(v.Name))
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

	selector := LabelsForVault(v.GetName())
	vaultPort := 8200

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
				Env: []v1.EnvVar{
					{
						Name:  evnVaultRedirectAddr,
						Value: VaultServiceURL(v.GetName(), v.GetNamespace()),
					},
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
					ContainerPort: int32(vaultPort),
				}},
				LivenessProbe: &v1.Probe{
					Handler: v1.Handler{
						Exec: &v1.ExecAction{
							Command: []string{
								"curl",
								"--connect-timeout", "5",
								"--max-time", "10",
								"-k", "-s",
								fmt.Sprintf("https://localhost:%d/v1/sys/health", vaultPort),
							},
						},
					},
					InitialDelaySeconds: 10,
					TimeoutSeconds:      10,
					PeriodSeconds:       60,
					FailureThreshold:    3,
				},
				ReadinessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path:   "/v1/sys/health",
							Port:   intstr.FromInt(vaultPort),
							Scheme: v1.URISchemeHTTPS,
						},
					},
					InitialDelaySeconds: 10,
					TimeoutSeconds:      10,
					PeriodSeconds:       10,
					FailureThreshold:    3,
				},
			}},
			Volumes: []v1.Volume{{
				Name: vaultConfigVolName,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: ConfigMapNameForVault(v),
						},
					},
				},
			}},
		},
	}

	configEtcdBackendTLS(&podTempl, v)
	configVaultServerTLS(&podTempl, v)

	d := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   v.GetName(),
			Labels: selector,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Replicas: &v.Spec.Nodes,
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: podTempl,
			Strategy: appsv1beta1.DeploymentStrategy{
				Type: appsv1beta1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1beta1.RollingUpdateDeployment{
					MaxUnavailable: func(a intstr.IntOrString) *intstr.IntOrString { return &a }(intstr.FromInt(1)),
					MaxSurge:       func(a intstr.IntOrString) *intstr.IntOrString { return &a }(intstr.FromInt(1)),
				},
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
		return fmt.Errorf("failed to create vault service: %v", err)
	}
	return nil
}

// UpdateVault updates a vault service.
func UpdateVault(kubecli kubernetes.Interface, oldV *spec.Vault, newV *spec.Vault) error {
	ns, name := oldV.GetNamespace(), oldV.GetName()
	d, err := kubecli.AppsV1beta1().Deployments(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if oldV.Spec.Nodes != newV.Spec.Nodes {
		d.Spec.Replicas = &(newV.Spec.Nodes)
		_, err = kubecli.AppsV1beta1().Deployments(ns).Update(d)
		if err != nil {
			return err
		}
	}

	return nil
}

// ConfigMapNameForVault is the configmap name for the given vault.
// If ConfigMapName is given is spec, it will make a new name based on that.
// Otherwise, we will create a default configmap using the Vault's name.
func ConfigMapNameForVault(v *spec.Vault) string {
	n := v.Spec.ConfigMapName
	if len(n) == 0 {
		n = v.Name
	}
	return n + "-copy"
}

// VaultServiceURL returns the DNS record of the vault service in the given namespace.
func VaultServiceURL(name, namespace string) string {
	return fmt.Sprintf("https://%s.%s.svc:8200", name, namespace)
}

// DestroyVault destroys a vault service.
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

// EtcdNameForVault returns the etcd cluster's name for the given vault's name
func EtcdNameForVault(name string) string {
	return name + "-etcd"
}

// EtcdURLForVault returns the URL to talk to etcd cluster for the given vault's name
func EtcdURLForVault(name string) string {
	return fmt.Sprintf("https://%s-client:2379", EtcdNameForVault(name))
}

// LabelsForVault returns the labels for selecting the resources
// belonging to the given vault name.
func LabelsForVault(name string) map[string]string {
	return map[string]string{"app": "vault", "name": name}
}

// configEtcdBackendTLS configures the volume and mounts in vault pod to
// set up etcd backend TLS assets
func configEtcdBackendTLS(pt *v1.PodTemplateSpec, v *spec.Vault) {
	sn := EtcdClientTLSSecretName(v.Name)
	pt.Spec.Volumes = append(pt.Spec.Volumes, v1.Volume{
		Name: vaultTLSAssetVolume,
		VolumeSource: v1.VolumeSource{
			Projected: &v1.ProjectedVolumeSource{
				Sources: []v1.VolumeProjection{{
					Secret: &v1.SecretProjection{
						LocalObjectReference: v1.LocalObjectReference{
							Name: sn,
						},
					},
				}},
			},
		},
	})
	pt.Spec.Containers[0].VolumeMounts = append(pt.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
		Name:      vaultTLSAssetVolume,
		ReadOnly:  true,
		MountPath: VaultTLSAssetDir,
	})
}

// configVaultServerTLS mounts the volume containing the vault server TLS assets for the vault pod
func configVaultServerTLS(pt *v1.PodTemplateSpec, v *spec.Vault) {
	secretName := DefaultVaultServerTLSSecretName(v.Name)
	if spec.IsTLSConfigured(v.Spec.TLS) {
		secretName = v.Spec.TLS.Static.ServerSecret
	}

	serverTLSVolume := v1.VolumeProjection{
		Secret: &v1.SecretProjection{
			LocalObjectReference: v1.LocalObjectReference{
				Name: secretName,
			},
		},
	}
	pt.Spec.Volumes[1].VolumeSource.Projected.Sources = append(pt.Spec.Volumes[1].VolumeSource.Projected.Sources, serverTLSVolume)
}
