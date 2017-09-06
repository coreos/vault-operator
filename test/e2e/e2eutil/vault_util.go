package e2eutil

import (
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
)

// SingleKeyInitOptions returns the init options that generate a single secret to unseal vault
func SingleKeyInitOptions() *vaultapi.InitRequest {
	return &vaultapi.InitRequest{
		SecretShares:    1,
		SecretThreshold: 1,
	}
}

// NewVaultClient returns a vault client configured to make requests to the specified hostname and port
func NewVaultClient(hostname string, port string, tlsConfig *vaultapi.TLSConfig) (*vaultapi.Client, error) {
	cfg := vaultapi.DefaultConfig()
	url := fmt.Sprintf("https://%s:%s", hostname, port)
	cfg.Address = url
	cfg.ConfigureTLS(tlsConfig)
	return vaultapi.NewClient(cfg)
}
