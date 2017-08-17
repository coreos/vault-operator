## Vault TLS Setup Guide

There are two ways to configure TLS on the vault servers for every vault cluster.

### Default TLS Assets

If the user does not specify the TLS assets for a cluster via the CR spec field `spec.TLS` then the operator will use a custom CA to generate self signed certificates for the vault servers in the cluster.

These default TLS assets will be generated and stored in the following secrets:
- `<vault-cluster-name>-default-vault-client-tls`: This secret has the file `vault-client-ca.crt` which is the CA certificate used to sign the vault server certificate. This CA can be used by vault clients to authenticate the cert presented by the vault server.
- `<vault-cluster-name>-default-vault-server-tls`: This secret has the files `server.crt` and `server.key` which are the TLS certificate and key used to configure TLS on the vault servers

For example, creating the following vault cluster with no TLS secrets specified:

```yaml
apiVersion: "vault.coreos.com/v1alpha1"
kind: "Vault"
metadata:
  name: example-vault
spec:
  nodes: 1
```

would give us the following default secrets:

```
$ kubectl get secrets
NAME                                        TYPE                                  DATA      AGE
example-vault-default-vault-client-tls      Opaque                                1         1m
example-vault-default-vault-server-tls      Opaque                                2         1m
```

### User Specified TLS Assets

Alternatively users can pass in their own TLS assets while creating a cluster by specifying the necessary client and server secrets via the following CR spec fields:
- `spec.TLS.static.clientSecret`: As above this secret must contain the CA certificate `vault-client-ca.crt` that was used to sign the server certificate.
- `spec.TLS.static.serverSecret`: This secret must have the files `server.crt` and `server.key` which are the TLS certificate and key for the vault server. The `server.crt` certificate must allow the following wildcard domains:
    - `localhost`
    - `*.<namespace>.pod`
    - `<vault-cluster-name>.<namespace>.svc`

The final CR spec should look like:

```yaml
apiVersion: "vault.coreos.com/v1alpha1"
kind: "Vault"
metadata:
  name: <vault-cluster-name>
spec:
  nodes: 1
  TLS:
    static:
      serverSecret: <server-secret-name>
      clientSecret: <client-secret-name>
```

There is a helper script [hack/tls-gen.sh](../../hack/tls-gen.sh) to generate the necessary TLS assets and bundle them into the required secrets.
The following are necessary for the script to run:
* `kubectl` is installed
* `cfssl` tools are installed: https://github.com/cloudflare/cfssl#installation
* `jq` tool is installed: https://stedolan.github.io/jq/download/

Run the following command by providing the environment variable values as necessary:

```bash
$ KUBE_NS=<namespace> SERVER_SECRET=<server-secret-name> CLIENT_SECRET=<client-secret-name> hack/tls-gen.sh
```

This would generate the required secrets in the desired namespace. For example:

```bash
$ KUBE_NS=vault-services SERVER_SECRET=vault-server-tls CLIENT_SECRET=vault-client-tls hack/tls-gen.sh
$ kubectl -n vault-services get secrets
NAME                  TYPE                                  DATA      AGE
vault-client-tls      Opaque                                1         1m
vault-server-tls      Opaque                                2         1m
```
