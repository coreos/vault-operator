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

package framework

import (
	"flag"
	"fmt"

	"github.com/coreos-inc/vault-operator/pkg/client"
	"github.com/coreos-inc/vault-operator/pkg/generated/clientset/versioned"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/pkg/util/probe"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"

	eopk8sutil "github.com/coreos/etcd-operator/pkg/util/k8sutil"
	"github.com/sirupsen/logrus"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	VaultsCRClient versioned.Interface
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
	eopImage := flag.String("etcd-operator-image", "quay.io/coreos/etcd-operator:v0.8.3", "etcd operator image, e.g. quay.io/coreos/etcd-operator")
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
	vaultsCRClient := client.MustNew(config)

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

// DeleteOperatorDeployment deletes the vault operator deployment with the specified name, waits for all its pods to be removed and deletes the Endpoint resource used for the leader election lock
func (f *Framework) DeleteOperatorDeployment(name string) error {
	err := f.KubeClient.AppsV1beta1().Deployments(f.Namespace).Delete(name, k8sutil.CascadeDeleteBackground())
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %v", err)
	}

	lo := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(e2eutil.PodLabelForOperator(name)).String(),
	}
	_, err = e2eutil.WaitPodsDeletedCompletely(f.KubeClient, f.Namespace, 3, lo)
	if err != nil {
		return fmt.Errorf("failed to wait for operator pods to be completely removed: %v", err)
	}
	// The deleted operator will not actively release the Endpoints lock causing a non-leader candidate to timeout for the lease duration: 15s
	// Deleting the Endpoints resource simulates the leader actively releasing the lock so that the next candidate avoids the timeout.
	// TODO: change this if we change to use another kind of lock, e.g. configmap.
	return f.KubeClient.CoreV1().Endpoints(f.Namespace).Delete("vault-operator", metav1.NewDeleteOptions(0))
}

func (f *Framework) UpgradeOperator(name string) error {
	uf := func(d *appsv1beta1.Deployment) {
		d.Spec.Template.Spec.Containers[0].Image = f.newVOPImage
	}
	// TODO: Put PatchDeployment into vault's k8sutil
	err := eopk8sutil.PatchDeployment(f.KubeClient, f.Namespace, name, uf)
	if err != nil {
		return fmt.Errorf("failed to patch deployment: %v", err)
	}

	lo := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(e2eutil.PodLabelForOperator(name)).String(),
	}
	pods, err := e2eutil.WaitPodsWithImageDeleted(f.KubeClient, f.Namespace, f.oldVOPImage, 3, lo)
	if err != nil {
		return fmt.Errorf("failed to wait for pods (%v) with old image (%v) to get deleted: %v", pods, f.oldVOPImage, err)
	}
	return e2eutil.WaitUntilOperatorReady(f.KubeClient, f.Namespace, name)
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
					Command:         []string{"etcd-operator", "--create-crd=false"},
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
