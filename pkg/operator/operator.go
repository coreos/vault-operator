package operator

import (
	"context"
	"os"

	"github.com/coreos-inc/vault-operator/pkg/client"
	"github.com/coreos-inc/vault-operator/pkg/generated/clientset/versioned"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	etcdCRClientPkg "github.com/coreos/etcd-operator/pkg/client"
	etcdCRClient "github.com/coreos/etcd-operator/pkg/generated/clientset/versioned"
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

	kubecli     kubernetes.Interface
	vaultsCRCli versioned.Interface
	etcdCRCli   etcdCRClient.Interface
}

// New creates a vault operator.
func New() *Vaults {
	return &Vaults{
		namespace:   os.Getenv("MY_POD_NAMESPACE"),
		ctxCancels:  map[string]context.CancelFunc{},
		kubecli:     k8sutil.MustNewKubeClient(),
		vaultsCRCli: client.MustNewInCluster(),
		etcdCRCli:   etcdCRClientPkg.MustNewInCluster(),
	}
}

// Start starts the vault operator.
func (v *Vaults) Start(ctx context.Context) error {
	v.run(ctx)
	return ctx.Err()
}
