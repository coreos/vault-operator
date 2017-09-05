# HA Vault Upgrade

N = Number of replicas

Vault node status:
- One node is active (ready pod), and the other `N-1` are standby (unready pods).

We will add following fields to Vault status:

```
// PodNames of the up-to-date Vault Pods
UpdatedNodes []string
```

The status update goroutine will take care of it.

Upgrade steps:

- If Vault spec version is different from the deployment version:
  - Change Deployment spec: `maxUnavailable` to `N - 1`, `maxSurge` to 1, image to desired version.
    - `maxUnavailable=N-1` make sure the number of available/ready pods is at least 1.
      With this guarantee, it will upgrade and only upgrade all standby nodes.
- If there are `N` up-to-date standby Vault nodes and
  only one not up-to-date but active Vault node:
  - Kill the active node to trigger step-down
    - This will transfer the leader lock to one of the standby nodes that should have been upgraded.
    - After stepping down, the active node will exit-program.

The above steps should be reconcile-able.

Limitation:
- No rollback.
