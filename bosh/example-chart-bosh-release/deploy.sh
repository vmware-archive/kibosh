#!/usr/bin/env bash

set -e

# Tar up the helm chart

TMPDIR="tmp"
mkdir -p $TMPDIR

# ci can't do this. Refactor script to optionally do it sometimes?
# docker pull cfplatformeng/spacebears:latest
# docker save cfplatformeng/spacebears:latest -o ../example-chart/images/spacebears.latest.tgz
# docker save gcr.io/kubernetes-helm/tiller:v2.8.2 -o ../example-chart/images/tiller.2.8.2.tgz

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
tar -cvzf ./${TMPDIR}/helm_chart_src.tgz -C "${DIR}/../../docs" example-chart

# Add it as a blob in the bosh release
bosh add-blob ./${TMPDIR}/helm_chart_src.tgz helm_chart_src.tgz

bosh create-release --name=example-chart --force

bosh upload-release --name=example-chart
