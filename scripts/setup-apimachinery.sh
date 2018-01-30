#!/usr/bin/env bash

set -ex

ls -lah ./vendor/k8s.io

#rm -rf ./vendor/k8s.io/{api,apimachinery,apiserver,client-go,metrics}
#cp -r ./vendor/k8s.io/kubernetes/staging/src/k8s.io/{api,apimachinery,apiserver,client-go,metrics} ./vendor/k8s.io
