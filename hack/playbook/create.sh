#!/usr/bin/env bash

# Run in project root dir

# If you don't have etcd running, do:
#   kubectl create -f https://raw.githubusercontent.com/coreos/etcd-operator/master/example/deployment.yaml
#   kubectl create -f $PWD/hack/playbook/etcd.yaml

kubectl create configmap example-vault-config --from-file=$PWD/hack/playbook/vault.hcl
# Assume vault operator is running
kubectl create -f $PWD/example/example_vault.yaml
