package operator

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
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
func (vs *Vaults) monitorAndUpdateStaus(ctx context.Context, vr *spec.Vault) {
	tlsConfig, err := vs.readClientTLSFromSecret(vr)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS config for vault client: %v", err))
	}

	s := spec.VaultStatus{}

	for {
		select {
		case err := <-ctx.Done():
			logrus.Infof("stopped monitoring vault: %s (%v)", vr.GetName(), err)
			return
		case <-time.After(10 * time.Second):
		}

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
		availableNodes = append(availableNodes, p.GetName())

		cfg := vaultapi.DefaultConfig()
		podURL := fmt.Sprintf("https://%s:8200", k8sutil.PodDNSName(p.Status.PodIP, namespace))
		cfg.Address = podURL
		cfg.ConfigureTLS(tlsConfig)
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
	vault.Status = status
	_, err = vs.vaultsCRCli.Update(ctx, vault)
	return err
}

func (vs *Vaults) readClientTLSFromSecret(vr *spec.Vault) (*vaultapi.TLSConfig, error) {
	secret, err := vs.kubecli.CoreV1().Secrets(vr.GetNamespace()).Get(vr.Spec.TLS.Static.ClientSecret, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("read client tls failed: failed to get secret (%s): %v", vr.Spec.TLS.Static.ClientSecret, err)
	}

	// Read the secret and write ca.crt to a temporary file
	caCertData := secret.Data[spec.CATLSCertName]
	if err := os.MkdirAll(k8sutil.VaultTLSAssetDir, 0700); err != nil {
		return nil, fmt.Errorf("read client tls failed: failed to make dir: %v", err)
	}
	caCertFile := path.Join(k8sutil.VaultTLSAssetDir, spec.CATLSCertName)
	err = ioutil.WriteFile(caCertFile, caCertData, 0600)
	if err != nil {
		return nil, fmt.Errorf("read client tls failed: write ca cert file failed: %v", err)
	}
	return &vaultapi.TLSConfig{CACert: caCertFile}, nil
}
