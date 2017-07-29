listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = "true"
}

# 'storage' will be filled in by the vault operator
