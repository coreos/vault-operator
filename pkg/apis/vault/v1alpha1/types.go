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

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultBaseImage = "quay.io/coreos/vault"
	// version format is "<upstream-version>-<our-version>"
	defaultVersion = "0.9.1-0"
	defaultConfigMapHash = "QC4hpsVMtN/vk+w2Htu6+eyxslwD0QITxA5SNGW6CrPb8gtLwRGEsJ0wBl9F4Gg0x569GV"
)

type ClusterPhase string

const (
	ClusterPhaseInitial ClusterPhase = ""
	ClusterPhaseRunning              = "Running"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VaultServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VaultService `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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

	// configMapHash of Vault configmap.
	ConfigMapHash string `json:"configMapHash"`

	// Pod defines the policy for pods owned by vault operator.
	// This field cannot be updated once the CR is created.
	Pod *PodPolicy `json:"pod,omitempty"`

	// Name of the ConfigMap for Vault's configuration
	// If this is empty, operator will create a default config for Vault.
	// If this is not empty, operator will create a new config overwriting
	// the "storage", "listener" sections in orignal config.
	ConfigMapName string `json:"configMapName"`

	// TLS policy of vault nodes
	TLS *TLSPolicy `json:"TLS,omitempty"`
}

// PodPolicy defines the policy for pods owned by vault operator.
type PodPolicy struct {
	// Resources is the resource requirements for the containers.
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
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
	if len(vs.ConfigMapHash) == 0 {
		vs.ConfigMapHash = defaultConfigMapHash
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
	// Phase indicates the state this Vault cluster jumps in.
	// Phase goes as one way as below:
	//   Initial -> Running
	Phase ClusterPhase `json:"phase"`

	// Initialized indicates if the Vault service is initialized.
	Initialized bool `json:"initialized"`

	// ServiceName is the LB service for accessing vault nodes.
	ServiceName string `json:"serviceName,omitempty"`

	// ClientPort is the port for vault client to access.
	// It's the same on client LB service and vault nodes.
	ClientPort int `json:"clientPort,omitempty"`

	// VaultStatus is the set of Vault node specific statuses: Active, Standby, and Sealed
	VaultStatus VaultStatus `json:"vaultStatus"`

	// PodNames of updated Vault nodes. Updated means the Vault container image version
	// matches the spec's version.
	UpdatedNodes []string `json:"updatedNodes,omitempty"`
}

type VaultStatus struct {
	// PodName of the active Vault node. Active node is unsealed.
	// Only active node can serve requests.
	// Vault service only points to the active node.
	Active string `json:"active"`

	// PodNames of the standby Vault nodes. Standby nodes are unsealed.
	// Standby nodes do not process requests, and instead redirect to the active Vault.
	Standby []string `json:"standby"`

	// PodNames of Sealed Vault nodes. Sealed nodes MUST be manually unsealed to
	// become standby or leader.
	Sealed []string `json:"sealed"`
}

// DefaultVaultClientTLSSecretName returns the name of the default vault client TLS secret
func DefaultVaultClientTLSSecretName(vaultName string) string {
	return vaultName + "-default-vault-client-tls"
}

// DefaultVaultServerTLSSecretName returns the name of the default vault server TLS secret
func DefaultVaultServerTLSSecretName(vaultName string) string {
	return vaultName + "-default-vault-server-tls"
}
