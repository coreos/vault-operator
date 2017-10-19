# Vault UI

There is a project https://github.com/djenriquez/vault-ui .
It can be used to setup UI for Vault.

## How to set up

Setup helm. Git clone that repo. Then go to dir:

```
cd $VAULT_UI_REPO/kubernetes/chart/vault-ui
```

Change values.yaml to:

```
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
  enabled: false
resources: {}
vault:
  auth: TOKEN
  url: https://<service-addr>
```

Replace `<service-addr>` above. For example, `example-vault.vault-services.svc:8200`.


Add env to `templates/deployment.yaml`:

```
- name: NODE_TLS_REJECT_UNAUTHORIZED
  value: "0"
```

Install helm chart:

```
helm install .
```

There should be a deployment/service called `dandy-aardwolf-vault-ui`.
