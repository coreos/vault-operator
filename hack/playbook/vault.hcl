storage "etcd" {
  address  = "http://example-vault-etcd-client:2379"
  etcd_api = "v3"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = "true"
}
