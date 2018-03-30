#!/usr/bin/env bash
set -ex

TMPDIR="tmp"
mkdir -p $TMPDIR

echo "Adding blobs"

bosh add-blob ./${TMPDIR}/kibosh.linux kibosh.linux
bosh add-blob ./${TMPDIR}/loader.linux loader.linux

bosh create-release --name=kibosh --force

bosh upload-release --name=kibosh

yes | bosh -d kibosh deploy manifests/lite-manifest.yml --no-redact --vars-store=manifests/values.yml

bosh -d kibosh run-errand load-image --keep-alive
