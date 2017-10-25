# Setup Vault-UI on Tectonic

[Vault-UI](https://github.com/djenriquez/vault-ui) is an open source project for managing and interacting with Vault via a web UI. Vault by itself does not provide a web UI all interactions are done either using the Vault CLI or REST API calls.

### Prerequisites

- Setup an initialized and unsealed vault cluster. Use the [create-cluster][create-cluster] script for a quick setup.
- Installed and initialized Helm per [these instructions][helm-install]

This example assumes a vault cluster named `example-vault` in the namespace `vault-services`.

## Install the Vault-UI

The Vault UI can be setup by configuring and installing the Helm chart provided in the Vault UI repo.

Clone Vault-UI repository:
```sh
git clone https://github.com/djenriquez/vault-ui
```

Modify the file `vault-ui/kubernetes/chart/vault-ui/templates/deployment.yaml` to add the `NODE_TLS_REJECT_UNAUTHORIZED` environment variable to to allow self-signed HTTPS certificates.
```sh
        env:
            - name: VAULT_URL_DEFAULT
              value: {{ .Values.vault.url }}
            - name: VAULT_AUTH_DEFAULT
              value: {{ .Values.vault.auth }}
            - name: NODE_TLS_REJECT_UNAUTHORIZED
              value: '0'

```

Next modify the file `vault-ui/kubernetes/chart/vault-ui/values.yaml` to configure the way to access the Vault-UI e.g using Ingress, ClusterIP, LoadBalancer etc.

**Configuration for Ingress:**

The following example will setup the vault UI to be accessbile at the ingress host `vault-ui.ingress.staging.core-os.net` in the namespace `vault-services`.
Change the ingress hostname and namespace as needed.


If you use ingress you’ll need to manually create TLS certificate that will be used to setup the ingress resource for the Vault-UI as described in the [ingress setup][ingress-tls] guide:
```sh
KUBE_NS=vault-services \
SERVER_SECRET=vault-ui-server-ingress-tls \
CLIENT_SECRET=vault-ui-client-ingress-tls \
SAN_HOSTS="vault-ui.ingress.staging.core-os.net" \
SERVER_CERT=tls.crt \
SERVER_KEY=tls.key \
hack/tls-gen.sh
```

With the above secrets the `vaules.yaml` file should be modified to look like:

```yaml
replicaCount: 1
image:
  repository: djenriquez/vault-ui
  tag: latest
  pullPolicy: IfNotPresent
service:
  name: vault-ui
  type: ClusterIP
  externalPort: 8000
  internalPort: 8000
ingress:
  enabled: true
  hosts:
    - vault-ui.ingress.staging.core-os.net
  annotations:
    kubernetes.io/ingress.class: tectonic
    ingress.kubernetes.io/force-ssl-redirect: "true"
  tls:
    - secretName: vault-ui-server-ingress-tls
      hosts:
        - vault-ui.ingress.staging.core-os.net
resources: {}
vault:
  auth: TOKEN
  url: https://example-vault:8200
```

Use Helm to install the Vault-UI within the `vault-services` namespace:

```sh
$ cd vault-ui/kubernetes/chart/vault-ui
$ helm install . --namespace=vault-services
```

## Accessing the Vault-UI

After running Helm install it will give you notes on how to access Vault-UI via port-forwarding. If you are unable to access Vault-UI via Ingress you might want to try accessing it via port-forwarding to isolate the issue.

With the ingress configuration the vault UI should be accessible at the ingress host `vault-ui.ingress.staging.core-os.net`. Make sure to setup the DNS record for the ingress host to make it accessible as described in the [ingress guide][ingress-dns].


[create-cluster]: ../../hack/helper/create-cluster.sh
[helm-install]: https://github.com/kubernetes/helm/blob/master/docs/install.md
[ingress-tls]: ./ingress.md#generate-custom-tls-assets-for-the-ingress-host
[ingress-dns]: ./ingress.md#create-dns-record-for-the-ingress-host
