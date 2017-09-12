## Resources
Currently the vault-operator creates the following Kubernetes resources to setup a vault cluster:
- A Custom Resource for the etcd cluster storage backend
- A Deployment for vault instances
- A Service to serve vault client requests
- TLS Secrets for for the etcd-cluster and vault
- A Configmap to store the vault configuration

## Labels
All of the above resources created for a vault cluster have the following labels:

- `app=vault`
- `vault_cluster=<cluster-name>`

where `<cluster-name>` is the name of the vault cluster to which that resource belongs.

## Ownership
For all the above resources their `metadata.ownerReferences` field points to the vault Custom Resource to which they belong.
