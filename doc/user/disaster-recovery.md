# Disaster Recovery

## Theory of Operation

All Vault state, including the encrypted secrets protected by the sealing keys, is persisted in an etcd cluster. This etcd cluster holds redundant copies of the Vault state. Using the etcd Open Cloud Service to power the etcd cluster ensures that data is resiliant to individual etcd node failures.

Having an off-cluster cold storage backup for important services is an important best practice to follow.

The Vault and etcd Open Cloud Services were designed to work together to facilitate the suggested best practice of maintaining off-cluster cold storage for important services.


The Vault Open Cloud Service (OCS) will use the etcd OCS to take an etcd snapshot, and store that snapshot on an object storage, like AWS S3, to later be restored in an automated fashion in case of disaster. This snapshot functionality is fully consistent and based on etcd's [upstream disaster recovery][etcd-dr] tooling and best practices.

[etcd-dr]: https://coreos.com/etcd/docs/latest/op-guide/recovery.html

## Roadmap

This backup/restore functionality is currently built and in testing for the etcd OCS and is expected to be available Q1 2018 as part of the next Tectonic release.

The features will include:

- Point-in-time API driven backups of etcd state to object storage
- Restoration from backups of etcd and creation of Vault instances post restoration
- Documentation for creating regular point-in-time backups

If you have further questions or feel additional features are missing [please send us feedback on our roadmap](mailto:tectonic-alpha-feedback@coreos.com?Subject=Tectonic%20Vault%20OCS%20Roadmap%20Feedback).
