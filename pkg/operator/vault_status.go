// Copyright 2018 The vault-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operator

import (
	"context"
	"reflect"
	"time"

	api "github.com/coreos/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos/vault-operator/pkg/util/k8sutil"
	"github.com/coreos/vault-operator/pkg/util/vaultutil"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// monitorAndUpdateStatus monitors the vault service and replicas statuses, and
// updates the status resource in the vault CR item.
func (vs *Vaults) monitorAndUpdateStatus(ctx context.Context, vr *api.VaultService) {
	var tlsConfig *vaultapi.TLSConfig

	s := api.VaultServiceStatus{
		Phase:       api.ClusterPhaseRunning,
		ServiceName: vr.GetName(),
		ClientPort:  k8sutil.VaultClientPort,
	}

	for {
		// Do not wait to update Phase ASAP.
		latest, err := vs.updateVaultCRStatus(ctx, vr.GetName(), vr.GetNamespace(), s)
		if err != nil {
			logrus.Errorf("failed updating the status for the vault service: %s (%v)", vr.GetName(), err)
		}
		if latest != nil {
			vr = latest
		}

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
		vs.updateLocalVaultCRStatus(ctx, vr, &s, tlsConfig)
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
	var standByNodes []string
	var updated []string
	inited := false
	// If it can't talk to any vault pod, we are not going to change the status.
	changed := false

	for _, p := range pods.Items {
		// If a pod is Terminating, it is still Running but has no IP.
		if p.Status.Phase != v1.PodRunning || p.DeletionTimestamp != nil {
			continue
		}

		// Make a note of any updated pod, even if it's not healthy.
		// This ensures that `Vaults.syncUpgrade()` doesn't lose track of
		// unhealthy, but already updated nodes.  (Otherwise, an already
		// updated, active, but unhealthy node could mistakenly be killed to
		// "complete" an update.)
		// This alone does not indicate the pods have "changed".
		if k8sutil.IsVaultVersionMatch(p.Spec, vr.Spec) {
			updated = append(updated, p.GetName())
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

		// TODO: add to vaultutil?
		if hr.Initialized && !hr.Sealed && !hr.Standby {
			s.VaultStatus.Active = p.GetName()
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

	s.VaultStatus.Standby = standByNodes
	s.VaultStatus.Sealed = sealNodes
	s.Initialized = inited
	s.UpdatedNodes = updated
}

// updateVaultCRStatus updates the status field of the Vault CR.
func (vs *Vaults) updateVaultCRStatus(ctx context.Context, name, namespace string, status api.VaultServiceStatus) (*api.VaultService, error) {
	vault, err := vs.vaultsCRCli.VaultV1alpha1().VaultServices(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if reflect.DeepEqual(vault.Status, status) {
		return vault, nil
	}
	vault.Status = status
	_, err = vs.vaultsCRCli.VaultV1alpha1().VaultServices(namespace).Update(vault)
	return vault, err
}
