package operator

import (
	"context"
	"os"

	"github.com/coreos-inc/vault-operator/pkg/client"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	etcdCRClient "github.com/coreos/etcd-operator/pkg/client"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Vaults struct {
	namespace string
	// ctxCancels stores vault clusters' contexts that are used to
	// cancel their goroutines when they are deleted
	ctxCancels map[string]context.CancelFunc

	// k8s workqueue pattern
	indexer  cache.Indexer
	informer cache.Controller
	queue    workqueue.RateLimitingInterface

	kubecli       kubernetes.Interface
	vaultsCRCli   client.Vaults
	kubeExtClient apiextensionsclient.Interface
	etcdCRCli     etcdCRClient.EtcdClusterCR
}

// New creates a vault operator.
func New() *Vaults {
	return &Vaults{
		namespace:     os.Getenv("MY_POD_NAMESPACE"),
		ctxCancels:    map[string]context.CancelFunc{},
		kubecli:       k8sutil.MustNewKubeClient(),
		vaultsCRCli:   client.MustNewInCluster(),
		kubeExtClient: k8sutil.MustNewKubeExtClient(),
		etcdCRCli:     etcdCRClient.MustNewCRInCluster(),
	}
}

// Start starts the vault operator.
func (v *Vaults) Start(ctx context.Context) error {
	err := v.init(ctx)
	if err != nil {
		return err
	}
	go v.run(ctx)
	<-ctx.Done()
	return ctx.Err()
}

func (v *Vaults) init(ctx context.Context) error {
	err := k8sutil.CreateVaultCRD(v.kubeExtClient)
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
