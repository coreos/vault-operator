#!/usr/bin/env bash

: ${TEST_NAMESPACE:?"Need to set TEST_NAMESPACE"}

function cleanup_all {
    pull_secret_cleanup
    rbac_cleanup
    kubectl delete namespace ${TEST_NAMESPACE}
}

function pull_secret_cleanup {
    : ${PULL_SECRET_PATH:?"Need to set PULL_SECRET_PATH"}
    kubectl -n ${TEST_NAMESPACE} delete -f ${PULL_SECRET_PATH}
}

function rbac_cleanup {
    kubectl -n ${TEST_NAMESPACE} delete -f example/rbac.yaml
}

function setup_all {
    kubectl create namespace $TEST_NAMESPACE
    rbac_setup
    pull_secret_setup
}

function pull_secret_setup {
    : ${PULL_SECRET_PATH:?"Need to set PULL_SECRET_PATH"}
    kubectl -n ${TEST_NAMESPACE} create -f ${PULL_SECRET_PATH}
}

function rbac_setup() {
    sed -e "s/<namespace>/${TEST_NAMESPACE}/g" -e "s/<service-account>/default/g" example/rbac-template.yaml > example/rbac.yaml
    kubectl -n ${TEST_NAMESPACE} create -f example/rbac.yaml
}

function setup_all_crds() {
    kubectl create -f example/vault_crd.yaml 2>/dev/null || :
    kubectl create -f example/etcd_crds.yaml 2>/dev/null || :
}

function copy_pull_secret() {
    kubectl get secrets -n tectonic-system -o yaml coreos-pull-secret | sed "s|tectonic-system|${TEST_NAMESPACE}|g" | kubectl create -f -
}
