# Vault

This doc talks about how to access Vault node deployed by Vault operator, and HA setup.
This doc is based on [example setup](../../README.md#getting-started).

Prerequisites:

* Vault CLI is installed: https://www.vaultproject.io/docs/install/index.html

## Access The Sealed Node

Create port-forwarding between local machine and the first sealed Vault node:

```
kubectl -n vault-services get vault example-vault -o jsonpath='{.status.sealedNodes[0]}' | xargs -0 -I {} kubectl -n vault-services port-forward {} 8200
```

Open a new terminal.
Export the following env for [Vault CLI env](https://www.vaultproject.io/docs/commands/environment.html):

```
export VAULT_ADDR='https://localhost:8200'
export VAULT_SKIP_VERIFY="true"
```

Verify Vault server is up using Vault CLI:

```
$ vault status

* server is not yet initialized
```

Now it is possible to use Vault CLI to interact with Vault server.
Check out [the docs on how to initialize and unseal Vault](https://www.vaultproject.io/intro/getting-started/deploy.html#initializing-the-vault).


## Access The Active Node

Once the node is unsealed, there should be one active node.
An active Vault node is initialized, unsealed and holding the leader-election lock.

Check the active Vault node:

```
kubectl -n vault-services get vault example-vault -o jsonpath='{.status.activeNode}'
```

Create port-forwarding between local machine and the active Vault node:

```
kubectl -n vault-services get vault example-vault -o jsonpath='{.status.activeNode}' | xargs -0 -I {} kubectl -n vault-services port-forward {} 8200
```

Open a new terminal.
Export the following env for [Vault CLI env](https://www.vaultproject.io/docs/commands/environment.html):

```
export VAULT_ADDR='https://localhost:8200'
export VAULT_SKIP_VERIFY="true"
```

Try to write and read secret:

```
$ vault write secret/foo value=bar

$ vault read secret/foo

Key             	Value
---             	-----
refresh_interval	768h0m0s
value           	bar
```

If it succeeds, it means the active Vault node is serving requests.

## High Availability

Vault support [HA setup](https://www.vaultproject.io/docs/concepts/ha.html) for production usage.

To enable a HA setup, scale Vault nodes from 1 to 2 (or more) by modifying customer resource:

```
kubectl -n vault-services get vault example-vault -o json | sed 's/"nodes": 1/"nodes": 2/g' | kubectl apply -f -
```

Wait until all two Vault nodes show up:

```
$ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.availableNodes}'
[example-vault-994933690-37ts8 example-vault-994933690-5v7c1]
```

If a Vault node has been unsealed before, there should be one active node.
In a HA Vault setup, only one active node could exist, and only the active node can serve user requests.
The other unsealed nodes become standby.

### Start a standby Vault node

A standby Vault node is initialized, unsealed, and not holding leader-election lock.
The standby node cannot serve user requests, and will forward user requests to the active node.
If the active node dies, standby node will try to become the active node.

Unseal the other sealed node based on [Access The Sealed Node](#access-the-sealed-node) section.

Verify it becomes standby:

```
$ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.standbyNodes}'
[example-vault-994933690-5v7c1]
```

Now we have one active and one standby Vault nodes.

### Automated failover

In HA Vault setup, if the active node is down, the standby node will take over the active role and server client requests.

To see how it works, first kill the active Vault node:

```
kubectl -n vault-services get vault example-vault -o jsonpath='{.status.activeNode}' | xargs -0 -I {} kubectl -n vault-services delete po {}
```

Previous standby node will become active. Run the following command to check:

```
$ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.activeNode}'
example-vault-994933690-5v7c1
```

Previous port-forward session should have been broken.
Redo [Access The Active Node](#access-the-active-node) to port-forward to the new active node.
If succeeded, we have verified automated failover would work.

### Failure recovery

Vault operator will recover any dead Vault pod to maintain the size of cluster.

Verify there are two Vault nodes even if we killed one before:

```
$ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.availableNodes}'
[example-vault-994933690-5v7c1 example-vault-994933690-h066h]
```

Check that a new sealed node was created:

```
$ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.sealedNodes}'
[example-vault-994933690-h066h]
```

This is the newly created Vault node to replaced our previous killed one. Unseal it and enjoy HA.
