#!/usr/bin/env bash

# Run in project root dir

# If you don't have etcd running, do:
#   kubectl create -f https://raw.githubusercontent.com/coreos/etcd-operator/master/example/deployment.yaml

kubectl create configmap example-vault-config --from-file=$PWD/hack/playbook/vault.hcl
# Assume vault operator is running
kubectl create -f $PWD/example/example_vault.yaml

# Use the following command to port-forward vault pod
#   kubectl get po -l app=vault -o jsonpath='{.items[*].metadata.name}' | xargs -0 -I {} kubectl port-forward {} 8200
