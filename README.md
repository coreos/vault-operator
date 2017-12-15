# Vault Operator

An operator to create and manage Vault instances for Kubernetes clusters on Tectonic. Vault instances created by the Vault operator are highly available and support automatic failover and upgrade.

For an overview of the resources created by the vault operator see the [resource labels and ownership](doc/user/resource_labels_and_ownership.md) doc.

An example namespace, `default`, is used in this document.

## Prerequisites

* [Tectonic 1.7+](https://coreos.com/tectonic) is installed
* `kubectl` is installed

## Getting Started

Verify `kubectl` is configured to use a 1.7+ Kubernetes cluster:

```sh
$ kubectl version | grep "Server Version"
Server Version: version.Info{Major:"1", Minor:"7", GitVersion:"v1.7.1+coreos.0", GitCommit:"fdd5383472eb43e60d2222503f03c76445e49899", GitTreeState:"clean", BuildDate:"2017-07-18T19:44:47Z", GoVersion:"go1.8.3", Compiler:"gc", Platform:"linux/amd64"}
```

### Creating a namespace

Create the namespace `default`:

```
kubectl create namespace default
```

### Configuring RBAC

By default, the Vault operator has no privilege to access any resources in Tectonic. Configure RBAC rules to grant access to the Vault operator.

1. Generate a RBAC yaml file from the template given in the repository:

    ```sh
    sed 's/<kube-ns>/default/g' example/rbac-template.yaml > example/rbac.yaml
    ```

2. Create the RBAC role:

    ```sh
    kubectl -n default create -f example/rbac.yaml
    ```

    The RBAC rule grants the `default` service account in the `default` namespace
    access to all resources under `default` namespace, but not outside.


### Deploying the etcd operator

The Vault operator employs etcd operator to deploy an etcd cluster as the storage backend. There is no etcd operator installed by default.
Deploy one into the `default` namespace:

```sh
kubectl -n default create -f example/etcd-operator-deploy.yaml
```

### Deploying the Vault operator

Create Vault Custom Resource Definition (CRD):

```
kubectl create -f ./example/vault_crd.yaml
```

Vault operator image is private. Use "quay.io" pull secret to get the image.

1. Create pull secret from the existing `coreos-pull-secret`:

    ```
    kubectl get secrets -n tectonic-system -o yaml coreos-pull-secret | sed 's/tectonic-system/default/g' | kubectl create -f -
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

2. Ensure that `example-vault-...` pods are up:

    ```
    $ kubectl -n default get pods
    NAME                              READY     STATUS    RESTARTS   AGE
    etcd-operator-346152359-34pwm     1/1       Running   0          43m
    example-vault-1003480066-b757c    0/1       Running   0          36m
    example-vault-1003480066-jzmwd    0/1       Running   0          36m
    example-vault-etcd-0000           1/1       Running   0          37m
    example-vault-etcd-0001           1/1       Running   0          37m
    example-vault-etcd-0002           1/1       Running   0          36m
    vault-operator-1388630079-7g04c   1/1       Running   0          37m
    ```

3. Print the Vault pods:

    ```
    $ kubectl -n default get pods -l app=vault,name=example-vault
    NAME                              READY     STATUS    RESTARTS   AGE
    example-vault-1003480066-b757c    0/1       Running   0          36m
    example-vault-1003480066-jzmwd    0/1       Running   0          36m
    ```

4. Verify that the Vault nodes can be viewed in the "vault" resource status:

      ```
      $ kubectl -n default get vault example-vault -o jsonpath='{.status.nodes.sealed}'
      [example-vault-1003480066-b757c example-vault-1003480066-jzmwd]
      ```

      Vault is unready because it is uninitialized and sealed.

For information on using the deployed Vault, see [vault.md](./doc/user/vault.md) .

#### Monitoring with Prometheus

By default the vault-operator will configure each vault pod to publish [statsd](https://www.vaultproject.io/docs/configuration/telemetry.html) metrics.

The vault-operator runs a [statsd-exporter](https://github.com/prometheus/statsd_exporter) container inside each vault pod to convert and expose those metrics in the format for Prometheus.

`curl` the `/metrics` endpoint for any available vault pod to get the Prometheus metrics:

```sh
$ VPOD=$(kubectl -n default get vault example-vault -o jsonpath='{.status.nodes.available[0]}')
$ kubectl -n default exec -ti ${VPOD} --container=vault -- curl localhost:9102/metrics
# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 2.7675e-05
go_gc_duration_seconds{quantile="0.25"} 5.5892e-05
go_gc_duration_seconds{quantile="0.5"} 5.7992e-05
go_gc_duration_seconds{quantile="0.75"} 7.804e-05
go_gc_duration_seconds{quantile="1"} 0.000185847
go_gc_duration_seconds_sum 0.000660497
go_gc_duration_seconds_count 9
. . .
```

### Uninstalling Vault operator

1. Delete the Vault resource and configuration:

    ```
    kubectl -n default delete -f example/example_vault.yaml
    ```

2. Wait for around 20s to make sure etcd and Vault pods are deleted.

3. Delete operators and other resources:

    ```
    kubectl -n default delete deploy vault-operator etcd-operator
    kubectl -n default delete secret coreos-pull-secret
    kubectl -n default delete -f example/rbac.yaml
    ```
