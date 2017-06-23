storage "etcd" {
  address  = "https://example-client.default.svc.cluster.local:2379"
  etcd_api = "v3"
  tls_ca_file = "/run/vault-etcd/ca.pem"
  tls_cert_file = "/run/vault-etcd/client.pem"
  tls_key_file = "/run/vault-etcd/client-key.pem"
}

listener "tcp" {
  address     = "localhost:8200"
  tls_cert_file = "/run/vault-listener/server.pem"
  tls_key_file = "/run/vault-listener/server-key.pem"
}
