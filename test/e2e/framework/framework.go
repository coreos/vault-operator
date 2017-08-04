package framework

import (
	"flag"
	"fmt"
	"time"

	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"github.com/Sirupsen/logrus"
	eopK8sutil "github.com/coreos/etcd-operator/pkg/util/k8sutil"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// The global framework variable used by all e2e tests
	Global       *Framework
	operatorName = "vault-operator"
)

// Framework struct contains the various clients and other information needed to run the e2e tests
type Framework struct {
	KubeClient kubernetes.Interface
	Namespace  string
	vopImage   string
}

// Setup initializes the Global framework by initializing necessary clients and creating the vault operator
func Setup() error {
	kubeconfig := flag.String("kubeconfig", "", "kube config path, e.g. $HOME/.kube/config")
	vopImage := flag.String("operator-image", "", "operator image, e.g. quay.io/coreos/vault-operator-dev:latest")
	ns := flag.String("namespace", "", "e2e test namespace")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return err
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// TODO: need a CR client to CRUD the vault CR later

	Global = &Framework{
		KubeClient: cli,
		Namespace:  *ns,
		vopImage:   *vopImage,
	}

	return Global.setup()
}

// Teardown removes the vault-operator deployment and waits for its termination
func Teardown() error {
	err := Global.KubeClient.CoreV1().Pods(Global.Namespace).Delete(operatorName, k8sutil.CascadeDeleteBackground())
	if err != nil {
		return fmt.Errorf("failed to delete pod: %v", err)
	}
	Global = nil
	logrus.Info("e2e teardown successfully")
	return nil
}

func (f *Framework) setup() error {
	if err := f.createVaultOperatorPod(); err != nil {
		return fmt.Errorf("failed to setup vault operator: %v", err)
	}
	logrus.Info("e2e setup successfully")
	return nil
}

func (f *Framework) createVaultOperatorPod() error {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorName,
			Namespace: f.Namespace,
			Labels:    podLabelForOperator(operatorName),
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{{
				Name:            operatorName,
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
			ImagePullSecrets: []v1.LocalObjectReference{{
				Name: "coreos-pull-secret",
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

func podLabelForOperator(name string) map[string]string {
	return map[string]string{"name": name}
}
