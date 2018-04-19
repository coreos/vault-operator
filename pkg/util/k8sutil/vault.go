// Copyright 2018 The vault-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8sutil

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	api "github.com/coreos/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos/vault-operator/pkg/util/vaultutil"

	etcdCRAPI "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	etcdCRClient "github.com/coreos/etcd-operator/pkg/generated/clientset/versioned"
	"github.com/coreos/etcd-operator/pkg/util/retryutil"
	vaultapi "github.com/hashicorp/vault/api"
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

	vaultTLSAssetVolume  = "vault-tls-secret"
	vaultConfigVolName   = "vault-config"
	evnVaultRedirectAddr = "VAULT_API_ADDR"
	evnVaultClusterAddr  = "VAULT_CLUSTER_ADDR"
)

const (
	VaultClientPort      = 8200
	vaultClusterPort     = 8201
	vaultClientPortName  = "vault-client"
	vaultClusterPortName = "vault-cluster"

	exporterStatsdPort = 9125
	exporterPromPort   = 9102
	exporterImage      = "prom/statsd-exporter:v0.5.0"
)

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
func DeployEtcdCluster(etcdCRCli etcdCRClient.Interface, v *api.VaultService) error {
	size := 3
	etcdCluster := &etcdCRAPI.EtcdCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       etcdCRAPI.EtcdClusterResourceKind,
			APIVersion: etcdCRAPI.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdNameForVault(v.Name),
			Namespace: v.Namespace,
			Labels:    LabelsForVault(v.Name),
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
			Pod: &etcdCRAPI.PodPolicy{
				EtcdEnv: []v1.EnvVar{{
					Name:  "ETCD_AUTO_COMPACTION_RETENTION",
					Value: "1",
				}},
			},
		},
	}
	if v.Spec.Pod != nil {
		etcdCluster.Spec.Pod.Resources = v.Spec.Pod.Resources
	}
	AddOwnerRefToObject(etcdCluster, AsOwner(v))
	_, err := etcdCRCli.EtcdV1beta2().EtcdClusters(v.Namespace).Create(etcdCluster)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("deploy etcd cluster failed: %v", err)
	}

	err = retryutil.Retry(10*time.Second, 10, func() (bool, error) {
		er, err := etcdCRCli.EtcdV1beta2().EtcdClusters(v.Namespace).Get(EtcdNameForVault(v.Name), metav1.GetOptions{})
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
func DeleteEtcdCluster(etcdCRCli etcdCRClient.Interface, v *api.VaultService) error {
	err := etcdCRCli.EtcdV1beta2().EtcdClusters(v.Namespace).Delete(EtcdNameForVault(v.Name), nil)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func vaultContainer(v *api.VaultService) v1.Container {
	return v1.Container{
		Name:  "vault",
		Image: fmt.Sprintf("%s:%s", v.Spec.BaseImage, v.Spec.Version),
		Command: []string{
			"/bin/vault",
			"server",
			"-config=" + VaultConfigPath,
		},
		Env: []v1.EnvVar{
			{
				Name:  evnVaultRedirectAddr,
				Value: VaultServiceURL(v.GetName(), v.GetNamespace(), VaultClientPort),
			},
			{
				Name:  evnVaultClusterAddr,
				Value: VaultServiceURL(v.GetName(), v.GetNamespace(), vaultClusterPort),
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
			Name:          vaultClientPortName,
			ContainerPort: int32(VaultClientPort),
		}, {
			Name:          vaultClusterPortName,
			ContainerPort: int32(vaultClusterPort),
		}},
		LivenessProbe: &v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{
						"curl",
						"--connect-timeout", "5",
						"--max-time", "10",
						"-k", "-s",
						fmt.Sprintf("https://localhost:%d/v1/sys/health", VaultClientPort),
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
					Port:   intstr.FromInt(VaultClientPort),
					Scheme: v1.URISchemeHTTPS,
				},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			FailureThreshold:    3,
		},
	}
}

func statsdExporterContainer() v1.Container {
	return v1.Container{
		Name:  "statsd-exporter",
		Image: exporterImage,
		Ports: []v1.ContainerPort{{
			Name:          "statsd",
			ContainerPort: exporterStatsdPort,
			Protocol:      "UDP",
		}, {
			Name:          "prometheus",
			ContainerPort: exporterPromPort,
			Protocol:      "TCP",
		}},
	}
}

// DeployVault deploys a vault service.
// DeployVault is a multi-steps process. It creates the deployment, the service and
// other related Kubernetes objects for Vault. Any intermediate step can fail.
//
// DeployVault is idempotent. If an object already exists, this function will ignore creating
// it and return no error. It is safe to retry on this function.
func DeployVault(kubecli kubernetes.Interface, v *api.VaultService) error {
	selector := LabelsForVault(v.GetName())

	podTempl := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:   v.GetName(),
			Labels: selector,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{vaultContainer(v), statsdExporterContainer()},
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
	if v.Spec.Pod != nil {
		applyPodPolicy(&podTempl.Spec, v.Spec.Pod)
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
	AddOwnerRefToObject(d, AsOwner(v))
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
			Ports: []v1.ServicePort{
				{
					Name:     vaultClientPortName,
					Protocol: v1.ProtocolTCP,
					Port:     VaultClientPort,
				},
				{
					Name:     vaultClusterPortName,
					Protocol: v1.ProtocolTCP,
					Port:     vaultClusterPort,
				},
				{
					Name:     "prometheus",
					Protocol: v1.ProtocolTCP,
					Port:     exporterPromPort,
				},
			},
		},
	}
	AddOwnerRefToObject(svc, AsOwner(v))
	_, err = kubecli.CoreV1().Services(v.Namespace).Create(svc)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create vault service: %v", err)
	}
	return nil
}

// UpgradeDeployment sets deployment spec to:
// - roll forward version
// - keep active Vault node available by setting `maxUnavailable=N-1` and `maxSurge=1`
func UpgradeDeployment(kubecli kubernetes.Interface, vr *api.VaultService, d *appsv1beta1.Deployment) error {
	mu := intstr.FromInt(int(vr.Spec.Nodes - 1))
	d.Spec.Strategy.RollingUpdate.MaxUnavailable = &mu
	d.Spec.Template.Spec.Containers[0].Image = vaultImage(vr.Spec)
	_, err := kubecli.AppsV1beta1().Deployments(d.Namespace).Update(d)
	if err != nil {
		return fmt.Errorf("failed to upgrade deployment to (%s): %v", vaultImage(vr.Spec), err)
	}
	return nil
}

func applyPodPolicy(s *v1.PodSpec, p *api.PodPolicy) {
	for i := range s.Containers {
		s.Containers[i].Resources = p.Resources
	}

	for i := range s.InitContainers {
		s.InitContainers[i].Resources = p.Resources
	}
}

func vaultImage(vs api.VaultServiceSpec) string {
	return fmt.Sprintf("%s:%s", vs.BaseImage, vs.Version)
}

func IsVaultVersionMatch(ps v1.PodSpec, vs api.VaultServiceSpec) bool {
	return ps.Containers[0].Image == vaultImage(vs)
}

// VaultTLSFromSecret reads Vault CR's TLS secret and converts it into a vault client's TLS config struct.
func VaultTLSFromSecret(kubecli kubernetes.Interface, vr *api.VaultService) (*vaultapi.TLSConfig, error) {
	secretName := vr.Spec.TLS.Static.ClientSecret

	secret, err := kubecli.CoreV1().Secrets(vr.GetNamespace()).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("read client tls failed: failed to get secret (%s): %v", secretName, err)
	}

	// Read the secret and write ca.crt to a temporary file
	caCertData := secret.Data[api.CATLSCertName]
	f, err := ioutil.TempFile("", api.CATLSCertName)
	if err != nil {
		return nil, fmt.Errorf("read client tls failed: create temp file failed: %v", err)
	}
	defer f.Close()

	_, err = f.Write(caCertData)
	if err != nil {
		return nil, fmt.Errorf("read client tls failed: write ca cert file failed: %v", err)
	}
	if err = f.Sync(); err != nil {
		return nil, fmt.Errorf("read client tls failed: sync ca cert file failed: %v", err)
	}
	return &vaultapi.TLSConfig{CACert: f.Name()}, nil
}

// IsPodReady checks the status of the pod for the Ready condition
func IsPodReady(p v1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == v1.PodReady {
			return c.Status == v1.ConditionTrue
		}
	}
	return false
}

// ConfigMapNameForVault is the configmap name for the given vault.
// If ConfigMapName is given is spec, it will make a new name based on that.
// Otherwise, we will create a default configmap using the Vault's name.
func ConfigMapNameForVault(v *api.VaultService) string {
	n := v.Spec.ConfigMapName
	if len(n) == 0 {
		n = v.Name
	}
	return n + "-copy"
}

// VaultServiceURL returns the DNS record of the vault service in the given namespace.
func VaultServiceURL(name, namespace string, port int) string {
	return fmt.Sprintf("https://%s.%s.svc:%d", name, namespace, port)
}

// DestroyVault destroys a vault service.
// TODO: remove this function when CRD GC is enabled.
func DestroyVault(kubecli kubernetes.Interface, v *api.VaultService) error {
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
	return map[string]string{"app": "vault", "vault_cluster": name}
}

// configEtcdBackendTLS configures the volume and mounts in vault pod to
// set up etcd backend TLS assets
func configEtcdBackendTLS(pt *v1.PodTemplateSpec, v *api.VaultService) {
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
		MountPath: vaultutil.VaultTLSAssetDir,
	})
}

// configVaultServerTLS mounts the volume containing the vault server TLS assets for the vault pod
func configVaultServerTLS(pt *v1.PodTemplateSpec, v *api.VaultService) {
	secretName := v.Spec.TLS.Static.ServerSecret

	serverTLSVolume := v1.VolumeProjection{
		Secret: &v1.SecretProjection{
			LocalObjectReference: v1.LocalObjectReference{
				Name: secretName,
			},
		},
	}
	pt.Spec.Volumes[1].VolumeSource.Projected.Sources = append(pt.Spec.Volumes[1].VolumeSource.Projected.Sources, serverTLSVolume)
}
