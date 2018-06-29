#!/usr/bin/env bash
set -ex

source create_release.sh

yes | bosh -d kibosh deploy manifests/lite-manifest.yml --no-redact --vars-store=manifests/values.yml

bosh -d kibosh run-errand registrar --keep-alive
