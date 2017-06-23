.PHONY: cfssl ca req clean

CFSSL	=  cfssl
JSON	=  cfssljson

all: etcd-peer etcd-server etcd-client vault

cfssl:
	go get -u -tags nopkcs11 github.com/cloudflare/cfssl/cmd/cfssl
	go get -u github.com/cloudflare/cfssl/cmd/cfssljson
	go get -u github.com/mattn/goreman

vault:
	mkdir -p certs/vault
	$(CFSSL) gencert -initca config/ca-csr.json | $(JSON) -bare certs/vault/ca

	$(CFSSL) gencert \
	  -ca certs/vault/ca.pem \
	  -ca-key certs/vault/ca-key.pem \
	  -config config/ca-config.json \
	  config/vault-csr.json | $(JSON) -bare certs/vault/client

	$(CFSSL) gencert \
	  -ca certs/vault/ca.pem \
	  -ca-key certs/vault/ca-key.pem \
	  -config config/ca-config.json \
	  config/vault-csr.json | $(JSON) -bare certs/vault/server

etcd-peer:
	mkdir -p certs/etcd-peer
	$(CFSSL) gencert -initca config/ca-csr.json | $(JSON) -bare certs/etcd-peer/ca

	$(CFSSL) gencert \
	  -ca certs/etcd-peer/ca.pem \
	  -ca-key certs/etcd-peer/ca-key.pem \
	  -config config/ca-config.json \
	  config/etcd-csr.json | $(JSON) -bare certs/etcd-peer/peer

etcd-server:
	mkdir -p certs/etcd-server
	$(CFSSL) gencert -initca config/ca-csr.json | $(JSON) -bare certs/etcd-server/ca

	$(CFSSL) gencert \
	  -ca certs/etcd-server/ca.pem \
	  -ca-key certs/etcd-server/ca-key.pem \
	  -config config/ca-config.json \
	  config/etcd-csr.json | $(JSON) -bare certs/etcd-server/server

etcd-client:
	mkdir -p certs/etcd-client
	$(CFSSL) gencert -initca config/ca-csr.json | $(JSON) -bare certs/etcd-client/ca

	$(CFSSL) gencert \
	  -ca certs/etcd-client/ca.pem \
	  -ca-key certs/etcd-client/ca-key.pem \
	  -config config/ca-config.json \
	  config/etcd-csr.json | $(JSON) -bare certs/etcd-client/client


clean:
	rm -rf certs

