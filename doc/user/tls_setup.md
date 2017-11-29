# Setting up TLS for Vault

This document describes the two methods to configure TLS on the Vault servers for a Vault cluster.

## Using the default TLS assets

If the TLS assets for a cluster are not specified using the custom resource (CR) specification field, `spec.TLS`, the operator creates a default CA and uses it to generate self-signed certificates for the Vault servers in the cluster.

These default TLS assets are stored in the following secrets:

* `<vault-cluster-name>-default-vault-client-tls`: This secret contains the `vault-client-ca.crt` file, which is the CA certificate used to sign the Vault server certificate. This CA can be used by the Vault clients to authenticate the certificate presented by the Vault server.

* `<vault-cluster-name>-default-vault-server-tls`: This secret contains the `server.crt` and `server.key` files. These are the TLS certificate and key used to configure TLS on the Vault servers.

For example, create a Vault cluster with no TLS secrets specified using the following specification:

```yaml
apiVersion: "vault.security.coreos.com/v1alpha1"
kind: "VaultService"
metadata:
  name: example-vault
spec:
  nodes: 1
```

The following default secrets are generated for the above Vault cluster:

```
$ kubectl get secrets
NAME                                        TYPE                                  DATA      AGE
example-vault-default-vault-client-tls      Opaque                                1         1m
example-vault-default-vault-server-tls      Opaque                                2         1m
```

## Using custom TLS assets

Users may pass in custom TLS assets while creating a cluster. Specify the client and server secrets in the following CR specification fields:

* `spec.TLS.static.clientSecret`: This secret contains the `vault-client-ca.crt` file, which is the CA certificate used to sign the Vault server certificate. This CA can be used by the Vault clients to authenticate the certificate presented by the Vault server.

* `spec.TLS.static.serverSecret`: This secret contains the `server.crt` and `server.key` files. These are the TLS certificate and key for the Vault server. The `server.crt` certificate allows the following wildcard domains:

    - `localhost`
    - `*.<namespace>.pod`
    - `<vault-cluster-name>.<namespace>.svc`

The final CR specification is given below:

```yaml
apiVersion: "vault.security.coreos.com/v1alpha1"
kind: "VaultService"
metadata:
  name: <vault-cluster-name>
spec:
  nodes: 1
  TLS:
    static:
      serverSecret: <server-secret-name>
      clientSecret: <client-secret-name>
```

## Generating TLS assets

Use the [hack/tls-gen.sh][hack-tls] script to generate the necessary TLS assets and bundle them into required secrets.

### Prerequisites

* `kubectl` installed
* [cfssl][cfssl] tools installed
* [jq][jq] tool installed

### Using tls-gen script

Run the following command by providing the environment variable values as necessary:

```bash
$ KUBE_NS=<namespace> SERVER_SECRET=<server-secret-name> CLIENT_SECRET=<client-secret-name> hack/tls-gen.sh
```

Successful execution generates the required secrets in the desired namespace.

For example:

```bash
$ KUBE_NS=vault-services SERVER_SECRET=vault-server-tls CLIENT_SECRET=vault-client-tls hack/tls-gen.sh
$ kubectl -n vault-services get secrets
NAME                  TYPE                                  DATA      AGE
vault-client-tls      Opaque                                1         1m
vault-server-tls      Opaque                                2         1m
```


[cfssl]: https://github.com/cloudflare/cfssl#installation
[jq]: https://stedolan.github.io/jq/download/
[hack-tls]: https://github.com/coreos-inc/vault-operator/tree/master/hack/tls-gen.sh
