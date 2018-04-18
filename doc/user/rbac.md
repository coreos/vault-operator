# RBAC guide

The Vault operator works in conjunction with the etcd operator to setup a Vault cluster. To do this both the etcd and Vault operators need RBAC permissions to access the necessary resources.

This guide shows an example of how to setup a Role and RoleBinding for the etcd and vault operators. The provided [RBAC template][rbac-template] contains the collective rules for both the etcd and Vault operator.

For an overview of the resources created by the vault operator see the [resource labels and ownership][resources-doc] doc

## Create a Role and RoleBinding

This example binds a Role to the `default` service account in the `default` namespace.

**Note:** For production usage you should create a specific service account to bind the Role to.

1. Generate the RBAC manifest from the template given in the repository by setting the namesapce and service account:

    ```sh
    $ sed -e 's/<namespace>/default/g' \
        -e 's/<service-account>/default/g' \
        example/rbac-template.yaml > example/rbac.yaml
    ```

2. Create the Role and RoleBinding from the RBAC manifest:

    ```sh
    kubectl -n default create -f example/rbac.yaml
    ```



[rbac-template]: ../../example/rbac-template.yaml
[resources-doc]: ./resource_labels_and_ownership.md
