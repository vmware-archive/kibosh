#!/usr/bin/env bash
set -ex

yes | bosh -d helm_chart delete-deployment
yes | bosh delete-release helm_chart

rm -rf ./.dev_builds/
rm -rf ./blobs/
rm -rf ./dev_releases/
rm -rf ./tmp/
