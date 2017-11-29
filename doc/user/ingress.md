# Set up Ingress for Vault Service

This guide shows how to make the Vault service accessible from outside a Kubernetes cluster by setting up an Ingress resource. For more information about Ingress see the [Tectonic Ingress docs][tectonic ingress docs].

Before beginning, create a Vault cluster that is initialized and unsealed. Use the [create-cluster][create-cluster] script for a quick setup.

### Assumptions

* This example assumes a Vault cluster named `example-vault` in the namespace `vault-services` whose service is accessible at `https://example-vault.vault-services.svc:8200` from inside the cluster.

* The Ingress hostname used to access the Vault service will be `vault.ingress.staging.core-os.net`.

* The Tectonic cluster is on AWS.

Modify the example as needed for your use case.

## Generate custom TLS assets for the Ingress host

The Ingress host can be configured with TLS assets for secure access.

Use the [tls-gen][tls-gen] script to generate the required TLS assets as secrets in the namespace of the Vault cluster:

```sh
KUBE_NS=vault-services \
SERVER_SECRET=vault-server-ingress-tls \
CLIENT_SECRET=vault-client-ingress-tls \
SAN_HOSTS="vault.ingress.staging.core-os.net" \
SERVER_CERT=tls.crt \
SERVER_KEY=tls.key \
hack/tls-gen.sh
```

* `vault-server-ingress-tls`: secret that contains the Ingress server certificate `tls.crt` and key `tls.key`
* `vault-client-ingress-tls`: secret that contains the CA certificate `vault-client-ca.crt` used to verify the Ingress host

## Create the Ingress resource

Create the following Ingress resource:

```yaml
kind: Ingress
apiVersion: extensions/v1beta1
metadata:
  name: vault
  namespace: vault-services
  annotations:
    ingress.kubernetes.io/secure-backends: 'true'
    kubernetes.io/ingress.class: tectonic
spec:
  tls:
    - hosts:
        - vault.ingress.staging.core-os.net
      secretName: vault-server-ingress-tls
  rules:
    - host: vault.ingress.staging.core-os.net
      http:
        paths:
          - path: /
            backend:
              serviceName: example-vault
              servicePort: 8200
```

## Create DNS record for the Ingress host

The traffic on the Ingress host `vault.ingress.staging.core-os.net` must reach the Ingress controller in the Tectonic cluster.

To enable this, create a DNS alias record for `vault.ingress.staging.core-os.net` which redirects traffic to the ELB of the Tectonic cluster.

1. Find the DNS name of your Tectonic ELB from the AWS console. The ELB should be named `<tectonic-cluster-name>-con`.

2. Create a record set in the hosted zone for the Ingress host. This example creates the record set named `vault.ingress.k8s.staging.core-os.net` in the hosted zone `staging.core-os.net`. Choose type: `A` and select Alias: `Yes`. Set the Alias Target to the the DNS name of the ELB from the previous step.

## Access the Vault service through the Ingress host

The Vault CLI should now be able to successfully interact with the Vault service through the Ingress host.

Set the following environment variables to access the Vault service:

```sh
VAULT_TLS_SERVER_NAME=vault.ingress.staging.core-os.net
VAULT_ADDR=https://vault.ingress.staging.core-os.net
VAULT_TOKEN=<root token>
VAULT_SKIP_VERIFY=true
```

To verify the Ingress server certificate, get the CA cert file `vault-client-ca.crt` from the `vault-client-ingress-tls` secret and base64 decode it into a local file. Then set the following envs:

```sh
VAULT_CACERT=<path-to-ca-cert>
VAULT_SKIP_VERIFY=false
```


[tectonic ingress docs]: https://coreos.com/tectonic/docs/latest/admin/ingress.html
[create-cluster]: https://github.com/coreos-inc/vault-operator/tree/master/hack/helper/create-cluster.sh
[tls-gen]: https://github.com/coreos-inc/vault-operator/tree/master/hack/tls-gen.sh
