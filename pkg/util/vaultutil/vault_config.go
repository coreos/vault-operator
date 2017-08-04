package vaultutil

import (
	"fmt"
	"path/filepath"

	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
)

// TODO: add TLS configs
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
