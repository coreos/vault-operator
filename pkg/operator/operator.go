package operator

import (
	"context"
	"os"

	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
)

type Vaults struct {
	namespace string

	restClient    *rest.RESTClient
	kubeExtClient apiextensionsclient.Interface
}

func New() *Vaults {
	return &Vaults{
		namespace: os.Getenv("MY_POD_NAMESPACE"),
	}
}

func (v *Vaults) Start(ctx context.Context) {
	err := v.init(ctx)
	if err != nil {
		panic(err)
	}
	v.run(ctx)
	<-ctx.Done()
}

func (v *Vaults) init(ctx context.Context) error {
	return k8sutil.CreateVaultCRD(v.kubeExtClient)
}
