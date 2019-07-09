#!/usr/bin/env bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
pushd ${DIR}/../integration-image

echo "Uploading dependent releases"
bosh_cli_path=./tmp/bosh
if [[ -f "${bosh_cli_path}" ]]; then
    echo "bosh-cli package already exist, skipping download"
else
    echo "bosh-cli package doesn't exist, downloading"
    url=https://github.com/cloudfoundry/bosh-cli/releases/download/v5.5.1/bosh-cli-5.5.1-linux-amd64
    wget ${url} -O "${bosh_cli_path}"
fi

docker build . -t cfplatformeng/kibosh-integration-image

echo "Be sure to push the image"