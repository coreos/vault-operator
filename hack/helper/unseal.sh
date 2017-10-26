#!/usr/bin/env bash

# This script unseals all sealed nodes of a vault cluster

set -o errexit
set -o nounset
set -o pipefail

# TODO: Use command line arguments
: ${KUBE_NS:?"Need to set KUBE_NS"}
: ${VAULT_CLUSTER_NAME:?"Need to set VAULT_CLUSTER_NAME"}
: ${UNSEAL_KEY:?"Need to set UNSEAL_KEY"}

# Get all sealed nodes
SEALED_NODES=$(kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} -o jsonpath='{.status.nodes.sealed}' | sed 's/^.\(.*\).$/\1/' )
if [ "${SEALED_NODES}" == "nil" ]; then
    echo "No sealed nodes found"
    exit 0
fi
IFS=' ' read -r -a SEALED_ARRAY <<< "${SEALED_NODES}"


# Unseal all sealed nodes
for NODE in "${SEALED_ARRAY[@]}"
do
    echo "Unsealing ${NODE}"
    kubectl -n ${KUBE_NS} exec ${NODE} -- /bin/sh -c "VAULT_ADDR=https://localhost:8200 VAULT_SKIP_VERIFY=true vault unseal ${UNSEAL_KEY} > /dev/null"
done
