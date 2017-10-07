#!/usr/bin/env bash

# This script upgrades a vault cluster to the desired version, unseals the new upgraded nodes and checks if upgrade succeeded.

set -o errexit
set -o nounset
set -o pipefail

# TODO: Use command line arguments
: ${KUBE_NS:?"Need to set KUBE_NS"}
: ${VAULT_CLUSTER_NAME:?"Need to set VAULT_CLUSTER_NAME"}
: ${UNSEAL_KEY:?"Need to set UNSEAL_KEY"}
# TODO: Check current version and automatically alternate between the two versions: 0.8.3-0 and 0.8.3-1
: ${UPGRADE_TO:?"Need to set the vault version to upgrade to UPGRADE_TO"}

RETRY_INTERVAL=5

if ! kubectl version 1> /dev/null ; then
    echo "kubectl with kubeconfig needs to be setup"
    exit 1
fi

# Get the size of the cluster N
NUM_NODES=$(kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} -o jsonpath='{.spec.nodes}')

# Check the cluster conditions before upgrade can be performed: 1 active and 0 sealed nodes
# Check for 1 active node
# There must be an active node before upgrade in order for N sealed nodes to show up after upgrade. Otherwise if N=1 and it is sealed then after upgrade there will be 2 sealed nodes.
ACT_NODE=$(kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} -o jsonpath='{.status.activeNode}')
if [ -z "$ACT_NODE" ]; then
    echo "Vault cluster must have an active node before upgrading"
    exit 1
fi
# TODO: Check for 0 sealed nodes


# Upgrade the vault version
echo "Upgrading vault to ${UPGRADE_TO}"
kubectl apply -f <(kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} -o yaml | sed "s/version: .*/version: ${UPGRADE_TO}/g")

# Wait for N sealed nodes to show up before unsealing them
echo "Waiting for ${NUM_NODES} sealed nodes after upgrade..."
NUM_SEALED=-1
while [ "${NUM_SEALED}" -ne "${NUM_NODES}" ]
do
    sleep ${RETRY_INTERVAL}

    SEALED_NODES=$(kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} -o jsonpath='{.status.sealedNodes}' | sed 's/^.\(.*\).$/\1/' )
    IFS=' ' read -r -a SEALED_ARRAY <<< "${SEALED_NODES}"
    NUM_SEALED=${#SEALED_ARRAY[@]}
done

# Unseal all sealed nodes
KUBE_NS=${KUBE_NS} VAULT_CLUSTER_NAME=${VAULT_CLUSTER_NAME} UNSEAL_KEY=${UNSEAL_KEY} hack/helper/unseal.sh

# Wait until new active node is of the new version
echo "Waiting until active node is of new version ${UPGRADE_TO}"
IS_UPGRADED="false"
while [ "${IS_UPGRADED}" != "true" ]
do
    # Wait before retrying
    sleep ${RETRY_INTERVAL}

    # Get the active node name
    ACT_NODE=$(kubectl -n ${KUBE_NS} get vault ${VAULT_CLUSTER_NAME} -o jsonpath='{.status.activeNode}')
    if [ -z "$ACT_NODE" ]; then
        echo "No active node found in CR status. Retrying"
        continue
    fi

    # Get the image version tag of the active node's pod.
    # Retry if "kubectl get pod" fails for some reason (if the pod of the active node shown has been deleted)
    IMAGE=$( (kubectl -n ${KUBE_NS} get pod ${ACT_NODE} -o jsonpath='{.spec.containers[0].image}' 2> /dev/null) || echo "")
    if [ -z "$IMAGE" ]; then
        echo "Get active pod ${ACT_NODE} failed. Retrying."
        continue
    fi
    IFS=':' read -r -a IMAGE_TOK <<< "${IMAGE}"
    VERSION=${IMAGE_TOK[1]}

    echo "Current active node: (${ACT_NODE}), version: (${VERSION})"
    if [ "${VERSION}" == "${UPGRADE_TO}" ]; then
        IS_UPGRADED="true"
    fi
done

echo "Upgrade and unseal complete!"
