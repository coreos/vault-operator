package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultBaseImage = "quay.io/coreos/vault"
	// version format is "<upstream-version>-<our-version>"
	defaultVersion = "0.8.3-0"
)

type VaultServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VaultService `json:"items"`
}

type VaultService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              VaultServiceSpec   `json:"spec"`
	Status            VaultServiceStatus `json:"status,omitempty"`
}

type VaultServiceSpec struct {
	// Number of nodes to deploy for a Vault deployment.
	// Default: 1.
	Nodes int32 `json:"nodes,omitempty"`

	// Base image to use for a Vault deployment.
	BaseImage string `json:"baseImage"`

	// Version of Vault to be deployed.
	Version string `json:"version"`

	// Name of the ConfigMap for Vault's configuration
	// If this is empty, operator will create a default config for Vault.
	// If this is not empty, operator will create a new config overwriting
	// the "storage", "listener" sections in orignal config.
	ConfigMapName string `json:"configMapName"`

	// TLS policy of vault nodes
	TLS *TLSPolicy `json:"TLS,omitempty"`
}

// SetDefaults sets the default vaules for the vault spec and returns true if the spec was changed
func (v *VaultService) SetDefaults() bool {
	changed := false
	vs := &v.Spec
	if vs.Nodes == 0 {
		vs.Nodes = 1
		changed = true
	}
	if len(vs.BaseImage) == 0 {
		vs.BaseImage = defaultBaseImage
		changed = true
	}
	if len(vs.Version) == 0 {
		vs.Version = defaultVersion
		changed = true
	}
	if vs.TLS == nil {
		vs.TLS = &TLSPolicy{Static: &StaticTLS{
			ServerSecret: DefaultVaultServerTLSSecretName(v.Name),
			ClientSecret: DefaultVaultClientTLSSecretName(v.Name),
		}}
		changed = true
	}
	return changed
}

type VaultServiceStatus struct {
	// Initialized indicates if the Vault service is initialized.
	Initialized bool `json:"initialized"`

	// PodNames of available nodes.
	// Avaliable node is a running Vault pod, but not necessarily unsealed and ready
	// to serve requests.
	AvailableNodes []string `json:"availableNodes"`

	// PodNames of the active Vault node. Active node is unsealed.
	// Only active node can serve requests.
	// Vault service only points to the active node.
	ActiveNode string `json:"activeNode"`

	// PodNames of the standby Vault nodes. Standby nodes are unsealed.
	// Standby nodes do not process requests, and instead redirect to the active Vault.
	StandbyNodes []string `json:"standbyNodes"`

	// PodNames of Sealed Vault nodes. Sealed nodes MUST be manually unsealed to
	// become standby or leader.
	SealedNodes []string `json:"sealedNodes"`

	// PodNames of updated Vault nodes. Updated means the Vault container image version
	// matches the spec's version.
	UpdatedNodes []string `json:"updatedNodes"`
}

// DefaultVaultClientTLSSecretName returns the name of the default vault client TLS secret
func DefaultVaultClientTLSSecretName(vaultName string) string {
	return vaultName + "-default-vault-client-tls"
}

// DefaultVaultServerTLSSecretName returns the name of the default vault server TLS secret
func DefaultVaultServerTLSSecretName(vaultName string) string {
	return vaultName + "-default-vault-server-tls"
}
