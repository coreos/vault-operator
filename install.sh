#!/bin/sh

# etcd Peer CA and certs
kubectl create secret generic etcd-server-peer-tls --from-file=peer-ca-crt.pem=example-certs/etcd-peer/ca.pem --from-file=peer-crt.pem=example-certs/etcd-peer/peer.pem --from-file=peer-key.pem=example-certs/etcd-peer/peer-key.pem

# etcd Server CA and certs
kubectl create secret generic etcd-server-client-tls --from-file=client-ca-crt.pem=example-certs/etcd-client/ca.pem --from-file=client-crt.pem=example-certs/etcd-server/server.pem --from-file=client-key.pem=example-certs/etcd-server/server-key.pem

# etcd Client CA and certs
kubectl create secret generic operator-etcd-client-tls --from-file=etcd-ca-crt.pem=example-certs/etcd-server/ca.pem --from-file=etcd-crt.pem=example-certs/etcd-client/client.pem --from-file=etcd-key.pem=example-certs/etcd-client/client-key.pem

# vault CA and certs
kubectl create secret generic vault-etcd-client-tls --from-file=example-certs/etcd-server/ca.pem --from-file=example-certs/etcd-client/client.pem --from-file=example-certs/etcd-client/client-key.pem
kubectl create secret generic vault-server-tls --from-file=example-certs/vault/ca.pem --from-file=example-certs/vault/server.pem  --from-file=example-certs/vault/server-key.pem
kubectl create secret generic vault-client-tls --from-file=example-certs/vault/ca.pem 

# install vault configuration
kubectl create configmap vault --from-file=vault.hcl

kubectl create -f manifests
