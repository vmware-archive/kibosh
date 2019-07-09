#!/usr/bin/env bash

set -ex

TMPDIR="tmp"
mkdir -p $TMPDIR

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
HELM_CHART_DIR=${1:-"${DIR}/../../docs"}
HELM_CHART_NAME=${2:-"example-chart"}

tar -cvzf ./${TMPDIR}/helm_chart_src.tgz -C ${HELM_CHART_DIR} ${HELM_CHART_NAME}

# Add it as a blob in the bosh release
bosh add-blob ./${TMPDIR}/helm_chart_src.tgz helm_chart_src.tgz

bosh create-release --name=example-chart --force

bosh upload-release --name=example-chart
