#!/usr/bin/env bash

kubectl --namespace=kube-system delete serviceaccount kibosh-admin
kubectl --namespace=kube-system delete clusterrolebindings kibosh-cluster-admin

kubectl create -f dev/minikube_rbac.yml
kubectl --namespace=kube-system get serviceaccount kibosh-admin -o jsonpath=secrets

secret_name=$(kubectl get serviceaccount kibosh-admin --namespace=kube-system -o jsonpath='{.secrets[0].name}')
secret_val=$(kubectl --namespace=kube-system get secret $secret_name -o jsonpath='{.data.token}')
echo ""
echo "Token"
echo $secret_val | base64 -D
echo ""
