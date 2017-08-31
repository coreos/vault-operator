package operator

import (
	"context"
	"reflect"
	"time"

	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/pkg/util/vaultutil"

	"github.com/Sirupsen/logrus"
	vaultapi "github.com/hashicorp/vault/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// monitorAndUpdateStaus monitors the vault service and replicas statuses, and
// updates the status resrouce in the vault CR item.
func (vs *Vaults) monitorAndUpdateStaus(ctx context.Context, vr *spec.Vault) {
	var tlsConfig *vaultapi.TLSConfig
	for {
		select {
		case err := <-ctx.Done():
			logrus.Infof("stop monitoring vault (%s), reason: %v", vr.GetName(), err)
			return
		case <-time.After(10 * time.Second):
		}

		if tlsConfig == nil {
			var err error
			tlsConfig, err = k8sutil.VaultTLSFromSecret(vs.kubecli, vr)
			if err != nil {
				logrus.Errorf("failed to read TLS config for vault client: %v", err)
				continue
			}
		}

		s := spec.VaultStatus{}
		vs.updateLocalVaultCRStatus(ctx, vr.GetName(), vr.GetNamespace(), &s, tlsConfig)

		err := vs.updateVaultCRStatus(ctx, vr.GetName(), vr.GetNamespace(), s)
		if err != nil {
			logrus.Errorf("failed updating the status for the vault service: %s (%v)", vr.GetName(), err)
		}
	}
}

// updateLocalVaultCRStatus updates local vault CR status by querying each vault pod's API.
func (vs *Vaults) updateLocalVaultCRStatus(ctx context.Context, name, namespace string, s *spec.VaultStatus, tlsConfig *vaultapi.TLSConfig) {
	sel := k8sutil.LabelsForVault(name)
	// TODO: handle upgrades when pods from two replicaset can co-exist :(
	opt := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(sel).String()}
	pods, err := vs.kubecli.CoreV1().Pods(namespace).List(opt)
	if err != nil {
		logrus.Errorf("failed to update vault replica status: failed listing pods for the vault service (%s.%s): %v", name, namespace, err)
		return
	}

	var sealNodes []string
	var availableNodes []string
	var standByNodes []string
	// If it can't talk to any vault pod, we are not changing the state.
	inited := s.Initialized

	for _, p := range pods.Items {
		if p.Status.Phase != v1.PodRunning {
			continue
		}
		availableNodes = append(availableNodes, p.GetName())

		vapi, err := vaultutil.NewClient(k8sutil.PodDNSName(p), tlsConfig)
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
			s.ActiveNode = p.GetName()
		}
		if hr.Initialized && !hr.Sealed && hr.Standby {
			standByNodes = append(standByNodes, p.GetName())
		}
		if hr.Sealed {
			sealNodes = append(sealNodes, p.GetName())
		}
		if hr.Initialized {
			inited = true
		}
	}

	s.AvailableNodes = availableNodes
	s.StandbyNodes = standByNodes
	s.SealedNodes = sealNodes
	s.Initialized = inited
}

// updateVaultCRStatus updates the status field of the Vault CR.
func (vs *Vaults) updateVaultCRStatus(ctx context.Context, name, namespace string, status spec.VaultStatus) error {
	vault, err := vs.vaultsCRCli.Get(ctx, namespace, name)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(vault.Status, status) {
		return nil
	}
	vault.Status = status
	_, err = vs.vaultsCRCli.Update(ctx, vault)
	return err
}
