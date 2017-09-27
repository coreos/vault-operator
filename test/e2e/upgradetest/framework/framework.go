package framework

import (
	"flag"
	"fmt"

	"github.com/coreos-inc/vault-operator/pkg/client"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/pkg/util/probe"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"

	"github.com/Sirupsen/logrus"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// The global framework variable used by all upgrade tests
	Global           *Framework
	etcdOperatorName = "etcd-operator"
)

// Framework struct contains the various clients and other information needed to run the upgrade tests
type Framework struct {
	KubeClient     kubernetes.Interface
	Config         *restclient.Config
	VaultsCRClient client.Vaults
	Namespace      string
	oldVOPImage    string
	newVOPImage    string
	eopImage       string
}

// Setup initializes the Global framework by initializing necessary clients and sets up the etcd-operator
func Setup() error {
	kubeconfig := flag.String("kubeconfig", "", "kube config path, e.g. $HOME/.kube/config")
	oldVOPImage := flag.String("old-vop-image", "", "operator image, e.g. quay.io/coreos/vault-operator-dev:latest")
	newVOPImage := flag.String("new-vop-image", "", "operator image, e.g. quay.io/coreos/vault-operator-dev:master")
	eopImage := flag.String("etcd-operator-image", "quay.io/coreos/etcd-operator:latest", "etcd operator image, e.g. quay.io/coreos/etcd-operator:latest")
	ns := flag.String("namespace", "", "e2e test namespace")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build config from kubeconfig: %v", err)
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("faild to create kube client: %v", err)
	}
	vaultsCRClient, err := client.NewCRClient(config)
	if err != nil {
		return fmt.Errorf("failed to create CR client: %v", err)
	}

	Global = &Framework{
		KubeClient:     kubeClient,
		Config:         config,
		VaultsCRClient: vaultsCRClient,
		Namespace:      *ns,
		oldVOPImage:    *oldVOPImage,
		newVOPImage:    *newVOPImage,
		eopImage:       *eopImage,
	}

	if err := Global.deployEtcdOperatorPod(); err != nil {
		return fmt.Errorf("failed to setup etcd operator: %v", err)
	}
	return nil
}

// TearDown removes the etcd-operator pod
func TearDown() error {
	err := Global.KubeClient.CoreV1().Pods(Global.Namespace).Delete(etcdOperatorName, k8sutil.CascadeDeleteBackground())
	if err != nil {
		return fmt.Errorf("failed to delete pod: %v", err)
	}
	Global = nil
	logrus.Info("e2e teardown successfully")
	return nil
}

// CreateOperatorDeployment creates a vault operator deployment with the specified name
func (f *Framework) CreateOperatorDeployment(name string) error {
	d := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Strategy: appsv1beta1.DeploymentStrategy{
				Type: appsv1beta1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1beta1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
			Selector: &metav1.LabelSelector{MatchLabels: e2eutil.PodLabelForOperator(name)},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: e2eutil.PodLabelForOperator(name),
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyNever,
					Containers: []v1.Container{{
						Name:            name,
						Image:           f.oldVOPImage,
						ImagePullPolicy: v1.PullAlways,
						Env: []v1.EnvVar{
							{
								Name:      "MY_POD_NAMESPACE",
								ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
							},
							{
								Name:      "MY_POD_NAME",
								ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.name"}},
							},
						},
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path: probe.HTTPReadyzEndpoint,
									Port: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
								},
							},
							InitialDelaySeconds: 3,
							PeriodSeconds:       3,
							FailureThreshold:    3,
						},
					}},
					ImagePullSecrets: []v1.LocalObjectReference{{
						Name: "coreos-pull-secret",
					}},
				},
			},
		},
	}
	_, err := f.KubeClient.AppsV1beta1().Deployments(f.Namespace).Create(d)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %v", err)
	}
	return nil
}

func (f *Framework) deployEtcdOperatorPod() error {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   etcdOperatorName,
			Labels: e2eutil.PodLabelForOperator(etcdOperatorName),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            etcdOperatorName,
					Image:           f.eopImage,
					ImagePullPolicy: v1.PullAlways,
					Env: []v1.EnvVar{
						{
							Name:      "MY_POD_NAMESPACE",
							ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
						},
						{
							Name:      "MY_POD_NAME",
							ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.name"}},
						},
					},
					ReadinessProbe: &v1.Probe{
						Handler: v1.Handler{
							HTTPGet: &v1.HTTPGetAction{
								Path: probe.HTTPReadyzEndpoint,
								Port: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
							},
						},
						InitialDelaySeconds: 3,
						PeriodSeconds:       3,
						FailureThreshold:    3,
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	_, err := f.KubeClient.CoreV1().Pods(f.Namespace).Create(pod)
	if err != nil {
		return fmt.Errorf("failed to create pod: %v", err)
	}
	return e2eutil.WaitUntilOperatorReady(f.KubeClient, f.Namespace, etcdOperatorName)
}
