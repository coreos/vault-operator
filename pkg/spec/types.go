package spec

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VaultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Vault `json:"items"`
}

type Vault struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              VaultSpec   `json:"spec"`
	Status            VaultStatus `json:"status,omitempty"`
}

type VaultSpec struct {
	// Number of nodes to deploy for a Vault deployment.
	// Default: 1.
	Nodes int32 `json:"nodes,omitempty"`

	// Base image to use for a Vault deployment.
	BaseImage string `json:"baseImage"`

	// Version of Vault to be deployed.
	Version string `json:"version"`

	// Name of the config map that configurates Vault.
	// The storage fields in the configuration will be ingored.
	ConfigMapName string `json:"configMapName"`

	// TLS policy of vault nodes
	TLS *TLSPolicy `json:"TLS,omitempty"`
}

// SetDefaults sets the default vaules for the vault spec.
// TODO: remove this when CRD support defaulting directly.
func (vs VaultSpec) SetDefaults() {
	if vs.Nodes == 0 {
		vs.Nodes = 1
	}
}

type VaultStatus struct {
	// Initialized indicates if the Vault service is initialized.
	Initialized bool `json:"initialized"`

	// Endpoints of available nodes.
	// Avaliable node is a running Vault pod, but not necessarily unsealed and ready
	// to serve requests.
	AvailableNodes []string `json:"availableNodes"`

	// Endpoint of the active Vault node. Active node is unsealed.
	// Only active node can serve requests.
	// Vault service only points to the active node.
	ActiveNode string `json:"activeNode"`

	// Endpoints of the standby Vault nodes. Standby nodes are unsealed.
	// Standby nodes do not process requests, and instead redirect to the active Vault.
	StandbyNodes string `json:"standbyNodes"`

	// Endpoints of Sealed Vault nodes. Sealed nodes MUST be manually unsealed to
	// become standby or leader.
	SealedNodes []string `json:"sealedNodes"`
}
