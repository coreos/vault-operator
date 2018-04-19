# Vault backup/restore workflow

Vault operator works in conjunction with etcd operator to create an etcd backed Vault.
The etcd backup operator can be used to backup Vault's data by backing up its etcd cluster. 
The etcd restore operator can then be used to restore Vault to a previous state by restoring its etcd cluster.

## Prerequisite

* [Vault Commands (CLI)][vault-cli] installed
* Before beginning, create the [example][example_vault] Vault cluster that is initialized and unsealed.

## Write a secret

Before writing a secret [initialize][vault_init] and [unseal][vault_unseal] the vault cluster.

Then [write the secret][vault_write] `secret/foo value=bar` to the active node.

Verify that the secret `secret/foo value=bar` is written into Vault:

```sh
$ vault read secret/foo
Key             	Value
---             	-----
refresh_interval	768h0m0s
value           	bar
```

## Backup Vault's etcd cluster

[Create the AWS secret][set_aws] named `aws` in the default namespace so that the backup operator can access the S3 bucket.

Then create the following EtcdBackup CR to perform a backup on vault's etcd cluster to the S3 path `mybucket/vault.etcd.backup`:

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

## Kill the etcd cluster

Simulate a complete etcd cluster failure by deleting all etcd pods for vault's etcd cluster:

```sh
kubectl delete pod -l app=etcd,etcd_cluster=example-etcd --force --grace-period=0
```

Wait until `example-etcd` cluster pods are gone:

```sh
$ kubectl get pods -l app=etcd,etcd_cluster=example-etcd
No resources found.
```

## Restore etcd cluster

With previous Vault cluster's state saved to `mybucket/vault.etcd.backup` on S3, the etcd restore operator can restore `example-etcd` cluster from the saved backup.

Create the following EtcdRestore CR to perform a restore of vault's etcd cluster from the backup `mybucket/vault.etcd.backup`:

```sh
$ sed -e 's|<full-s3-path>|mybucket/vault.etcd.backup|g' \
    -e 's|<aws-secret>|aws|g' \
    -e 's|<restore-name>|example-etcd|g' \
    example/etcd_restore/restore_cr.yaml \
    | kubectl create -f -
etcdrestore "example-etcd" created
```

Wait until the etcd pods for vault's etcd cluster `example-etcd` are running again:

```sh
$ kubectl get pods -l app=etcd
NAME                READY     STATUS    RESTARTS   AGE
example-etcd-gxkmr9ql7z   1/1       Running   0          2m
example-etcd-m6g62x6mwc   1/1       Running   0          2m
example-etcd-rqk62l46kw   1/1       Running   0          2m
```

## Verify restored etcd cluster

Configure port forwarding between the local machine and the active Vault node:

```sh
kubectl get vault example -o jsonpath='{.status.vaultStatus.active}' | xargs -0 -I {} kubectl port-forward {} 8200
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
[vault_write]:https://github.com/coreos/vault-operator/blob/master/doc/user/vault.md#writing-secrets-to-the-active-node
[vault_unseal]:https://github.com/coreos/vault-operator/blob/master/doc/user/vault.md#unsealing-a-sealed-node
[vault_init]:https://github.com/coreos/vault-operator/blob/master/doc/user/vault.md#initializing-a-vault-cluster
[vault_configured]:https://github.com/coreos/vault-operator/blob/master/README.md#getting-started
[readme]:https://github.com/coreos/vault-operator/blob/master/README.md
[vault-cli]: https://www.vaultproject.io/docs/install/index.html
[example_vault]:./example_vault.yaml
