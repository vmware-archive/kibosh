#!/usr/bin/env bash
set -ex

yes | bosh -d kibosh delete-deployment
yes | bosh delete-release kibosh

rm -rf ./.dev_builds/
rm -rf ./blobs/
rm -rf ./dev_releases/
#rm ./tmp/kibosh_src.tgz
rm -rf ./tmp/
