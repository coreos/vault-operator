> Note: This guide assumes the use of a [Tectonic][tectonic] cluster.

# Set up Vault-UI on Tectonic

[Vault-UI][vault-ui] is an open source project for managing and interacting with Vault through a web UI. Vault itself does not provide a web UI; all interactions are through either the Vault CLI or REST API calls.

### Prerequisites

* Before beginning, create the [example][example_vault] Vault cluster that is initialized and unsealed.
* Install and initialize Helm using the [Helm installation instructions][helm-install]

## Install Vault-UI

To install Vault-UI, install and configure the Helm chart provided in the Vault-UI repo.

Clone the Vault-UI repository:

```sh
git clone https://github.com/djenriquez/vault-ui
```

Modify the file `vault-ui/kubernetes/chart/vault-ui/templates/deployment.yaml` to add the `NODE_TLS_REJECT_UNAUTHORIZED` environment variable to allow self-signed HTTPS certificates.

<!-- {% raw %} -->
```sh
        env:
            - name: VAULT_URL_DEFAULT
              value: {{ .Values.vault.url }}
            - name: VAULT_AUTH_DEFAULT
              value: {{ .Values.vault.auth }}
            - name: NODE_TLS_REJECT_UNAUTHORIZED
              value: '0'
```
<!-- {% endraw %} -->

Next modify the file `vault-ui/kubernetes/chart/vault-ui/values.yaml` to configure the way to access the Vault-UI using Ingress, ClusterIP, LoadBalancer, or other access means.

**Configuration for Ingress:**

The following example will set up Vault-UI to be accessible at the Ingress host `vault-ui.ingress.staging.core-os.net` in the namespace `default`. Change the Ingress hostname and namespace as needed.

To use Ingress, first manually create a TLS certificate that will be used to set up the Ingress resource for Vault-UI, as described in the [Vault TLS setup guide][ingress-tls]:

```sh
KUBE_NS=default \
SERVER_SECRET=vault-ui-server-ingress-tls \
CLIENT_SECRET=vault-ui-client-ingress-tls \
SAN_HOSTS="vault-ui.ingress.staging.core-os.net" \
SERVER_CERT=tls.crt \
SERVER_KEY=tls.key \
hack/tls-gen.sh
```

Use the secrets listed above to modify the `values.yaml` file:

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
  url: https://example:8200
```

Use Helm to install Vault-UI within the `default` namespace:

```sh
$ cd vault-ui/kubernetes/chart/vault-ui
$ helm install . --namespace=default
```

## Accessing Vault-UI

When complete, the Helm installation will provide notes on how to access Vault-UI using port forwarding.

With the Ingress configuration, Vault-UI should be accessible at the Ingress host `vault-ui.ingress.staging.core-os.net`. Make sure to set up the DNS record for the Ingress host to make it accessible as described in the [Ingress guide][ingress-dns]. If you are unable to access Vault-UI via Ingress, try accessing it via port forwarding to isolate the issue.


[helm-install]: https://github.com/kubernetes/helm/blob/master/docs/install.md
[ingress-tls]: ingress.md#generate-custom-tls-assets-for-the-ingress-host
[ingress-dns]: ingress.md#create-dns-record-for-the-ingress-host
[vault-ui]: https://github.com/djenriquez/vault-ui
[example_vault]:../example_vault.yaml
