# Vault Operator

An Operator for managing Vault instances.

## Getting Started

### Setup RBAC

In Tectonic cluster, "default" role has no access to any resource.
We need to setup RBAC rules to grant access to operators.

Replace `<my-namespace>` below with your current working namespace to
create a RBAC yaml:

```
sed 's/${KUBE_NS}/<my-namespace>/g' example/rbac-template.yaml > example/rbac.yaml
```

Then create the RBAC role:

```
kubectl create -f example/rbac.yaml
```

This will give "admin" role to default users in your current namespace.
(We will provide more production-grade RBAC setup later.)

### Deploy etcd operator

Vault operator makes use of etcd operator to deploy etcd cluster as storage backend.
So we also need to deploy etcd operator:

```
kubectl create -f https://raw.githubusercontent.com/coreos/etcd-operator/master/example/deployment.yaml
```

### Deploy vault operator

Vault operator image is private. Using it requires "quay.io" pull secret.
Download "pull secret" from "account.coreos.com" page.
It looks like:
```
    "quay.io": {
      "auth": "YOUR_PULL_SECRET",
```

Replace `.dockerconfigjson` field value below with `YOUR_PULL_SECRET` from `auth` field above :

```yaml
apiVersion: v1
data:
  .dockerconfigjson: ${YOUR_PULL_SECRET}
kind: Secret
metadata:
  name: coreos-pull-secret
type: kubernetes.io/dockerconfigjson
```

Save above into `pull_secret.yaml` and create pull secret:

```
kubectl create -f pull_secret.yaml
```

Deploy vault operator:

```
kubectl create -f example/deployment.yaml
```

Wait ~10s until vault operator is running.

Create a Vault config:

```
kubectl create configmap example-vault-config --from-file=hack/playbook/vault.hcl
```

Create a Vault custom resource:

```
kubectl create -f example/example_vault.yaml
```

Wait ~20s. Then you can see pods:

```
$ kubectl get pods
NAME                             READY     STATUS    RESTARTS   AGE
etcd-operator-809151189-1j7jl    1/1       Running   0          2d
example-vault-613074584-5lwbg    0/1       Running   0          49s
example-vault-etcd-0000          1/1       Running   0          1m
example-vault-etcd-0001          1/1       Running   0          1m
example-vault-etcd-0002          1/1       Running   0          1m
vault-operator-146442885-gj98d   1/1       Running   0          1m
```

To get Vault pods only:

```
$ kubectl get pods -l app=vault,name=example-vault
NAME                            READY     STATUS    RESTARTS   AGE
example-vault-613074584-5lwbg   0/1       Running   0          8m
```

It is also viable to see all Vault nodes in "vault" resource status:

```
$ kubectl get vault example-vault -o jsonpath='{.status.sealedNodes}'
[https://10-2-1-16.hongchao-test.pod:8200]
```

Vault is unready since it is uninitialized and sealed.
To learn how to access Vault and turn it into ready state, check out [vault.md](./doc/user/vault.md) .


### Cleanup

Delete Vault resource and config:

```
kubectl delete -f example/example_vault.yaml
kubectl delete configmap example-vault-config
```

Vault operator will clean up other resources (vault and etcd instances) for 
the above vault custom resource. Wait ~20s until they are deleted.
Then delete vault and etcd operator:

```
kubectl delete deploy --all
kubectl delete -f example/rbac.yaml
```
