package spec

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	VaultResourceKind   = "Vault"
	VaultResourcePlural = "Vaults"
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
}
