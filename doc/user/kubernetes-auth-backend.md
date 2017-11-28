# Using the Kubernetes auth backend

This guide shows a simple example of how to set up and authenticate against the Kubernetes auth backend. For more details, consult the Vault documentation on the [Kubernetes Auth Backend][kubernetes-auth-backend].

This example will:
- Set up the Kubernetes Auth Backend
- Configure a Role for a service account with some policy
- Authenticate Vault requests using the service account token

### Prerequisites:
- Tectonic 1.7+

## Kubernetes Auth Backend Setup

Initialize and unseal a Vault cluster. The [create-cluster.sh][create-cluster] script can be used to do this.

This example assumes a Vault cluster running in the namespace `vault-services`. Adjust the commands below as needed for your namespace.

### Configure port forwarding

To enable and configure the auth backend with the necessary roles and policies, make the Vault client requests authenticate with the root token.

1. Configure port forwarding between the local machine and the active Vault node:

    ```sh
    $ kubectl -n vault-services get vault example-vault -o jsonpath='{.status.nodes.active}' | xargs -0 -I {} kubectl -n vault-services port-forward {} 8200
    ```

2. Open a new terminal. Use this terminal for the rest of this guide.

3. Export the following environment for the [Vault CLI environment][vault-cli-env].
    Replace `<root-token>` with the root token generated during initialization.

    ```sh
    export VAULT_ADDR='https://localhost:8200'
    export VAULT_SKIP_VERIFY="true"
    export VAULT_TOKEN=<root-token>
    ```

### Enable and configure the backend

1. Enable the Kubernetes auth backend:

    ```sh
    $ vault auth-enable kubernetes

    ```
2. Configure the backend with the Kubernetes master server URL and certificate-authority-data.

    ```sh
    $ vault write auth/kubernetes/config kubernetes_host=<server-url> kubernetes_ca_cert=@ca.crt
    ```

### Create a policy and role

The Kubernetes backend authorizes an entity by granting it a role mapped to a service account. A role is configured with policies which control the entity's access to paths and operations in Vault.

1. Create a policy file `demo-policy.hcl` as shown:

```sh
$ cat << EOF > demo-policy.hcl
path "secret/demo/*" {
    capabilities = ["create", "read", "update", "delete", "list"]
}
EOF
```

2. Create a new policy `demo-policy` using the policy file `demo-policy.hcl`.

    ```sh
    $ vault write sys/policy/demo-policy rules=@demo-policy.hcl
    ```

3. Create a new role `demo-role` configured for the service account `demo` and policy `demo-policy`:

    ```sh
    $ vault write auth/kubernetes/role/demo-role \
        bound_service_account_names=demo \
        bound_service_account_namespaces=vault-services \
        policies=demo-policy \
        ttl=1h
    ```

## Authenticate requests using the service account token

The backend can now be used to authenticate Vault requests using the service account `demo`.

### Set up service account and RBAC

1. Create the service account `demo`:

```sh
$ kubectl -n vault-services create serviceaccount demo
```

2. Create the ClusterRoleBinding for the `demo` service account so that it can be used to access the Token Review API:

```sh
$ cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: demo-tokenreview-binding
  namespace: vault-services
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: demo
  namespace: vault-services
EOF
```

### Authenticate

Use the service account token to authenticate for the role `demo-role`.

1. Fetch the service account token for the `demo` service account:

    ```sh
    $ SECRET_NAME=$(kubectl -n vault-services get serviceaccount demo -o jsonpath='{.secrets[0].name}')
    $ DEMO_TOKEN=$(kubectl -n vault-services get secret ${SECRET_NAME} -o jsonpath='{.data.token}' | base64 --decode)
    ```

2. Log in to the Kubernetes auth backend using the service account token:

    ```sh
    $ vault write auth/kubernetes/login role=demo-role jwt=${DEMO_TOKEN}
    Key                                   	Value
    ---                                   	-----
    token                                 	74603479-607d-4ab8-a406-d0456d9f3d65
    token_accessor                        	4893b0a1-f42a-bfd8-cd9c-c14b9bdb6095
    token_duration                        	1h0m0s
    token_renewable                       	true
    token_policies                        	[default demo-policy]
    token_meta_role                       	"demo-role"
    token_meta_service_account_name       	"demo"
    token_meta_service_account_namespace  	"vault-services"
    token_meta_service_account_secret_name	"demo-token-fndln"
    token_meta_service_account_uid        	"aaf6c23c-b04a-11e7-9aea-0245c85cf1cc"
    ```

3. Set the `VAULT_TOKEN` to the value of the key `token` from the output of the last step:

    ```sh
    $ export VAULT_TOKEN=<token>
    ```

### Issue requests

With the above `VAULT_TOKEN` set, Vault requests should be authenticated according to the role `demo-role` and the policy `demo-policy`.

To test the policy, confirm that secrets may be created only under the path `secret/demo/`, and nowhere else:

```sh
$ vault write secret/demo/foo value=bar
Success! Data written to: secret/demo/foo
```

```sh
$ vault write secret/foo value=bar
Error writing data to secret/foo: Error making API request.

URL: PUT https://localhost:8200/v1/secret/foo
Code: 403. Errors:

* permission denied
```

### Cleanup

```sh
$ kubectl -n vault-services delete serviceaccount demo
$ kubectl -n vault-services delete clusterrolebinding demo-tokenreview-binding
```


[kubernetes-auth-backend]: https://www.vaultproject.io/docs/auth/kubernetes.html
[vault-cli-env]: https://www.vaultproject.io/docs/commands/environment.html
[create-cluster]: https://github.com/coreos-inc/vault-operator/tree/master/hack/helper/create-cluster.sh
