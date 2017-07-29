package vaultutil

import (
	"fmt"
)

var etcdStorageFmt = `
storage "etcd" {
  address = "%s"
  etcd_api = "v3"
}
`

// NewConfigWithEtcd returns the new config data combining
// original config and new etcd storage section.
func NewConfigWithEtcd(data, etcdURL string) string {
	storageSection := fmt.Sprintf(etcdStorageFmt, etcdURL)
	data = fmt.Sprintf("%s\n%s\n", data, storageSection)
	return data
}
