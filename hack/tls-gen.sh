#!/usr/bin/env bash 

: ${KUBE_NS:?"Need to set KUBE_NS"}

if [ -z "${SERVER_CERT}" ]; then
	SERVER_CERT="server.crt"
fi
if [ -z "${SERVER_KEY}" ]; then
	SERVER_KEY="server.key"
fi
# Create temporary output directory
OUTPUT_DIR=$(mktemp -d)

# Deletes the temp directory
function cleanup {      
  rm -rf "$OUTPUT_DIR"
}
trap cleanup EXIT


if ! which cfssl > /dev/null; then
	echo "cfssl needs to be installed"
	exit 1
fi

if ! which cfssljson > /dev/null; then
	echo "cfssljson needs to be installed"
	exit 1
fi

if ! which jq > /dev/null; then
	echo "jq needs to be installed"
	exit 1
fi

if ! kubectl version 1> /dev/null ; then
    echo "kubectl with kubeconfig needs to be setup"
	exit 1
fi

rm -rf $OUTPUT_DIR/config
mkdir -p $OUTPUT_DIR/config
rm -rf $OUTPUT_DIR/certs/tmp
mkdir -p $OUTPUT_DIR/certs/tmp

# Generate ca-config.json and ca-csr.json
cfssl print-defaults config | jq 'del(.signing.profiles) | .signing.default.expiry="8760h" | .signing.default.usages=["signing", "key encipherment", "server auth"]' > $OUTPUT_DIR/config/ca-config.json
cfssl print-defaults csr | jq 'del(.hosts) | .CN = "Autogenerated CA" | .names[0].O="Autogen CA for Vault-Operator"' > $OUTPUT_DIR/config/ca-csr.json

# add additional hosts to SANs
HOSTS="\"localhost\", \"*.${KUBE_NS}.pod\", \"*.${KUBE_NS}.svc\""
for i in $(echo ${SAN_HOSTS} | sed "s/,/ /g")
do
    HOSTS="\"$i\",${HOSTS}"
done
echo $HOSTS
# Generate vault-server-csr.json with the SANs according to the namespace and name of the vault cluster
cfssl print-defaults csr | jq ".hosts = [${HOSTS}] | .CN = \"vault-server\"" > $OUTPUT_DIR/config/vault-csr.json

# Generate ca cert and key
cfssl gencert -initca $OUTPUT_DIR/config/ca-csr.json | cfssljson -bare $OUTPUT_DIR/certs/tmp/ca

# Generate server cert and key
cfssl gencert \
    -ca $OUTPUT_DIR/certs/tmp/ca.pem \
    -ca-key $OUTPUT_DIR/certs/tmp/ca-key.pem \
    -config $OUTPUT_DIR/config/ca-config.json \
    $OUTPUT_DIR/config/vault-csr.json | cfssljson -bare $OUTPUT_DIR/certs/tmp/server

# Rename certs for vault-operator consumption
mv $OUTPUT_DIR/certs/tmp/ca.pem $OUTPUT_DIR/certs/vault-client-ca.crt
mv $OUTPUT_DIR/certs/tmp/server.pem $OUTPUT_DIR/certs/${SERVER_CERT}
mv $OUTPUT_DIR/certs/tmp/server-key.pem $OUTPUT_DIR/certs/${SERVER_KEY}

# Create server secret
echo ${SERVER_SECRET}
if [ -n "${SERVER_SECRET}" ]; then
	echo "creating secret"
	kubectl -n $KUBE_NS create secret generic $SERVER_SECRET --from-file=$OUTPUT_DIR/certs/${SERVER_CERT} --from-file=$OUTPUT_DIR/certs/${SERVER_KEY}
fi

# Create client secret
if [ -n "${CLIENT_SECRET}" ]; then
	kubectl -n $KUBE_NS create secret generic $CLIENT_SECRET --from-file=$OUTPUT_DIR/certs/vault-client-ca.crt
fi
