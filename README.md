# Vault Operator

### Project status: beta
The basic features have been completed, and while no breaking API changes are currently planned, the API can change in a backwards incompatible way before the project is declared stable.

## Overview
The Vault operator deploys and manages [Vault][vault] clusters on Kubernetes. Vault instances created by the Vault operator are highly available and support automatic failover and upgrade.


## Getting Started

### Prerequisites

- Kubernetes 1.8+

### Configuring RBAC

Consult the [RBAC guide](./doc/user/rbac.md) on how to configure RBAC for the Vault operator.


### Deploying the etcd operator

The Vault operator employs the [etcd operator][etcd-operator] to deploy an etcd cluster as the storage backend.

1. Create the etcd operator Custom Resource Definitions (CRD):

    ```
    kubectl create -f example/etcd_crds.yaml
    ``` 
2. Deploy the etcd operator:

    ```sh
    kubectl -n default create -f example/etcd-operator-deploy.yaml
    ```

### Deploying the Vault operator

1. Create the Vault CRD:

    ```
    kubectl create -f example/vault_crd.yaml
    ```

2. Deploy the Vault operator:

    ```
    kubectl -n default create -f example/deployment.yaml
    ```

3. Verify that the operators are running:    

      ```
      $ kubectl -n default get deploy
      NAME             DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
      etcd-operator    1         1         1            1           5m
      vault-operator   1         1         1            1           5m
      ```


### Deploying a Vault cluster

A Vault cluster can be deployed by creating a `VaultService` Custom Resource(CR). For each Vault cluster the Vault operator will also create an etcd cluster for the storage backend.

1. Create a Vault CR that deploys a 2 node Vault cluster in high availablilty mode:

    ```
    kubectl -n default create -f example/example_vault.yaml
    ```

2. Wait until the `example-...` pods for the etcd and Vault cluster are up:

    ```
    $ kubectl -n default get pods
    NAME                              READY     STATUS    RESTARTS   AGE
    etcd-operator-78899f87f6-qdn5h    3/3       Running   0          10m
    example-7678c8f49c-kfx2w          1/2       Running   0          2m
    example-7678c8f49c-pqrj8          1/2       Running   0          2m
    example-etcd-7lpjg7n76d           1/1       Running   0          2m
    example-etcd-dhxrksssgx           1/1       Running   0          2m
    example-etcd-s7mzhffz92           1/1       Running   0          2m
    vault-operator-5976f74f84-pxkf6   1/1       Running   0          10m
    ```

3. Get the Vault pods:

    ```
    $ kubectl -n default get pods -l app=vault,vault_cluster=example
    NAME                       READY     STATUS    RESTARTS   AGE
    example-7678c8f49c-kfx2w   1/2       Running   0          2m
    example-7678c8f49c-pqrj8   1/2       Running   0          2m
    ```

4. Check the Vault CR status:

    ```
    $ kubectl -n default get vault example -o yaml
    apiVersion: vault.security.coreos.com/v1alpha1
    kind: VaultService
    metadata:
        name: example
        namespace: default
        ...
    spec:
        nodes: 2
        version: 0.9.1-0
        ...
    status:
        initialized: false
        phase: Running
        updatedNodes:
        - example-7678c8f49c-kfx2w
        - example-7678c8f49c-pqrj8
        vaultStatus:
            active: ""
            sealed:
            - example-7678c8f49c-kfx2w
            - example-7678c8f49c-pqrj8
            standby: null
        ...
    ```

    The Vault CR status shows the cluster is currently uninitialized and sealed.

### Using the Vault cluster

See the [Vault usage guide](./doc/user/vault.md) on how to initialize, unseal, and use the deployed Vault cluster.

Consult the [monitoring guide](./doc/user/monitoring.md) on how to monitor and alert on a Vault cluster with Prometheus.

See the [recovery guide](./doc/user/recovery.md) on how to backup and restore Vault cluster data using the etcd opeartor

For an overview of the default TLS configuration or how to specify custom TLS assets for a Vault cluster see the [TLS setup guide](doc/user/tls_setup.md).

### Uninstalling Vault operator

1. Delete the Vault Service:

    ```
    kubectl -n default delete -f example/example_vault.yaml
    ```

2. Delete the operators and rbac:

    ```
    kubectl -n default delete deploy vault-operator etcd-operator
    kubectl -n default delete -f example/rbac.yaml
    ```

2. Delete the CRDs:

    ```
    kubectl -n default delete -f example/vault_crd.yaml
    kubectl -n default delete -f example/etcd_crds.yaml
    ```

[vault]: https://www.vaultproject.io/
[etcd-operator]: https://github.com/coreos/etcd-operator/
