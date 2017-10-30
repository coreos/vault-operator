package client

import (
	"github.com/coreos-inc/vault-operator/pkg/generated/clientset/versioned"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"k8s.io/client-go/rest"
)

func MustNewInCluster() versioned.Interface {
	cfg, err := k8sutil.InClusterConfig()
	if err != nil {
		panic(err)
	}
	return MustNew(cfg)
}

// MustNew create a new vault client based on the Kubernetes client configuration passed in
func MustNew(cfg *rest.Config) versioned.Interface {
	return versioned.NewForConfigOrDie(cfg)
}
