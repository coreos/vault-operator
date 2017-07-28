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
	Spec              VaultSpec    `json:"spec"`
	Status            *VaultStatus `json:"status,omitempty"`
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
}

type VaultStatus struct {
	// Initialized indicates if the Vault service is initialized.
	Initialized bool `json:"initialized"`

	// Endpoints of available nodes.
	// Avaliable node is a running Vault pod, but not necessarily unsealed and ready
	// to serve requests.
	AvailableNodes []string `json:"availableNodes"`

	// Endpoint of the leader Vault node. Leader node is unsealed.
	// Only leader node can serve requests.
	// Vault service only points to the leader node.
	LeaderNode string `json:"leaderNode"`

	// Endpoints of the standby Vault nodes. Standby nodes are unsealed.
	// Standby nodes cannot serve requests directly. All requests will
	// be directed to the leader node eventually through vault service.
	StandbyNodes string `json:"standbyNodes"`

	// Endpoints of Sealed Vault nodes. Sealed nodes MUST be manually unsealed to
	// become standby or leader.
	SealedNodes []string `json:"sealedNodes"`
}
