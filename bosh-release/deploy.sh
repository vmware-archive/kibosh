#!/usr/bin/env bash
set -ex

TMPDIR="tmp"
mkdir -p $TMPDIR

echo "This script automates the steps in the README"

go_pkg_remote=https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz
go_pkg_path=./${TMPDIR}/go-linux-amd64.tar.gz

if [ -f "${go_pkg_path}" ]; then
    echo "Go package already exist, skipping download"
else
    echo "Go package doesn't exist, downloading"
    wget "${go_pkg_remote}" -O "${go_pkg_path}"
fi
echo "${go_pkg_remote}" > ./${TMPDIR}/go-version.txt

echo "Packaging local source"

# Go up to the src directory above github.com in src/github.com/cf-platform-eng/kibosh in order
# to get the go path correct.
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
tar -cvzf ./${TMPDIR}/kibosh_src.tgz -C "${DIR}/../../../../" \
  github.com/cf-platform-eng/kibosh/broker/ \
  github.com/cf-platform-eng/kibosh/config/ \
  github.com/cf-platform-eng/kibosh/helm/ \
  github.com/cf-platform-eng/kibosh/k8s/ \
  github.com/cf-platform-eng/kibosh/vendor/ \
  github.com/cf-platform-eng/kibosh/main.go \
  github.com/cf-platform-eng/kibosh/Makefile

echo "Adding blobs"

bosh add-blob ./${TMPDIR}/go-linux-amd64.tar.gz go-linux-amd64.tar.gz
bosh add-blob ./${TMPDIR}/go-version.txt go-version.txt
bosh add-blob ./${TMPDIR}/kibosh_src.tgz kibosh_src.tgz
bosh add-blob ./${TMPDIR}/loader.linux loader.linux


bosh create-release --name=kibosh --force

bosh upload-release --name=kibosh

yes | bosh -d kibosh deploy manifests/lite-manifest.yml --no-redact --vars-store=manifests/values.yml

bosh -d kibosh run-errand load-image --keep-alive
