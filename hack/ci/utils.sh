#!/usr/bin/env bash

: ${TEST_NAMESPACE:?"Need to set TEST_NAMESPACE"}
: ${PULL_SECRET_PATH:?"Need to set PULL_SECRET_PATH"}

function cleanup_all {
    pull_secret_cleanup
    rbac_cleanup
    kubectl delete namespace ${TEST_NAMESPACE}
}

function pull_secret_cleanup {
    kubectl -n ${TEST_NAMESPACE} delete -f ${PULL_SECRET_PATH}
}

function rbac_cleanup {
    kubectl delete clusterrolebinding "etcd-operator-${TEST_NAMESPACE}"
    kubectl delete clusterrole "etcd-operator-${TEST_NAMESPACE}"
    kubectl delete clusterrolebinding "vault-operator-${TEST_NAMESPACE}"
    kubectl delete clusterrole "vault-operator-${TEST_NAMESPACE}"
}

function setup_all {
    kubectl create namespace $TEST_NAMESPACE
    rbac_setup
    pull_secret_setup
}

function pull_secret_setup {
    kubectl -n ${TEST_NAMESPACE} create -f ${PULL_SECRET_PATH}
}

function rbac_setup() {
    # Create ClusterRole for vault-operator
    # TODO: Make the rules specific to vault-operator
    cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: "vault-operator-${TEST_NAMESPACE}"
rules:
- apiGroups:
  - "*"
  resources:
  - "*"
  verbs:
  - "*"
EOF

    # Create ClusterRoleBinding in test namespace for vault-operator
    cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: "vault-operator-${TEST_NAMESPACE}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: "vault-operator-${TEST_NAMESPACE}"
subjects:
- kind: ServiceAccount
  name: default
  namespace: $TEST_NAMESPACE
EOF


    # Create ClusterRole for etcd-operator
    cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: "etcd-operator-${TEST_NAMESPACE}"
rules:
- apiGroups:
  - etcd.database.coreos.com
  resources:
  - etcdclusters
  verbs:
  - "*"
- apiGroups:
  - apiextensions.k8s.io
  resources:
  - customresourcedefinitions
  verbs:
  - "*"
- apiGroups:
  - storage.k8s.io
  resources:
  - storageclasses
  verbs:
  - "*"
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - endpoints
  - persistentvolumeclaims
  - events
  verbs:
  - "*"
- apiGroups:
  - apps
  resources:
  - deployments
  verbs:
  - "*"
- apiGroups:
  - ""
  resources:
  - secrets
  - configmaps
  verbs:
  - get
EOF

    # Create ClusterRoleBinding in test namespace for etcd-operator
    cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: "etcd-operator-${TEST_NAMESPACE}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: "etcd-operator-${TEST_NAMESPACE}"
subjects:
- kind: ServiceAccount
  name: default
  namespace: $TEST_NAMESPACE
EOF
}
