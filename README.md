# Vault Operator

An Operator for creating Vault instances.

## Getting Started

Install the CAs, deployments, etcd Operator, and Vault in the `default` namespace. It must be installed into the `default` namespace to make the CA certs work. You must have `kubectl get pods` working when running this script.

```
./install.sh
```

Proxy the vault service to your local machine so you can use the command line tooling to localhost.

```
export ns='default' label='app=vault'; kubectl -n $ns get pod -l $label -o jsonpath='{.items[0].metadata.name}' | xargs -I{} kubectl -n $ns port-forward {} 8200
```

Run `vault init` to get things going, this should print out the Vault unseal keys, unsealing Vault should get you a fully working setup!

```
export VAULT_ADDR='https://localhost:8200'
vault init --ca-cert=example-certs/ca.pem
vault unseal
```


### Debugging

```
export ns='kube-system' label='etcd_cluster=vault-etcd'; kubectl -n $ns get pods -l $label -o jsonpath='{.items[0].metadata.name}' | xargs -I{} kubectl -n $ns port-forward {} 2379
```

```
etcdctl get "" --prefix=true --keys-only
```

## Resources

- Install vault tool https://www.vaultproject.io/intro/getting-started/install.html
- Configure vault to use etcd https://www.vaultproject.io/docs/configuration/storage/etcd.html
