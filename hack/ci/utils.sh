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
    sed "s/<kube-ns>/${TEST_NAMESPACE}/g" example/rbac-template.yaml > example/rbac.yaml
    kubectl -n ${TEST_NAMESPACE} create -f example/rbac.yaml
}
