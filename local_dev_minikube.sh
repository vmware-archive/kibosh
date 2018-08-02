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

# REG_* settings are optional, for configuring a private docker registry
# export REG_SERVER='gcr.io'
# export REG_USER='_json_key'
# export REG_PASS='{
#   "type": "service_account",
#   ...
# }'

#export REG_EMAIL='_json_key'

LDFLAGS="-X github.com/cf-platform-eng/kibosh/pkg/helm.tillerTag=$(cat tiller-version)"

go run -ldflags "${LDFLAGS}" cmd/kibosh/main.go
