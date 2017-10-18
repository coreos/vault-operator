package operator

import (
	"context"
	"os"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos-inc/vault-operator/pkg/client"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	etcdCRClient "github.com/coreos/etcd-operator/pkg/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Vaults struct {
	namespace string
	// ctxCancels stores vault clusters' contexts that are used to
	// cancel their goroutines when they are deleted
	ctxCancels map[string]context.CancelFunc

	// vault objects that need to be Garbage Collected during sync
	// saved here since deleted objects are removed from the cache
	toDelete map[string]*api.VaultService

	// k8s workqueue pattern
	indexer  cache.Indexer
	informer cache.Controller
	queue    workqueue.RateLimitingInterface

	kubecli     kubernetes.Interface
	vaultsCRCli client.Vaults
	etcdCRCli   etcdCRClient.EtcdClusterCR
}

// New creates a vault operator.
func New() *Vaults {
	return &Vaults{
		namespace:   os.Getenv("MY_POD_NAMESPACE"),
		ctxCancels:  map[string]context.CancelFunc{},
		toDelete:    map[string]*api.VaultService{},
		kubecli:     k8sutil.MustNewKubeClient(),
		vaultsCRCli: client.MustNewInCluster(),
		etcdCRCli:   etcdCRClient.MustNewCRInCluster(),
	}
}

// Start starts the vault operator.
func (v *Vaults) Start(ctx context.Context) error {
	v.run(ctx)
	return ctx.Err()
}
