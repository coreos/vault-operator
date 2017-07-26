package operator

import (
	"context"
	"os"

	"github.com/coreos-inc/vault-operator/pkg/client"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Vaults struct {
	namespace string

	kubecli       kubernetes.Interface
	restClient    *rest.RESTClient
	kubeExtClient apiextensionsclient.Interface
}

// New creates a vault operator.
func New() *Vaults {
	vc := client.MustNewInCluster()

	return &Vaults{
		namespace:     os.Getenv("MY_POD_NAMESPACE"),
		kubecli:       k8sutil.MustNewKubeClient(),
		restClient:    vc.RESTClient(),
		kubeExtClient: k8sutil.MustNewKubeExtClient(),
	}
}

// Start starts the vault operator.
func (v *Vaults) Start(ctx context.Context) error {
	err := v.init(ctx)
	if err != nil {
		return err
	}
	v.run(ctx)
	<-ctx.Done()
	return ctx.Err()
}

func (v *Vaults) init(ctx context.Context) error {
	return k8sutil.CreateVaultCRD(v.kubeExtClient)
}
