#!/usr/bin/env bash
set -ex

source create_release.sh

yes | bosh -d bazaar deploy manifests/lite-bazaar-manifest.yml --no-redact --vars-store=manifests/values.yml
