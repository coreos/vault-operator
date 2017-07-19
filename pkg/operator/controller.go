package operator

import (
	"context"
	"log"

	"github.com/coreos-inc/vault-operator/pkg/spec"

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
	// nothing
}

func (v *Vaults) onUpdate(oldObj, newObj interface{}) {
	// nothing
}

func (v *Vaults) onDelete(obj interface{}) {
	// nothing
}
