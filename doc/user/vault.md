# Vault

This doc talks about how to access and use Vault after deployed by Vault operator.

## Access Vault

Continuing the example setup, we can use the following command to create a port-forwarding
between local machine and the Vault server running on Kubernetes:

```
kubectl get po -l app=vault,name=example-vault -o jsonpath='{.items[*].metadata.name}' | xargs -0 -I {} kubectl port-forward {} 8200
```

Open a new terminal. Use the following commands to check vault server's status:

```
export VAULT_ADDR='https://localhost:8200'
export VAULT_SKIP_VERIFY="true"
vault status
```

Seeing following messages means that Vault server is up and running:

```
URL: GET https://localhost:8200/v1/sys/seal-status
Code: 400. Errors:

* server is not yet initialized
```

Now you have access to Vault. Check out [how to initialize Vault](https://www.vaultproject.io/intro/getting-started/deploy.html#initializing-the-vault) .

