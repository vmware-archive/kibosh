#!/usr/bin/env bash
set -ex

echo "Generate CA key and cert"

cat > ca.conf <<EOF
[ req ]
distinguished_name = req_distinguished_name
req_extensions = v3_req
[ req_distinguished_name ]
countryName_default = US
stateOrProvinceName_default = WA
localityName_default = Seattle
organizationalUnitName_default = Pivotal
[ v3_ca ]
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid:always,issuer:always
basicConstraints = CA:true
[ v3_req ]
EOF

openssl genrsa -out ./ca.key.pem 4096
openssl req -key ca.key.pem -new -x509 -days 7300  -subj "/CN=*.kibosh.cf/O=Pivotal/C=US" -sha256 -out ca.cert.pem -extensions v3_ca -config ca.conf

rm ca.conf


echo "Generating tiller key and signed cert"

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
openssl req -key tiller.key.pem -new -sha256 -subj "/CN=tiller.kibosh.cf/O=Pivotal/C=US" -out tiller.csr.pem -config tiller.conf
openssl x509 -req -CA ca.cert.pem -CAkey ca.key.pem -CAcreateserial -in tiller.csr.pem -out tiller.cert.pem -days 7300

rm tiller.csr.pem
rm tiller.conf

echo "Generating helm key and signed cert"

cat > helm.conf <<EOF
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

openssl genrsa -out ./helm.key.pem 4096
openssl req -key helm.key.pem -new -sha256 -subj "/CN=helm.kibosh.cf/O=Pivotal/C=US" -out helm.csr.pem -config helm.conf
openssl x509 -req -CA ca.cert.pem -CAkey ca.key.pem -CAcreateserial -in helm.csr.pem -out helm.cert.pem -days 7300

rm helm.csr.pem
rm helm.conf

rm ca.srl
