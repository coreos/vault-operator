package operator

import (
	"context"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
)

const (
	// Copy from deployment_controller.go:
	// maxRetries is the number of times a Vault will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a Vault is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

func (v *Vaults) runWorker() {
	for v.processNextItem() {
	}
}

func (v *Vaults) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := v.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer v.queue.Done(key)

	// Invoke the method containing the business logic
	err := v.reconcileVault(key.(string))
	// Handle the error if something went wrong during the execution of the business logic
	v.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (v *Vaults) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		v.queue.Forget(key)
		return
	}

	// This controller retries maxRetries times if something goes wrong. After that, it stops trying.
	if v.queue.NumRequeues(key) < maxRetries {
		logrus.Errorf("error syncing Vault (%v): %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		v.queue.AddRateLimited(key)
		return
	}

	v.queue.Forget(key)
	// Report that, even after several retries, we could not successfully process this key
	logrus.Infof("Dropping Vault (%v) out of the queue: %v", key, err)
}

func (v *Vaults) reconcileVault(key string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("reconcile Vault failed: %v", err)
		}
	}()

	obj, exists, err := v.indexer.GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		logrus.Infof("deleting Vault: %s", key)
		return nil
	}
	vr := obj.(*spec.Vault)

	err = k8sutil.DeployEtcdCluster(v.etcdCRCli, vr)
	if err != nil {
		return err
	}

	// TODO: we should do
	// if ! deployment exists -> then create deployment
	// else -> check size, version skew
	// If ! service exists -> then create service
	err = k8sutil.DeployVault(v.kubecli, vr)
	if err != nil {
		return err
	}

	err = k8sutil.UpdateVault(v.kubecli, vr)
	if err != nil {
		return err
	}

	if _, ok := v.ctxCancels[vr.Name]; !ok {
		ctx, cancel := context.WithCancel(context.Background())
		v.ctxCancels[vr.Name] = cancel
		go v.monitorAndUpdateStaus(ctx, vr)
	}

	return nil
}
