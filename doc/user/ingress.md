# Setup Ingress for Vault Service

This guide shows how to make the vault service accessible from outside a k8s cluster by setting up an Ingress resource. For more information about ingress see the [tectonic ingress docs][tectonic ingress docs].

Have a vault cluster that is already initialized and unsealed. You can use the [create-cluster][create-cluster] script for a quick setup.

### Assumptions

- This example assumes a vault cluster named `example-vault` in the namespace `vault-services` whose service is accessible at `https://example-vault.vault-services.svc:8200` from inside the cluster.

- The ingress hostname used to access the vault service will be `vault.ingress.staging.core-os.net`.

- The tectonic cluster is on AWS.

Modify the example as needed for your use case.

## Generate custom TLS assets for the ingress host:

The ingress host can be configured with TLS assets for secure access.

Use the [tls-gen][tls-gen] script to generate the required TLS assets as secrets in the namespace of your vault cluster:

```sh
KUBE_NS=vault-services \
SERVER_SECRET=vault-server-ingress-tls \
CLIENT_SECRET=vault-client-ingress-tls \
SAN_HOSTS="vault.ingress.staging.core-os.net" \
SERVER_CERT=tls.crt \
SERVER_KEY=tls.key \
hack/tls-gen.sh
```

- `vault-server-ingress-tls`: secret that contains the ingress server certificate `tls.crt` and key `tls.key`
- `vault-client-ingress-tls`: secret that contains the CA certificate `vault-client-ca.crt` that can be used to verify the ingress host

## Create the ingress resource

Create the following ingress resource:

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

## Create DNS record for the ingress host

The traffic on the ingress host `vault.ingress.staging.core-os.net` needs to reach the ingress controller in the tectonic cluster.

To enable that a DNS alias record for `vault.ingress.staging.core-os.net` needs to be present which redirects traffic to the ELB of the tectonic cluster.

1. Find the DNS name of your tectonic ELB from the aws console. The ELB should be named `<tectonic-cluster-name>-con`.

2. Create a record set in the hosted zone for the ingress host. In this case we create the record set named `vault.ingress.k8s.staging.core-os.net` in the hosted zone `staging.core-os.net`. Choose the type `A` and select Alias as `Yes`. Set the Alias Target to the the DNS name of the ELB from the previous step.

## Access the vault service through the ingress host

Your vault CLI should now be able to successfully interact with the vault service through ingress host.

Set the following environment variables to access the vault service:

```sh
VAULT_TLS_SERVER_NAME=vault.ingress.staging.core-os.net
VAULT_ADDR=https://vault.ingress.staging.core-os.net
VAULT_TOKEN=<root token>
VAULT_SKIP_VERIFY=true
```

To verify the ingress server certificate first get the CA cert file `vault-client-ca.crt` from the `vault-client-ingress-tls` secret and base64 decode it into a local file, then set the following envs:
```sh
VAULT_CACERT=<path-to-ca-cert>
VAULT_SKIP_VERIFY=false
```


[tectonic ingress docs]: https://coreos.com/tectonic/docs/latest/admin/ingress.html
[create-cluster]: ../../hack/helper/create-cluster.sh
[tls-gen]: ../../hack/tls-gen.sh
