# Vault Operator

An Operator for managing Vault instances.

## Prerequisites

* [Tectonic 1.7+](https://coreos.com/tectonic) is installed
* `kubectl` is installed
* `vault` is installed: https://www.vaultproject.io/docs/install/index.html
* `cfssl` tools are installed: https://github.com/cloudflare/cfssl#installation
* `jq` tool is installed: https://stedolan.github.io/jq/download/


## Getting Started

Verify `kubectl` is configured to use a 1.7+ Kubernetes cluster:

```
$ kubectl version | grep "Server Version"
Server Version: version.Info{Major:"1", Minor:"7", GitVersion:"v1.7.1+coreos.0", GitCommit:"fdd5383472eb43e60d2222503f03c76445e49899", GitTreeState:"clean", BuildDate:"2017-07-18T19:44:47Z", GoVersion:"go1.8.3", Compiler:"gc", Platform:"linux/amd64"}
```

### Setup RBAC

In Tectonic cluster, "default" role has no access to any resource.
We need to setup RBAC rules to grant access to operators.

Replace `<my-kube-ns>` below with your current working namespace to
create a RBAC yaml:

```
sed 's/${KUBE_NS}/<my-kube-ns>/g' example/rbac-template.yaml > example/rbac.yaml
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

### Deploy Vault operator

Vault operator image is private. Using it requires "quay.io" pull secret.

Download "pull secret" from "account.coreos.com" page and save it as `config.json` file.

Encode it into base64 format (replace `<download_dir>` with the dir where config.json is downloaded):

```
base64 <download_dir>/config.json
```


Create a `pull_secret.yaml` (replace `<base64_encoded_pull_secret>` field with above result):

```yaml
apiVersion: v1
data:
  .dockerconfigjson: <base64_encoded_pull_secret>
kind: Secret
metadata:
  name: coreos-pull-secret
type: kubernetes.io/dockerconfigjson
```

Run the following:

```
kubectl create -f pull_secret.yaml
```

Deploy vault operator:

```
kubectl create -f example/deployment.yaml
```

Wait ~10s until vault operator is running:

```
$ kubectl get deploy
vault-operator   1         1         1            1           21h
```

Next we are going to deploy Vault server.

### Deploy Vault

#### Setup TLS secrets

There is a helper script `hack/tls-gen.sh` to generate the necessary TLS assets and bundle them into the required secrets.

Run the following command and replace `<my-kube-ns>` with your current working namespace:

```bash
KUBE_NS=<my-kube-ns> SERVER_SECRET=vault-server-tls CLIENT_SECRET=vault-client-tls hack/tls-gen.sh
```

This should create the two secrets needed for Vault server TLS:

```
$ kubectl get secrets
vault-client-tls      Opaque                                1         21h
vault-server-tls      Opaque                                2         21h
```

#### Submit Vault Custom Resource

Create a Vault config:

```
kubectl create configmap example-vault-config --from-file=hack/playbook/vault.hcl
```

Create a Vault custom resource:

```
kubectl create -f example/example_vault.yaml
```

**Wait ~20s.** Then you can see pods:

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
the above vault custom resource. **Wait ~20s until they are deleted.**
Then delete operators and rest resources:

```
kubectl delete deploy vault-operator etcd-operator
kubectl delete secret coreos-pull-secret
kubectl delete -f example/rbac.yaml
```
