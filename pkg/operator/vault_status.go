package operator

import (
	"context"
	"reflect"
	"time"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
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
func (vs *Vaults) monitorAndUpdateStaus(ctx context.Context, vr *api.VaultService) {
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

		s := api.VaultServiceStatus{
			ServiceName: vr.GetName(),
			ClientPort:  k8sutil.VaultClientPort,
		}
		vs.updateLocalVaultCRStatus(ctx, vr, &s, tlsConfig)

		latest, err := vs.updateVaultCRStatus(ctx, vr.GetName(), vr.GetNamespace(), s)
		if err != nil {
			logrus.Errorf("failed updating the status for the vault service: %s (%v)", vr.GetName(), err)
		}
		if latest != nil {
			vr = latest
		}
	}
}

// updateLocalVaultCRStatus updates local vault CR status by querying each vault pod's API.
func (vs *Vaults) updateLocalVaultCRStatus(ctx context.Context, vr *api.VaultService, s *api.VaultServiceStatus, tlsConfig *vaultapi.TLSConfig) {
	name, namespace := vr.Name, vr.Namespace
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
	var updatedNodes []string
	inited := false
	// If it can't talk to any vault pod, we are not going to change the status.
	changed := false

	for _, p := range pods.Items {
		// If a pod is Terminating, it is still Running but has no IP.
		if p.Status.Phase != v1.PodRunning || p.DeletionTimestamp != nil {
			continue
		}

		vapi, err := vaultutil.NewClient(k8sutil.PodDNSName(p), "8200", tlsConfig)
		if err != nil {
			logrus.Errorf("failed to update vault replica status: failed creating client for the vault pod (%s/%s): %v", namespace, p.GetName(), err)
			continue
		}

		hr, err := vapi.Sys().Health()
		if err != nil {
			logrus.Errorf("failed to update vault replica status: failed requesting health info for the vault pod (%s/%s): %v", namespace, p.GetName(), err)
			continue
		}

		changed = true

		availableNodes = append(availableNodes, p.GetName())
		if k8sutil.IsVaultVersionMatch(p.Spec, vr.Spec) {
			updatedNodes = append(updatedNodes, p.GetName())
		}

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

	if !changed {
		return
	}

	s.AvailableNodes = availableNodes
	s.StandbyNodes = standByNodes
	s.SealedNodes = sealNodes
	s.Initialized = inited
	s.UpdatedNodes = updatedNodes
}

// updateVaultCRStatus updates the status field of the Vault CR.
func (vs *Vaults) updateVaultCRStatus(ctx context.Context, name, namespace string, status api.VaultServiceStatus) (*api.VaultService, error) {
	vault, err := vs.vaultsCRCli.Get(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	if reflect.DeepEqual(vault.Status, status) {
		return vault, nil
	}
	vault.Status = status
	_, err = vs.vaultsCRCli.Update(ctx, vault)
	return vault, err
}
