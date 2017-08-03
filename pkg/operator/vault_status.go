package operator

import (
	"context"
	"time"

	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"github.com/Sirupsen/logrus"
	vaultapi "github.com/hashicorp/vault/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// monitorAndUpdateStaus monitors the vault service and replicas statuses, and
// updates the status resrouce in the vault CR item.
func (vs *Vaults) monitorAndUpdateStaus(ctx context.Context, name, namespace string) {
	s := spec.VaultStatus{}

	for {
		select {
		case err := <-ctx.Done():
			logrus.Infof("stopped monitoring vault: %s (%v)", name, err)
		case <-time.After(10 * time.Second):
		}

		vs.updateLocalVaultCRStatus(ctx, name, namespace, &s)

		err := vs.updateVaultCRStatus(ctx, name, namespace, s)
		if err != nil {
			logrus.Errorf("failed updating the status for the vault service: %s (%v)", name, err)
		}
	}
}

// updateLocalVaultCRStatus updates local vault CR status by querying each vault pod's API.
func (vs *Vaults) updateLocalVaultCRStatus(ctx context.Context, name, namespace string, s *spec.VaultStatus) {
	sel := k8sutil.PodsLabelsForVault(name)
	// TODO: handle upgrades when pods from two replicaset can co-exist :(
	opt := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(sel).String()}
	pods, err := vs.kubecli.CoreV1().Pods(namespace).List(opt)
	if err != nil {
		logrus.Errorf("failed to update vault replica status: failed listing pods for the vault service (%s.%s): %v", name, namespace, err)
		return
	}

	var sealNodes []string
	// If it can't talk to any vault pod, we are not changing the state.
	inited := s.Initialized

	for _, p := range pods.Items {
		cfg := vaultapi.DefaultConfig()
		// TODO: change to https.
		// TODO: use FQDN?
		podURL := "http://" + p.Status.PodIP + ":8200"
		cfg.Address = podURL
		vapi, err := vaultapi.NewClient(cfg)
		if err != nil {
			logrus.Errorf("failed to update vault replica status: failed creating client for the vault pod (%s/%s): %v", namespace, p.GetName(), err)
			continue
		}

		hr, err := vapi.Sys().Health()
		if err != nil {
			logrus.Errorf("failed to update vault replica status: failed requesting health info for the vault pod (%s/%s): %v", namespace, p.GetName(), err)
			continue
		}
		// is active node?
		// TODO: add to vaultutil?
		if hr.Initialized && !hr.Sealed && !hr.Standby {
			s.ActiveNode = podURL
		}
		if hr.Sealed {
			sealNodes = append(sealNodes, podURL)
		}
		if hr.Initialized {
			inited = true
		}
	}

	s.SealedNodes = sealNodes
	s.Initialized = inited
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
