# Resources

The vault-operator creates the following Kubernetes resources to set up a Vault cluster:
* A Custom Resource for the etcd cluster storage backend
* A Deployment for Vault instances
* A Service to serve Vault client requests
* TLS Secrets for the etcd-cluster and Vault
* A Configmap to store the Vault configuration

## Labels

All of the above resources created for a Vault cluster have the following labels:

- `app=vault`
- `vault_cluster=<cluster-name>`

where `<cluster-name>` is the name of the Vault cluster to which that resource belongs.

## Ownership

For all the above resources their `metadata.ownerReferences` field points to the Vault Custom Resource to which they belong.
