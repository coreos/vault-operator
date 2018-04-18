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
    kubectl create -f ./example/etcd_crds.yaml
    ``` 
2. Deploy the etcd operator:

    ```sh
    kubectl -n default create -f example/etcd-operator-deploy.yaml
    ```

### Deploying the Vault operator

1. Create the Vault CRD:

    ```
    kubectl create -f ./example/vault_crd.yaml
    ```

2. Deploy the Vault operator:

    ```
    kubectl -n default create -f example/deployment.yaml
    ```

    Wait for 10s until the Vault operator is up and running.

3. Verify that the operators are running:    

      ```
      $ kubectl -n default get deploy
      NAME             DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
      etcd-operator    1         1         1            1           1d
      vault-operator   1         1         1            0           6s
      ```


### Deploying Vault

#### Configuring TLS secrets

In this example, the Vault operator configures a default TLS setup for all the Vault pods in the cluster. For an overview of the default TLS configuration or how to specify specify custom TLS assets see the [TLS setup guide](doc/user/tls_setup.md).

#### Submitting Vault Custom Resource

In this example, a Vault cluster is configured with two nodes in high availability mode.

1. Create a Vault custom resource:

    ```
    kubectl -n default create -f example/example_vault.yaml
    ```

    Wait for around 20s.

2. Ensure that `example-...` pods are up:

    ```
    $ kubectl -n default get pods
    NAME                              READY     STATUS    RESTARTS   AGE
    etcd-operator-346152359-34pwm     1/1       Running   0          43m
    example-1003480066-b757c    0/1       Running   0          36m
    example-1003480066-jzmwd    0/1       Running   0          36m
    example-etcd-gxkmr9ql7z           1/1       Running   0          37m
    example-etcd-m6g62x6mwc           1/1       Running   0          37m
    example-etcd-rqk62l46kw           1/1       Running   0          36m
    vault-operator-1388630079-7g04c   1/1       Running   0          37m
    ```

3. Print the Vault pods:

    ```
    $ kubectl -n default get pods -l app=vault,name=example
    NAME                              READY     STATUS    RESTARTS   AGE
    example-1003480066-b757c    0/1       Running   0          36m
    example-1003480066-jzmwd    0/1       Running   0          36m
    ```

4. Verify that the Vault nodes can be viewed in the "vault" resource status:

      ```
      $ kubectl -n default get vault example -o jsonpath='{.status.vaultStatus.sealed}'
      [example-1003480066-b757c example-1003480066-jzmwd]
      ```

      Vault is unready because it is uninitialized and sealed.

### Using the Vault cluster

For information on using the deployed Vault cluster, see the [Vault usage guide](./doc/user/vault.md).

Consult the [monitoring guide](./doc/user/monitoring.md) on how to monitor and alert on a Vault cluster with Prometheus.

See the [recovery guide](./doc/user/recovery.md) on how to backup and restore Vault cluster data using the etcd opeartor

### Uninstalling Vault operator

1. Delete the Vault resource and configuration:

    ```
    kubectl -n default delete -f example/example_vault.yaml
    ```

2. Wait for around 20s to make sure etcd and Vault pods are deleted.

3. Delete operators and other resources:

    ```
    kubectl -n default delete deploy vault-operator etcd-operator
    kubectl -n default delete -f example/rbac.yaml
    ```

[vault]: https://www.vaultproject.io/
[etcd-operator]: https://github.com/coreos/etcd-operator/
