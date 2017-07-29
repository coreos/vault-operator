package operator

import (
	"context"
	"time"

	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"github.com/Sirupsen/logrus"
	vaultapi "github.com/hashicorp/vault/api"
)

// monitorAndUpdateStaus monitors the vault service and replicas statuses, and
// updates the status resrouce in the vault CR item.
func monitorAndUpdateStaus(ctx context.Context, v *spec.Vault) {
	// create a long-live client for accssing vault service.
	cfg := vaultapi.DefaultConfig()
	cfg.Address = k8sutil.VaultServiceAddr(v.Name, v.Namespace)
	vsc, err := vaultapi.NewClient(cfg)
	if err != nil {
		logrus.Errorf("failed creating client for vault service: %s/%s", v.Name, v.Namespace)
	}

	s := v.Status

	for {
		select {
		case err := <-ctx.Done():
			logrus.Infof("stopped monitoring vault: %s (%v)", v.Name, err)
		case <-time.After(10 * time.Second):
		}
		updateVaultStatus(ctx, vsc, v, &s)

		// TODO: update status in the CR item.
	}
}

// updateVaultStatus updates the vault service status through the service DNS address.
func updateVaultStatus(ctx context.Context, vc *vaultapi.Client, v *spec.Vault, s *spec.VaultStatus) {
	inited, err := vc.Sys().InitStatus()
	if err != nil {
		logrus.Errorf("failed getting the init status for vault service: %s (%v)", v.Name, err)
		return
	}
	s.Initialized = inited
}

// updateVaultReplicasStatus updates the status of every vault replicas in the vault deployment.
func updateVaultReplicasStatus(ctx context.Context, v *spec.Vault, s *spec.VaultStatus) {
	// nothing here.
}
