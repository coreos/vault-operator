# Vault backup/restore workflow

Vault operator works in conjunction with etcd operator to create an etcd backed Vault.
The etcd backup operator can be used to backup Vault's data by backing up its etcd cluster. 
The etcd restore operator can then be used to restore Vault to a previous state by restoring its etcd cluster.

## Prerequisite
* Vault Commands (CLI) installed
* existing `coreos-pull-secret` for pulling vault image.

> Note: all Vault, etcd, etcd backup, and etcd restore operators must be deployed in same namespace. This doc uses namespace `default`.

## Deploy etcd and Vault Operators

First, deploy a vault operator in preparation of creating a vault cluster:

1. [Configuring RBAC][config_rbac]
2. [Deploying the Vault operator][deploy_vault_operator]

Second, deploy a etcd operator which works with vault operator to create etcd backed Vault:

1. [Deploying the etcd operator][deploy_etcd_operator]
2. Create etcdcluster CRD if there isn't one:

```sh
$ kubectl create -f example/etcd_cluster_crd.yaml
customresourcedefinition "etcdclusters.etcd.database.coreos.com" created
```

Verify that etcd and vault operators are running:

```sh
$ kubectl get deploy
NAME             DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
etcd-operator    1         1         1            1           1d
vault-operator   1         1         1            0           6s
```

## Setup up a Vault cluster

With a Vault Operator running, create and initialize a vault cluster in preparation for backup:

1. [Deploying a Vault cluster][deploy_vault]
2. [Initializing a Vault cluster][vault_init] before performing any operations.
3. [Unsealing a sealed node][vault_unseal]
4. [Writing `secret/foo value=bar` to the active node][vault_write]

Verify that the secret `secret/foo value=bar` is written into Vault:

```sh
$ vault read secret/foo
Key             	Value
---             	-----
refresh_interval	768h0m0s
value           	bar
```

## Deploy backup operator

Since the Vault instance is backed by an etcd cluster, etcd backup operator can backup vault's etcd cluster to S3.

1. Create the backup CRD: 
    ```sh
    kubectl create -f example/etcd_backup/backup_operator_crd.yaml
    ```
2. Deploy etcd backup operator: 
    ```sh
    kubectl create -f example/etcd_backup/backup_operator_deploy.yaml
    ```
3. [Set up AWS secret `aws`][set_aws] in this namespace.

Verify that backup operator is running:

```sh
$ kubectl get pods -l name=etcd-backup-operator
NAME                                    READY     STATUS    RESTARTS   AGE
etcd-backup-operator-65b56b54cd-tqd94   1/1       Running   0          1m
```

## Backup Vault

Once backup operator is running, perform a backup on vault's etcd cluster to S3 `mybucket/vault.etcd.backup`:

```sh
$ sed -e 's|<full-s3-path>|mybucket/vault.etcd.backup|g' \
    -e 's|<aws-secret>|aws|g' \
    -e 's|<tls-secret>|example-etcd-client-tls|g' \
    -e 's|<etcd-cluster-endpoints>|"https://example-etcd-client:2379"|g' \
    example/etcd_backup/backup_cr.yaml \
    | kubectl create -f -
```

Verify that backup is saved to S3:

```sh
$ aws s3 ls mybucket
2017-12-21 15:45:27      49184 vault.etcd.backup
```

## Kill etcd cluster

Simulate a complete etcd cluster failure by deleting `example-etcd` etcd cluster CR:

```sh
$ kubectl delete etcdcluster example-etcd
etcdcluster "example-etcd" deleted
```

Wait until `example-etcd` cluster pods are gone:

```sh
$ kubectl get pods -l app=etcd
No resources found.
```

## Deploy etcd restore operator

With previous Vault cluster's state saved to `mybucket/vault.etcd.backup` on S3, etcd restore operator can restore `example-etcd` cluster from the saved backup.

1. Create the restore CRD: 
    ```sh
    kubectl create -f example/etcd_restore/restore_operator_crd.yaml
    ```
2. Deploy etcd restore operator: 
    ```sh
    kubectl create -f example/etcd_restore/restore_operator_deploy.yaml
    ```

Verify that etcd restore operator is running:

```sh
$ kubectl get pods -l name=etcd-restore-operator
NAME                                     READY     STATUS    RESTARTS   AGE
etcd-restore-operator-5f687b9c6f-2cqjx   1/1       Running   0          1m
```

## Restore etcd cluster

Once etcd restore operator is running, perform a restore using S3 backup `mybucket/vault.etcd.backup` to create
`example-etcd` etcd cluster.

```sh
$ sed -e 's|<full-s3-path>|jenkins-testing-operator/vault.etcd.backup|g' \
    -e 's|<aws-secret>|aws|g' \
    -e 's|<restore-name>|example-etcd|g' \
    example/etcd_restore/restore_cr.yaml \
    | kubectl create -f -
etcdrestore "example-etcd" created
```

Wait until `example-etcd` pods are up running:

```sh
$ kubectl get pods -l app=etcd
NAME                READY     STATUS    RESTARTS   AGE
example-etcd-0000   1/1       Running   0          22h
example-etcd-0001   1/1       Running   0          22h
example-etcd-0002   1/1       Running   0          22h
```

Port forward an active vault node to the local machine:

```sh
$ kubectl get vault example -o jsonpath='{.status.nodes.active}' | xargs -0 -I {} kubectl port-forward {} 8200
```

In a separate terminal, verify that vault can retrieve the secret `secret/foo value=bar`:

```sh
$ vault read secret/foo
Key             	Value
---             	-----
refresh_interval	768h0m0s
value           	bar
```

[set_aws]:https://github.com/coreos/etcd-operator/blob/master/doc/user/walkthrough/backup-operator.md#setup-aws-secret
[deploy_vault]:https://github.com/coreos-inc/vault-operator/blob/master/README.md#deploying-vault
[deploy_vault_operator]:https://github.com/coreos-inc/vault-operator#deploying-the-vault-operator
[deploy_etcd_operator]:https://github.com/coreos-inc/vault-operator#deploying-the-etcd-operator
[config_rbac]:https://github.com/coreos-inc/vault-operator#configuring-rbac
[vault_write]:https://github.com/coreos-inc/vault-operator/blob/master/doc/user/vault.md#writing-secrets-to-the-active-node
[vault_unseal]:https://github.com/coreos-inc/vault-operator/blob/master/doc/user/vault.md#unsealing-a-sealed-node
[vault_init]:https://github.com/coreos-inc/vault-operator/blob/master/doc/user/vault.md#initializing-a-vault-cluster
[vault_configured]:https://github.com/coreos-inc/vault-operator/blob/master/README.md#getting-started
[backup_operator]:https://github.com/coreos/etcd-operator/blob/master/doc/user/walkthrough/backup-operator.md
[restore_operator]:https://github.com/coreos/etcd-operator/blob/master/doc/user/walkthrough/restore-operator.md
