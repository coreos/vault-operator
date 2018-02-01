#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This script creates a vault cluster, initializes and unseals the nodes and prints out the unseal key

: ${KUBE_NS:?"Need to set KUBE_NS"}

RETRY_INTERVAL=5

kubectl version

# Setup vault cluster
kubectl -n ${KUBE_NS} create -f example/example_vault.yaml
# TODO: Get cluster name from CR
VAULT_CLUSTER_NAME="example"

# Wait for vault CR to appear
until kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} > /dev/null 2>&1;
do
    echo "Waiting for vault CR"
    sleep ${RETRY_INTERVAL}
done

# Get size of cluster N
NUM_NODES=$(kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} -o jsonpath='{.spec.nodes}')

# Wait for N sealed nodes
echo "Waiting for ${NUM_NODES} sealed nodes..."
NUM_SEALED=-1
while [ "${NUM_SEALED}" -ne "${NUM_NODES}" ]
do
    sleep ${RETRY_INTERVAL}

    SEALED_NODES=$(kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} -o jsonpath='{.status.vaultStatus.sealed}' | sed 's/^.\(.*\).$/\1/' )
    IFS=' ' read -r -a SEALED_ARRAY <<< "${SEALED_NODES}"
    NUM_SEALED=${#SEALED_ARRAY[@]}
done

# Init via the first sealed node
echo $'Initializing vault\n'
INIT_RESPONSE=$(kubectl -n ${KUBE_NS} exec ${SEALED_ARRAY[0]} \
                -- /bin/sh -c "VAULT_ADDR=https://localhost:8200 VAULT_SKIP_VERIFY=true vault init --key-shares=1 --key-threshold=1" | tr '\n' ' ')

# Write init response to file
mkdir -p _output
echo ${INIT_RESPONSE} > _output/init_response.txt
echo "Unseal key and root token written to _output/init_response.txt"

# Get the unseal key from the response
UNSEAL_KEY=$(echo "${INIT_RESPONSE}" | sed 's/Unseal Key 1: \(.*\) Initial Root Token: .*/\1/')
echo "UNSEAL KEY: ${UNSEAL_KEY}"

# Unseal all the sealed nodes
KUBE_NS=${KUBE_NS} VAULT_CLUSTER_NAME=${VAULT_CLUSTER_NAME} UNSEAL_KEY=${UNSEAL_KEY} hack/helper/unseal.sh

# Wait for active node to show up
ACT_NODE=""
while [ -z "${ACT_NODE}" ]
do
    echo "Waiting for active node to show up"
    sleep ${RETRY_INTERVAL}
    ACT_NODE=$(kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} -o jsonpath='{.status.vaultStatus.active}')
done

echo "Vault cluster setup complete!"
