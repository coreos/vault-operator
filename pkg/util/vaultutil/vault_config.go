package vaultutil

import (
	"bytes"
	"fmt"
	"path/filepath"

	vaultapi "github.com/hashicorp/vault/api"
)

const (
	// VaultTLSAssetDir is the dir where vault's server TLS and etcd TLS assets sits
	VaultTLSAssetDir = "/run/vault/tls/"
	// ServerTLSCertName is the filename of the vault server cert
	ServerTLSCertName = "server.crt"
	// ServerTLSKeyName is the filename of the vault server key
	ServerTLSKeyName = "server.key"
)

var listenerFmt = `
listener "tcp" {
  address     = "0.0.0.0:8200"
  cluster_address = "0.0.0.0:8201"
  tls_cert_file = "%s"
  tls_key_file  = "%s"
}
`

var etcdStorageFmt = `
storage "etcd" {
  address = "%s"
  etcd_api = "v3"
  ha_enabled = "true"
  tls_ca_file = "%s"
  tls_cert_file = "%s"
  tls_key_file = "%s"
  sync = "false"
}
`

// NewConfigWithDefaultParams appends to given config data some default params:
// - telemetry setting
// - tcp listener
func NewConfigWithDefaultParams(data string) string {
	buf := bytes.NewBufferString(data)
	buf.WriteString(`
telemetry {
	statsd_address = "localhost:9125"
}
`)

	listenerSection := fmt.Sprintf(listenerFmt,
		filepath.Join(VaultTLSAssetDir, ServerTLSCertName),
		filepath.Join(VaultTLSAssetDir, ServerTLSKeyName))
	buf.WriteString(listenerSection)

	return buf.String()
}

// NewConfigWithEtcd returns the new config data combining
// original config and new etcd storage section.
func NewConfigWithEtcd(data, etcdURL string) string {
	storageSection := fmt.Sprintf(etcdStorageFmt, etcdURL, filepath.Join(VaultTLSAssetDir, "etcd-client-ca.crt"),
		filepath.Join(VaultTLSAssetDir, "etcd-client.crt"), filepath.Join(VaultTLSAssetDir, "etcd-client.key"))
	data = fmt.Sprintf("%s%s", data, storageSection)
	return data
}

func NewClient(hostname string, port string, tlsConfig *vaultapi.TLSConfig) (*vaultapi.Client, error) {
	cfg := vaultapi.DefaultConfig()
	podURL := fmt.Sprintf("https://%s:%s", hostname, port)
	cfg.Address = podURL
	cfg.ConfigureTLS(tlsConfig)
	return vaultapi.NewClient(cfg)
}
