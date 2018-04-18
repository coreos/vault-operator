# Developer guide

This document explains how to setup your dev environment.

## Vendor dependencies

We use [dep](https://github.com/golang/dep) to manage dependencies.
Run the following in the project root directory to update the vendored dependencies:

```sh
$ hack/update_vendor.sh
```

## Build the container image

Requirement:
- Go 1.9+

Build the vault operator binary:

```sh
$ hack/build
```

Build and push the container image to a specified repository e.g `quay.io/coreos/vault-operator:dev`:

```sh
$ IMAGE=<image-name> hack/push
```
