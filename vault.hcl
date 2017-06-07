storage "etcd" {
  address  = "http://vault-etcd:2379"
  etcd_api = "v3"
}

listener "tcp" {
  address     = "127.0.0.1:8200"
  tls_disable = 1
}
