# Configuring Vault nodes

This document describes how to access a Vault node deployed by the Vault operator and configure an high availability (HA) Vault setup.
See [Vault operator][getting-started] for information on managing Vault instances by using the Vault Operator.

## Prerequisites

* [Vault Commands (CLI)][vault-cli] is installed
* [Vault operator][getting-started] is configured

## Initiating a sealed node

1. Configure port forwarding between the local machine and the first sealed Vault node:

    ```sh
    kubectl -n vault-services get vault example-vault -o jsonpath='{.status.sealedNodes[0]}' | xargs -0 -I {} kubectl -n vault-services port-forward {} 8200
    ```

2. Open a new terminal.
3. Export the following environment for [Vault CLI environment][vault-cli-env]:

    ```sh
    export VAULT_ADDR='https://localhost:8200'
    export VAULT_SKIP_VERIFY="true"
    ```

4. Verify that the Vault server is up using the Vault CLI:

    ```sh
    $ vault status

    * server is initialized
    ```

    The Vault CLI is ready to interact with the Vault server.
    For information on initializing and unsealing, see [Initializing the Vault][initialize-vault].


## Writing secret to the active node

When a node is unsealed, it becomes active and initialized. The active Vault holds the leader election lock.


1. Check the active Vault node:

    ```sh
    kubectl -n vault-services get vault example-vault -o jsonpath='{.status.activeNode}'
    ```

2. Configure port forwarding between the local machine and the active Vault node:

    ```sh
    kubectl -n vault-services get vault example-vault -o jsonpath='{.status.activeNode}' | xargs -0 -I {} kubectl -n vault-services port-forward {} 8200
    ```

3. Open a new terminal.
4. Export the following environment for [Vault CLI environment][vault-cli-env]:

    ```sh
    export VAULT_ADDR='https://localhost:8200'
    export VAULT_SKIP_VERIFY="true"
    ```

5. Write and read an example secret:

    ```sh
    $ vault write secret/foo value=bar

    $ vault read secret/foo

    Key             	Value
    ---             	-----
    refresh_interval	768h0m0s
    value           	bar
    ```

    Successful operations indicate that the active Vault node is serving requests.

## Accessing Vault on Kubernetes

Vault operator creates [Kubernetes services][k8s-services] for accessing Valut deployments.

The service always exposes the active Vault node. It hides failures by switching the service pointer to the current active node when failover occurs.

The name and namespace of the service are the same as the Vault resource. For example, if the Vault resource's name is `example-vault`  and the namespace is `vault-services`, the service's name and namespace will also be `example-vault` and `vault-services` respectively.

Applications in the Kubernetes pod network can access the service through `https://example-vault.vault-services.svc:8200`.

## Enabling high availability

Vault supports [HA mode][ha] for production usage. To enable HA mode, scale Vault nodes from one to two or more by modifying the custom resource:

```sh
kubectl -n vault-services get vault example-vault -o json | sed 's/"nodes": 1/"nodes": 2/g' | kubectl apply -f -
```

Wait until all the Vault nodes are up:

```sh
$ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.availableNodes}'
[example-vault-994933690-37ts8 example-vault-994933690-5v7c1]
```

The first Vault node that is unsealed becomes the active node. In an HA Vault setup, only one active node is allowed to exist and serve user requests. The other unsealed nodes become standby.

### Starting a standby Vault node

A standby Vault node is initialized and unsealed, but does not hold the leader election lock. The standby node cannot serve user requests. It forwards user requests to the active node. If the active node goes down, a standby node becomes the active node.


1. Unseal a sealed node by using the instructions given in [Initiating a sealed node](#initiating-a-sealed-node).

2. Verify that the node becomes standby:

    ```sh
    $ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.standbyNodes}'
    [example-vault-994933690-5v7c1]
    ```

    The setup now contains an active and a standby Vault nodes.

### Automated failover

In an HA Vault setup, when the active node goes down the standby node takes over the active role and starts serving client requests.

To see how it works, perform the following:

1. Terminate the active Vault node:

    ```
    kubectl -n vault-services get vault example-vault -o jsonpath='{.status.activeNode}' | xargs -0 -I {} kubectl -n vault-services delete po {}
    ```

    The standby node becomes active.

2. Verify that the node is active:

    ```
    $ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.activeNode}'
    example-vault-994933690-5v7c1
    ```

    The previous port forward session should be terminated.

3. Create a new port forward session between the local machine and the new active node.

   See [Initiating a sealed node](#initiating-a-sealed-node) for more information.
   Successful operations indicate that automated failover works as expected.

### Failure recovery

Vault operator recovers any inactive or terminated Vault pods to maintain the size of cluster.

To see how it works, perform the following:

1. Ensure that a Vault node is terminated.

   A Vault node has already been terminated in the [Automated failover](#automated-failover) section.

2. Verify that a new Vault node is created:

    ```
    $ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.availableNodes}'
    [example-vault-994933690-5v7c1 example-vault-994933690-h066h]
    ```

3. Verify that the newly created Vault node is sealed:

    ```
    $ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.sealedNodes}'
    [example-vault-994933690-h066h]
    ```

    A new Vault node is created to replace the terminated one. Unseal the node and continue using HA.

[getting-started]: ../../README.md#getting-started
[ha]: https://www.vaultproject.io/docs/concepts/ha.html
[initialize-vault]: https://www.vaultproject.io/intro/getting-started/deploy.html#initializing-the-vault
[vault-cli]: https://www.vaultproject.io/docs/install/index.html
[vault-cli-env]: https://www.vaultproject.io/docs/commands/environment.html
[k8s-services]: https://kubernetes.io/docs/concepts/services-networking/service/
