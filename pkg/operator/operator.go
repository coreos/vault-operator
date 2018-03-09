// Copyright 2018 The vault-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
