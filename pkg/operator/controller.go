package operator

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/pkg/util/vaultutil"

	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (v *Vaults) run(ctx context.Context) {
	source := cache.NewListWatchFromClient(
		v.vaultsCRCli.RESTClient(),
		spec.VaultResourcePlural,
		v.namespace,
		fields.Everything())

	v.queue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "vault-operator")
	v.indexer, v.informer = cache.NewIndexerInformer(source, &spec.Vault{}, 0, cache.ResourceEventHandlerFuncs{
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

	const numWorkers = 1
	for i := 0; i < numWorkers; i++ {
		go wait.Until(v.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
	logrus.Info("stopping Vaults controller")
}

func (v *Vaults) onAddVault(obj interface{}) {
	vr := obj.(*spec.Vault)

	if !spec.IsTLSConfigured(vr.Spec.TLS) {
		err := v.prepareDefaultVaultTLSSecrets(vr)
		if err != nil {
			// TODO: retry or report failure status in CR
			panic(err)
		}
	}

	// Simulate initializer.
	// TODO: remove this when we have initializers for Vault CR.
	vr.SetDefaults()
	vr, err := v.vaultsCRCli.Update(context.TODO(), vr)
	if err != nil {
		panic(err)
	}

	err = v.prepareEtcdTLSSecrets(vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}

	err = v.prepareVaultConfig(vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	v.queue.Add(key)
}

// prepareVaultConfig applies our section into Vault config file.
// - If given user configmap, appends into user provided vault config
//   and creates another configmap "${configMapName}-copy" for it.
// - Otherwise, creates a new configmap "${vaultName}-copy" with our section.
func (v *Vaults) prepareVaultConfig(vr *spec.Vault) error {
	var cfgData string
	if len(vr.Spec.ConfigMapName) != 0 {
		cm, err := v.kubecli.CoreV1().ConfigMaps(vr.Namespace).Get(vr.Spec.ConfigMapName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("prepare vault config error: get configmap (%s) failed: %v", vr.Spec.ConfigMapName, err)
		}
		cfgData = cm.Data[filepath.Base(k8sutil.VaultConfigPath)]
	}
	cfgData = vaultutil.NewConfigWithListener(cfgData)
	cfgData = vaultutil.NewConfigWithEtcd(cfgData, k8sutil.EtcdURLForVault(vr.Name))

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sutil.ConfigMapNameForVault(vr),
		},
		Data: map[string]string{
			filepath.Base(k8sutil.VaultConfigPath): cfgData,
		},
	}

	_, err := v.kubecli.CoreV1().ConfigMaps(vr.Namespace).Create(cm)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("prepare vault config error: create new configmap (%s) failed: %v", cm.Name, err)
	}

	return nil
}

func (v *Vaults) onUpdateVault(oldObj, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		panic(err)
	}
	v.queue.Add(key)
}

func (v *Vaults) onDeleteVault(obj interface{}) {
	// TODO: Make use of k8s GC.

	vr, ok := obj.(*spec.Vault)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			panic(fmt.Sprintf("unknown object from Vault delete event: %#v", obj))
		}
		vr, ok = tombstone.Obj.(*spec.Vault)
		if !ok {
			panic(fmt.Sprintf("Tombstone contained object that is not a Vault: %#v", obj))
		}
	}

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
		k8sutil.ConfigMapNameForVault(vr), nil)
	if err != nil && !apierrors.IsNotFound(err) {
		// TODO: retry or report failure status in CR
		panic(err)
	}
	err = v.cleanupEtcdTLSSecrets(vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}

	err = v.cleanupDefaultVaultTLSSecrets(vr)
	if err != nil {
		// TODO: retry or report failure status in CR
		panic(err)
	}

	cancel := v.ctxCancels[vr.Name]
	cancel()
	delete(v.ctxCancels, vr.Name)

	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function.
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	v.queue.Add(key)
}
