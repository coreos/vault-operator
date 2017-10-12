## Upgrade failure testing

The process of [upgrading][upgrading-vault] a vault cluster involves some transactional logic. Specifically the vault operator will:
- Upgrade all nodes except the active node.
- Wait until all sealed nodes are unsealed
- Enforce the active node to step down and exit

The operator can fail and restart at any point during the upgrade process. Upon restart the operator should successfully be able to complete the upgrade process.

To test this behavior, failure points can be injected in the upgrade path at which the operator will fail with some specified probability.

The [chaos branch](https://github.com/coreos-inc/vault-operator/commit/e37eccf73bbed6f6736cfb42b58831233ba7463f) does this by causing the operator to fail at the following stages during the upgrade:
- Before upgrading the vault nodes
- After upgrading but before triggering the stepdown for the active node
- After triggering the stepdown


## Testing process

For this example the namespace `chaos` will be used. The RBAC rules and the pull secret should already be setup as described in [README][README].

### Setup the etcd and vault operator

```bash
$ kubectl -n chaos create -f https://raw.githubusercontent.com/coreos/etcd-operator/master/example/deployment.yaml
```

The vault-operator [chaos deployment][chaos-deployment] uses an image built from the [chaos branch](https://github.com/coreos-inc/vault-operator/commit/e37eccf73bbed6f6736cfb42b58831233ba7463f).

```
$ kubectl -n chaos create -f example/chaos/chaos-deployment.yaml
```

To repeatedly test the upgrade path the helper scripts provided at `hack/helper` can be used to create, initialize, unseal and upgrade a vault cluster.

### Create a vault cluster

Use [create-cluster.sh][create-cluster.sh] to create, initialize and unseal a vault cluster. The unseal key will be written to `_output/init_response.txt` and displayed in the output.
The script will wait until the vault cluster has an active node ready.

```bash
$ KUBE_NS="chaos" hack/helper/create-cluster.sh
```
```
Client Version: version.Info{Major:"1", Minor:"7", GitVersion:"v1.7.2", GitCommit:"922a86cfcd65915a9b2f69f3f193b8907d741d9c", GitTreeState:"clean", BuildDate:"2017-07-21T08:23:22Z", GoVersion:"go1.8.3", Compiler:"gc", Platform:"darwin/amd64"}
Server Version: version.Info{Major:"1", Minor:"7", GitVersion:"v1.7.1+coreos.0", GitCommit:"fdd5383472eb43e60d2222503f03c76445e49899", GitTreeState:"clean", BuildDate:"2017-07-18T19:44:47Z", GoVersion:"go1.8.3", Compiler:"gc", Platform:"linux/amd64"}
vaultservice "example-vault" created
Waiting for 2 sealed nodes...
Initializing vault

Unseal key and root token written to _output/init_response.txt
UNSEAL KEY: VQ/Tc9V73+t5NIPRJ52I5PvZtaN6NseDYxhtZ7kJPPs=
Unsealing example-vault-1395307387-51tq8
Unsealing example-vault-1395307387-b4r7k
Waiting for active node to show up
Waiting for active node to show up
Waiting for active node to show up
Waiting for active node to show up
Waiting for active node to show up
Vault cluster setup complete!
```

### Upgrade the vault cluster
Use [upgrade.sh][upgrade.sh] to upgrade the vault version. The script will upgrade the vault version in the CR, unseal the upgraded nodes, and wait until the active node is of the upgraded version.
Pass the unseal key and cluster name from the previous step to the script.

```bash
$ KUBE_NS="chaos" \
VAULT_CLUSTER_NAME="example-vault" \
UNSEAL_KEY="VQ/Tc9V73+t5NIPRJ52I5PvZtaN6NseDYxhtZ7kJPPs=" \
UPGRADE_TO="0.8.3-1" \
hack/helper/upgrade.sh
```
```
Upgrading vault to 0.8.3-1
vaultservice "example-vault" configured
Waiting for 2 sealed nodes after upgrade...
Unsealing example-vault-4065723276-cv1t3
Unsealing example-vault-4065723276-h1qn9
Waiting until active node is of new version 0.8.3-1
Current active node: (example-vault-1395307387-51tq8), version: (0.8.3-0)
Current active node: (example-vault-1395307387-51tq8), version: (0.8.3-0)
Current active node: (example-vault-1395307387-51tq8), version: (0.8.3-0)
Get active pod example-vault-1395307387-51tq8 failed. Retrying.
Get active pod example-vault-1395307387-51tq8 failed. Retrying.
Current active node: (example-vault-4065723276-cv1t3), version: (0.8.3-1)
Upgrade and unseal complete!
```

[README]: ../../README.md
[chaos-deployment]: ../../example/chaos/chaos-deployment.yaml
[upgrading-vault]: ../user/upgrade.md
[create-cluster.sh]: ../../hack/helper/create-cluster.sh
[upgrade.sh]: ../../hack/helper/upgrade.sh
