package e2eutil

import (
	"github.com/coreos-inc/vault-operator/pkg/spec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCluster returns a minimal vault cluster CR
func NewCluster(genName, namespace string, size int) *spec.Vault {
	return &spec.Vault{
		TypeMeta: metav1.TypeMeta{
			Kind:       spec.VaultResourceKind,
			APIVersion: spec.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: genName,
			Namespace:    namespace,
		},
		Spec: spec.VaultSpec{
			Nodes: int32(size),
		},
	}
}
