package spec

// TLSPolicy defines the TLS policy of the vault nodes
type TLSPolicy struct {
	// StaticTLS enables user to generate static x509 certificates and keys,
	// put them into Kubernetes secrets, and specify them here.
	Static *StaticTLS `json:"static,omitempty"`
}

type StaticTLS struct {
	// ServerSecret is the secret containing TLS certs used by each vault node
	// for the communication between the vault server and its clients.
	// The server secret should contain two files: server.crt and server.key
	// server.crt can optionally have the ca certificate concatenated inside it.
	// See https://www.vaultproject.io/docs/configuration/listener/tcp.html#tls_cert_file
	ServerSecret string `json:"serverSecret,omitempty"`
}
