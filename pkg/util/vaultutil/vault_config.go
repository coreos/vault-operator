package vaultutil

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
)

// TODO: add TLS configs

const (
	serverTLSCertName = "server.crt"
	serverTLSKeyName  = "server.key"
)

var vaultServerTLSFmt = `
  tls_cert_file = "%s"
  tls_key_file  = "%s"
`

var etcdStorageFmt = `
storage "etcd" {
  address = "%s"
  etcd_api = "v3"
  ha_enabled = "true"
  tls_ca_file = "%s"
  tls_cert_file = "%s"
  tls_key_file = "%s"
}
`

// NewConfigWithEtcd returns the new config data combining
// original config and new etcd storage section.
func NewConfigWithEtcd(data, etcdURL string) string {
	storageSection := fmt.Sprintf(etcdStorageFmt, etcdURL, filepath.Join(k8sutil.VaultTLSAssetDir, "etcd-client-ca.crt"),
		filepath.Join(k8sutil.VaultTLSAssetDir, "etcd-client.crt"), filepath.Join(k8sutil.VaultTLSAssetDir, "etcd-client.key"))
	data = fmt.Sprintf("%s\n%s\n", data, storageSection)
	return data
}

// NewConfigWithTLS appends the TLS fields for vault
// to the original config.
func NewConfigWithTLS(data string) string {
	// TODO: Sanitize the config data by stripping the existing TLS section fields.
	// Or don't take in the vault.hcl file
	clientServerTLSFields := fmt.Sprintf(vaultServerTLSFmt, filepath.Join(k8sutil.VaultTLSAssetDir, serverTLSCertName),
		filepath.Join(k8sutil.VaultTLSAssetDir, serverTLSKeyName))
	listenerHeader := `listener "tcp" {`
	data = strings.Replace(data, listenerHeader, listenerHeader+clientServerTLSFields, 1)
	return data
}
