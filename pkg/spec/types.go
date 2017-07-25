package spec

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	VaultResourceKind   = "Vault"
	VaultResourcePlural = "vaults"
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
	// Number of instances to deploy for a Vault deployment.
	// Default: 1.
	Replicas int32 `json:"replicas,omitempty"`

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

	// Endpoints of available replicas.
	// Avaliable replica is a running Vault pod, but not necessarily unsealed and ready
	// to serve requests.
	AvailableReplicas []string `json:"availableReplicas"`

	// Endpoints of ready Vault replicas. Ready replicas are unsealed and ready to serve
	// requests.
	ReadyReplicas []string `json:"readyReplicas"`

	// Endpoints of Sealed Vault replicas. Sealed replicas MUST be manually unsealed to
	// be ready to serve requests.
	SealedReplicas []string `json:"sealedReplicas"`
}
