#!/usr/bin/env bash

# Run in project root dir

kubectl delete -f $PWD/example/example_vault.yaml
kubectl delete configmap example-vault-config
