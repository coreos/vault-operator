#!/usr/bin/env bash

tar czvf vault-operator.tar.gz README.* doc/user/ example/ hack/tls-gen.sh

# Generating Zip file:
# In Mac, once we have above tarball, we can untar it by double clicking the tarball,
# and then right click -> "Compress" to make a Zip file.
