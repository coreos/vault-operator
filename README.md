# Vault Operator

An Operator for creating Vault instances.

## Getting Started

Create an etcd cluster with the Operator named `vault-etcd`

```
kubectl create -f vault.yaml -n kube-system
kubectl create configmap vault --from-file=vault.hcl -n kube-system
```

```
ns='kube-system' label='app=vault' kubectl -n $ns get pod -l $label -o jsonpath='{.items[1].metadata.name}' | xargs -I{} kubectl -n $ns port-forward {} 8200
```

```
export VAULT_ADDR='http://127.0.0.1:8200'
vault init
vault unseal
```


boom


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
