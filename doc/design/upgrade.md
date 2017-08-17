# HA Vault Upgrade

N = Number of replicas

Vault node status:
- One node is active (ready pod), and the other `N-1` are standby (unready pods).

Upgrade steps:
- Change Deployment spec: `maxUnavailable` to `N - 1`, `maxSurge` to 1, image to desired version.
  - `maxUnavailable=N-1` make sure the number of available/ready pods is at least 1.
    With this guarantee, it will upgrade and only upgrade all standby nodes.
- Wait until upgraded nodes unsealed and become standby.
- `/sys/step-down` call to active node.
  - This will transfer the leader lock to one of the standby nodes that should have been upgraded.
  - After stepping down, the active node will exit-program. Deployment will create a new pod with
    new image and finish the upgrade.

Failure recovery:
- If there are two replica sets, RS1 and RS2:
  RS1 version is desired version, but RS2's current size is non-zero,
  it means upgrade has encountered error.
- Just repeat above upgrade steps from scratch.

Limitation:
- No rollback.
- Assuming no other updates on spec while upgrading.
