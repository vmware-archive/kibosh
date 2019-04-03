#!/bin/bash

# $1 is cluster name

# kubectl doesn't properly export the ca data below, so hard coding here

#CA_DATA_RAW=$(kubectl config view -o jsonpath='{.clusters[?(@.name == "'$1'")].cluster.certificate-authority-data}')
CA_DATA_RAW=
export SERVER=$(kubectl config view -o jsonpath='{.clusters[?(@.name == "'$1'")].cluster.server}')

USER_ID=$(kubectl config view -o jsonpath='{.contexts[?(@.context.cluster == "'$1'")].context.user}')
 
secret_val=$(kubectl config view -o jsonpath='{.users[?(@.name == "'$USER_ID'")].user.token}')

export TOKEN=$secret_val

export CA_DATA=$(echo $CA_DATA_RAW | base64 -D)

export SECURITY_USER_NAME=admin
export SECURITY_USER_PASSWORD=pass
export TILLER_NAMESPACE=kibosh

go run cmd/kibosh/main.go
