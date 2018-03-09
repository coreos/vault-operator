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

const (
	// Name of CA cert file in the client secret
	CATLSCertName = "vault-client-ca.crt"
)

// TLSPolicy defines the TLS policy of the vault nodes
type TLSPolicy struct {
	// StaticTLS enables user to use static x509 certificates and keys,
	// by putting them into Kubernetes secrets, and specifying them here.
	// If this is not set, operator will auto-gen TLS assets and secrets.
	Static *StaticTLS `json:"static,omitempty"`
}

type StaticTLS struct {
	// ServerSecret is the secret containing TLS certs used by each vault node
	// for the communication between the vault server and its clients.
	// The server secret should contain two files: server.crt and server.key
	// The server.crt file should only contain the server certificate.
	// It should not be concatenated with the optional ca certificate as allowed by https://www.vaultproject.io/docs/configuration/listener/tcp.html#tls_cert_file
	// The server certificate must allow the following wildcard domains:
	// localhost
	// *.<namespace>.pod
	// <vault-cluster-name>.<namespace>.svc
	ServerSecret string `json:"serverSecret,omitempty"`
	// ClientSecret is the secret containing the CA certificate
	// that will be used to verify the above server certificate
	// The ca secret should contain one file: vault-client-ca.crt
	ClientSecret string `json:"clientSecret,omitempty"`
}

// IsTLSConfigured checks if the vault TLS secrets have been specified by the user
func IsTLSConfigured(tp *TLSPolicy) bool {
	if tp == nil || tp.Static == nil {
		return false
	}
	return len(tp.Static.ServerSecret) != 0 && len(tp.Static.ClientSecret) != 0
}
