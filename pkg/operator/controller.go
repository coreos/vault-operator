package operator

import (
	"context"
	"log"

	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func (v *Vaults) run(ctx context.Context) {
	source := cache.NewListWatchFromClient(
		v.restClient,
		spec.VaultResourcePlural,
		v.namespace,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,
		// The object type.
		&spec.Vault{},
		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,
		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    v.onAdd,
			UpdateFunc: v.onUpdate,
			DeleteFunc: v.onDelete,
		})

	go controller.Run(ctx.Done())
	log.Println("start processing Vaults changes...")
}

func (v *Vaults) onAdd(obj interface{}) {
	vr := obj.(*spec.Vault)
	err := k8sutil.DeployEtcdCluster(v.etcdCRCli, vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}

	err = k8sutil.DeployVault(v.kubecli, vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}
	go monitorAndUpdateStaus(context.TODO(), vr)
}

func (v *Vaults) onUpdate(oldObj, newObj interface{}) {
	// nothing
}

func (v *Vaults) onDelete(obj interface{}) {
	vr := obj.(*spec.Vault)
	err := k8sutil.DestroyVault(v.kubecli, vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}
	err = k8sutil.DeleteEtcdCluster(v.etcdCRCli, vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}
}
