#!/usr/bin/env bash


kubectl --namespace=kube-system delete serviceaccount kibosh-admin
kubectl --namespace=kube-system delete clusterrolebindings kibosh-cluster-admin

kubectl create -f docs/dev/minikube_rbac.yml
kubectl --namespace=kube-system get serviceaccount kibosh-admin -o jsonpath=secrets

ca_file=$(kubectl config view -o jsonpath='{.clusters[?(@.name == "minikube")].cluster.certificate-authority}')
export CA_DATA=$(cat $ca_file)

export SERVER=$(kubectl config view -o jsonpath='{.clusters[?(@.name == "minikube")].cluster.server}')

secret_name=$(kubectl get serviceaccount kibosh-admin --namespace=kube-system -o jsonpath='{.secrets[0].name}')
secret_val=$(kubectl --namespace=kube-system get secret $secret_name -o jsonpath='{.data.token}')

export TOKEN=$(echo $secret_val | base64 -D)

export SECURITY_USER_NAME=admin
export SECURITY_USER_PASSWORD=pass
export TILLER_NAMESPACE=kibosh

export CF_BROKER_URL="http://192.168.0.4:8080"
export CF_BROKER_NAME="bazaar"
export CF_API_ADDRESS="https://api.v3.pcfdev.io"
export CF_USERNAME="admin"
export CF_PASSWORD="admin"
export CF_SKIP_SSL_VALIDATION="true"

# REG_* settings are optional, for configuring a private docker registry
# export REG_SERVER='gcr.io'
# export REG_USER='_json_key'
# export REG_PASS='{
#   "type": "service_account",
#   ...
# }'

#export REG_EMAIL='_json_key'

go run cmd/kibosh/main.go
