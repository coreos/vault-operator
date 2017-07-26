package k8sutil

import (
	"github.com/coreos-inc/vault-operator/pkg/spec"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	appsv1beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

var (
	vaultImage = "vault"
)

// DeployVault deploys a vault service.
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
					"-dev",
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

	return err
}
