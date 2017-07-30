package operator

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/pkg/util/vaultutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func (v *Vaults) run(ctx context.Context) {
	source := cache.NewListWatchFromClient(
		v.vaultsCRCli.RESTClient(),
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
	err := v.prepareVaultConfig(vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}

	err = k8sutil.DeployEtcdCluster(v.etcdCRCli, vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}

	err = k8sutil.DeployVault(v.kubecli, vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}
	go v.monitorAndUpdateStaus(context.TODO(), vr.GetName(), vr.GetNamespace())
}

// prepareVaultConfig appends etcd storage section into user provided vault config
// and creates another (predefined-name) configmap for it.
func (v *Vaults) prepareVaultConfig(vr *spec.Vault) error {
	cm, err := v.kubecli.CoreV1().ConfigMaps(vr.Namespace).Get(vr.Spec.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("prepare vault config error: get configmap (%s) failed: %v", vr.Spec.ConfigMapName, err)
	}

	cfgData := cm.Data[filepath.Base(k8sutil.VaultConfigPath)]
	cm.Data[filepath.Base(k8sutil.VaultConfigPath)] =
		vaultutil.NewConfigWithEtcd(cfgData, k8sutil.EtcdURLForVault(vr.Name))
	cm.ObjectMeta = metav1.ObjectMeta{
		Name: k8sutil.ConfigMapCopyName(vr.Spec.ConfigMapName),
	}

	_, err = v.kubecli.CoreV1().ConfigMaps(vr.Namespace).Create(cm)
	if err != nil {
		return fmt.Errorf("prepare vault config error: create new configmap (%s) failed: %v", cm.Name, err)
	}

	return nil
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
	err = v.kubecli.CoreV1().ConfigMaps(vr.Namespace).Delete(
		k8sutil.ConfigMapCopyName(vr.Spec.ConfigMapName), nil)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}
}
