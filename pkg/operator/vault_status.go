package operator

import (
	"context"
	"time"

	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"github.com/Sirupsen/logrus"
	vaultapi "github.com/hashicorp/vault/api"
)

// monitorVaultStatus monitors the vault service status through the service DNS address.
func monitorVaultStatus(ctx context.Context, v *spec.Vault) {
	cfg := vaultapi.DefaultConfig()
	cfg.Address = k8sutil.VaultServiceAddr(v.Name, v.Namespace)

	vc, err := vaultapi.NewClient(cfg)
	if err != nil {
		logrus.Errorf("failed creating client for vault service: %s/%s", v.Name, v.Namespace)
	}

	for {
		select {
		case err := <-ctx.Done():
			logrus.Infof("stopped monitoring vault service: %s (%v)", v.Name, err)
		case <-time.After(10 * time.Second):
		}
		// TODO: change to update status
		inited, err := vc.Sys().InitStatus()
		if err != nil {
			logrus.Errorf("failed getting the init status for vault service: %s (%v)", v.Name, err)
		} else {
			logrus.Infof("vault init: %v", inited)
		}
	}
}

// monitorVaultReplicasStatus monitors the status of every vault replicas in the vault deployment.
func monitorVaultReplicasStatus(ctx context.Context, v *spec.Vault) {
	// nothing here.
}
