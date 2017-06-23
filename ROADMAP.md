**Prototype TLS Setup**

- use etcd Operator test certificates
- hack up vault deployment to use certs
- hack up vault tool to us certs

**TLS Setup**

- Generate a CA and client certificates for etcd/valut
- Generate a CA and client certificates for vault/clients

refs:

- https://www.vaultproject.io/docs/configuration/storage/etcd.html#tls_ca_file

**HA Setups**

- Enable the HA setup in Vault and test it manually
- Test that the HA setup actually works with an integration test

refs:

- Add HA setup https://www.vaultproject.io/docs/configuration/storage/etcd.html#ha_enabled

**Define Vault Schema for CRD**
