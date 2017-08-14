#!/usr/bin/env bash

# Run in project root dir

# If you don't have etcd operator running, do:
#   kubectl create -f https://raw.githubusercontent.com/coreos/etcd-operator/master/example/deployment.yaml

# Assume vault operator is running
kubectl create -f $PWD/example/example_vault.yaml

# Use the following command to port-forward vault pod
#   kubectl get po -l app=vault -o jsonpath='{.items[*].metadata.name}' | xargs -0 -I {} kubectl port-forward {} 8200
