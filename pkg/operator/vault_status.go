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
func (vs *Vaults) monitorAndUpdateStaus(ctx context.Context, name, namespace string) {
	// create a long-live client for accssing vault service.
	cfg := vaultapi.DefaultConfig()
	cfg.Address = k8sutil.VaultServiceAddr(name, namespace)
	vapi, err := vaultapi.NewClient(cfg)
	if err != nil {
		logrus.Errorf("failed creating client for the vault service: %s/%s", name, namespace)
	}

	s := spec.VaultStatus{}

	for {
		select {
		case err := <-ctx.Done():
			logrus.Infof("stopped monitoring vault: %s (%v)", name, err)
		case <-time.After(10 * time.Second):
		}
		err := updateVaultStatus(ctx, vapi, &s)
		if err != nil {
			logrus.Errorf("failed getting the init status for the vault service: %s (%v)", name, err)
			continue
		}

		err = vs.updateVaultCRStatus(ctx, name, namespace, s)
		if err != nil {
			logrus.Errorf("failed updating the status for the vault service: %s (%v)", name, err)
		}
	}
}

// updateVaultStatus updates the vault service status through the service DNS address.
func updateVaultStatus(ctx context.Context, vc *vaultapi.Client, s *spec.VaultStatus) error {
	inited, err := vc.Sys().InitStatus()
	if err != nil {
		return err
	}
	s.Initialized = inited
	return nil
}

// updateVaultReplicasStatus updates the status of every vault replicas in the vault deployment.
func updateVaultReplicasStatus(ctx context.Context, name, namespace string, s *spec.VaultStatus) {
	// nothing here.
}

// updateVaultCRStatus updates the status field of the Vault CR.
func (vs *Vaults) updateVaultCRStatus(ctx context.Context, name, namespace string, status spec.VaultStatus) error {
	vault, err := vs.vaultsCRCli.Get(ctx, namespace, name)
	if err != nil {
		return err
	}
	vault.Status = status
	_, err = vs.vaultsCRCli.Update(ctx, vault)
	return err
}
