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
	"fmt"
	"time"

	api "github.com/coreos/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos/vault-operator/pkg/util/probe"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (v *Vaults) run(ctx context.Context) {
	source := cache.NewListWatchFromClient(
		v.vaultsCRCli.VaultV1alpha1().RESTClient(),
		api.VaultServicePlural,
		v.namespace,
		fields.Everything())

	v.queue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "vault-operator")
	v.indexer, v.informer = cache.NewIndexerInformer(source, &api.VaultService{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc:    v.onAddVault,
		UpdateFunc: v.onUpdateVault,
		DeleteFunc: v.onDeleteVault,
	}, cache.Indexers{})

	defer v.queue.ShutDown()

	logrus.Info("starting Vaults controller")
	go v.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), v.informer.HasSynced) {
		logrus.Error("Timed out waiting for caches to sync")
		return
	}

	probe.SetReady()

	const numWorkers = 1
	for i := 0; i < numWorkers; i++ {
		go wait.Until(v.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
	logrus.Info("stopping Vaults controller")
}

func (v *Vaults) onAddVault(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	annotations := api.GetAnnotations()
	if err != nil {
		panic(err)
	}
	v.queue.Add(key)
	logrus.Infof("Vault CR (%s) is created", annotations)
	//annotations := api.GetAnnotations()
	//logrus.Infof("Annotations: %s", annotations)
}

func (v *Vaults) onUpdateVault(oldObj, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		panic(err)
	}
	v.queue.Add(key)
}

func (v *Vaults) onDeleteVault(obj interface{}) {
	vr, ok := obj.(*api.VaultService)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			panic(fmt.Sprintf("unknown object from Vault delete event: %#v", obj))
		}
		vr, ok = tombstone.Obj.(*api.VaultService)
		if !ok {
			panic(fmt.Sprintf("Tombstone contained object that is not a Vault: %#v", obj))
		}
	}

	if cancel, ok := v.ctxCancels[vr.Name]; ok {
		cancel()
		delete(v.ctxCancels, vr.Name)
	}

	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function.
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	v.queue.Add(key)
}
