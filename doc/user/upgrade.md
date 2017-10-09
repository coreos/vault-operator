# Upgrading Vault Cluster

TODO: Non-HA case.

This document describes how to upgrade an HA-enabled Vault cluster.
Vault operator simulates the suggested upgrade process as recommended
in official Vault upgrade docs:
  https://www.vaultproject.io/guides/upgrading/index.html#ha-installations

Prerequisites

- Before upgrading to a specific version, see the official Vault upgrade docs:
  https://www.vaultproject.io/guides/upgrading/index.html
- Read [vault.md](vault.md)

Assuming we will create the following Vault CR:

```yaml
apiVersion: "vault.security.coreos.com/v1alpha1"
kind: "VaultService"
metadata:
  name: "example-vault"
spec:
  nodes: 2
  version: "0.8.3-0"
```

After the Vault cluster is deployed and unsealed, there will be one active and one standby.

Upgrade Vault version to `0.8.3-1`:

```
kubectl -n vault-services get vault example-vault -o yaml | sed 's/version: 0.8.3-0/version: 0.8.3-1/g' | kubectl apply -f -
```

Vault operator will upgrade all nodes except the active node to keep service availability.
After upgrade, 2 Vault nodes of target version and 1 active node of old version exist.

Unseal all the upgraded nodes.

After all upgraded nodes are unsealed, Vault operator will enforce the old version active node
to step down and exit gracefully. One of the two new version standby nodes will take over and
become active.
