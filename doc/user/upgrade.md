# Upgrading Vault Cluster

This document describes how to upgrade an HA-enabled Vault cluster.
Vault operator simulates the suggested upgrade process as recommended
in the official Vault documentation for [Upgrading Vault HA Installations][upgrade-ha].

## Prerequisites

* Before upgrading to a specific version, see the official Vault upgrade docs:
  [https://www.vaultproject.io/guides/upgrading/index.html][upgrade-vault]
* Read [Configuring Vault nodes][vault-md]

## Upgrade the Vault nodes

Create the following Vault CR to use as the basis for the upgrade:

```yaml
apiVersion: "vault.security.coreos.com/v1alpha1"
kind: "VaultService"
metadata:
  name: "example"
spec:
  nodes: 2
  version: "0.8.3-0"
```

After the Vault cluster is deployed and unsealed, there will be one active and one standby node.

Use `kubectl` to upgrade to Vault `0.9.0-0`:

```
kubectl -n default get vault example -o yaml | sed 's/version: 0.8.3-0/version: 0.9.0-0/g' | kubectl apply -f -
```

Vault-operator will upgrade all nodes except the active node to keep service availability.
After upgrade, 2 Vault nodes of the target version and 1 active node of the old version will exist.

## Unseal all the upgraded nodes

After all upgraded nodes are unsealed, vault-operator will enforce the old version active node
to step down and exit gracefully. One of the two new version standby nodes will take over and
become active.


[vault-md]: vault.md
[upgrade-ha]: https://www.vaultproject.io/guides/upgrading/index.html#ha-installations
[upgrade-vault]: https://www.vaultproject.io/guides/upgrading/index.html
