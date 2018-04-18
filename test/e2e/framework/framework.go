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
	"time"

	"github.com/coreos-inc/vault-operator/pkg/client"
	"github.com/coreos-inc/vault-operator/pkg/generated/clientset/versioned"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"

	etcdclient "github.com/coreos/etcd-operator/pkg/client"
	etcdversioned "github.com/coreos/etcd-operator/pkg/generated/clientset/versioned"
	eopK8sutil "github.com/coreos/etcd-operator/pkg/util/k8sutil"
	"github.com/coreos/etcd-operator/pkg/util/probe"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// The global framework variable used by all e2e tests
	Global            *Framework
	vaultOperatorName = "vault-operator"
	etcdOperatorName  = "etcd-operator"
)

// Framework struct contains the various clients and other information needed to run the e2e tests
type Framework struct {
	KubeClient     kubernetes.Interface
	crdcli         apiextensionsclient.Interface
	Config         *restclient.Config
	RESTClient     *restclient.RESTClient
	VaultsCRClient versioned.Interface
	EtcdCRClient   etcdversioned.Interface
	Namespace      string
	vopImage       string
	eopImage       string
}

// Setup initializes the Global framework by initializing necessary clients and creating the vault operator
func Setup() error {
	kubeconfig := flag.String("kubeconfig", "", "kube config path, e.g. $HOME/.kube/config")
	vopImage := flag.String("operator-image", "", "operator image, e.g. quay.io/coreos/vault-operator-dev:latest")
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
	crdcli, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kube CRD client: %v", err)
	}
	vaultsCRClient := client.MustNew(config)
	etcdCRClient := etcdclient.MustNew(config)

	Global = &Framework{
		KubeClient:     kubeClient,
		crdcli:         crdcli,
		Config:         config,
		VaultsCRClient: vaultsCRClient,
		EtcdCRClient:   etcdCRClient,
		Namespace:      *ns,
		vopImage:       *vopImage,
		eopImage:       *eopImage,
	}

	return Global.setup()
}

// Teardown removes the vault-operator deployment and waits for its termination
func Teardown() error {
	err := Global.KubeClient.CoreV1().Pods(Global.Namespace).Delete(vaultOperatorName, k8sutil.CascadeDeleteBackground())
	if err != nil {
		return fmt.Errorf("failed to delete pod: %v", err)
	}
	err = Global.KubeClient.CoreV1().Pods(Global.Namespace).Delete(etcdOperatorName, k8sutil.CascadeDeleteBackground())
	if err != nil {
		return fmt.Errorf("failed to delete pod: %v", err)
	}
	Global = nil
	logrus.Info("e2e teardown successfully")
	return nil
}

func (f *Framework) setup() error {
	if err := f.deployEtcdOperatorPod(); err != nil {
		return fmt.Errorf("failed to setup etcd operator: %v", err)
	}

	if err := f.deployVaultOperatorPod(); err != nil {
		return fmt.Errorf("failed to setup vault operator: %v", err)
	}
	logrus.Info("e2e setup successfully")
	return nil
}

func (f *Framework) deployVaultOperatorPod() error {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vaultOperatorName,
			Namespace: f.Namespace,
			Labels:    e2eutil.PodLabelForOperator(vaultOperatorName),
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{{
				Name:            vaultOperatorName,
				Image:           f.vopImage,
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
			}},
		},
	}

	// Create and wait for pod phase to become running
	// TODO: Replace with operator pod actually becoming ready once the vault-operator supports a readiness probe
	_, err := eopK8sutil.CreateAndWaitPod(f.KubeClient, f.Namespace, pod, 60*time.Second)
	if err != nil {
		return err
	}
	if err != nil {
		return fmt.Errorf("failed to create pod: %v", err)
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
				{
					Name:            "etcd-backup-operator",
					Image:           f.eopImage,
					ImagePullPolicy: v1.PullAlways,
					Command:         []string{"etcd-backup-operator", "--create-crd=false"},
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
				},
				{
					Name:            "etcd-restore-operator",
					Image:           f.eopImage,
					ImagePullPolicy: v1.PullAlways,
					Command:         []string{"etcd-restore-operator", "--create-crd=false"},
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
