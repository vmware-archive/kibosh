#!/usr/bin/env bash
set -ex

openssl genrsa -out ./ca.key.pem 4096

cat > ca.conf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
[req_distinguished_name]
countryName_default = US
stateOrProvinceName_default = WA
localityName_default = Seattle
organizationalUnitName_default = Pivotal
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
EOF

openssl req -new -x509 -days 7300 -sha256 -out ca.csr -subj "/CN=*.kibosh.cf/O=Pivotal/C=US" -key ca.key.pem -config ca.conf
#openssl req -key ca.key.pem -new -x509 -days 7300 -sha256 -out ca.cert.pem -extensions v3_ca


cat > tiller.conf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
[req_distinguished_name]
countryName_default = US
stateOrProvinceName_default = WA
localityName_default = Seattle
organizationalUnitName_default = Pivotal
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
EOF

openssl genrsa -out ./tiller.key.pem 4096
openssl req -key tiller.key.pem -new -sha256  -subj "/CN=tiller..kibosh.cf/O=Pivotal/C=US" -out tiller.csr.pem -config ca.conf


cat > helm_cli.conf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
[req_distinguished_name]
countryName_default = US
stateOrProvinceName_default = WA
localityName_default = Seattle
organizationalUnitName_default = Pivotal
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
EOF

openssl genrsa -out ./helm_cli.key.pem 4096
openssl req -key helm_cli.key.pem -new -sha256  -subj "/CN=helm.kibosh.cf/O=Pivotal/C=US" -out helm_cli.csr.pem -config ca.conf
