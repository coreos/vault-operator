package vaultutil

import (
	"fmt"
	"path/filepath"

	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
)

const (
	// ServerTLSCertName is the filename of the vault server cert
	ServerTLSCertName = "server.crt"
	// ServerTLSKeyName is the filename of the vault server key
	ServerTLSKeyName = "server.key"
)

var listenerFmt = `
listener "tcp" {
  address     = "0.0.0.0:8200"
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

// NewConfigWithListener appends the Listener to Vault config data.
func NewConfigWithListener(data string) string {
	listenerSection := fmt.Sprintf(listenerFmt,
		filepath.Join(k8sutil.VaultTLSAssetDir, ServerTLSCertName),
		filepath.Join(k8sutil.VaultTLSAssetDir, ServerTLSKeyName))
	data = fmt.Sprintf("%s\n%s\n", data, listenerSection)
	return data
}
