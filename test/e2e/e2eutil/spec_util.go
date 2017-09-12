package e2eutil

import (
	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCluster returns a minimal vault cluster CR
func NewCluster(genName, namespace string, size int) *api.VaultService {
	return &api.VaultService{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.VaultServiceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: genName,
			Namespace:    namespace,
		},
		Spec: api.VaultServiceSpec{
			Nodes: int32(size),
		},
	}
}
