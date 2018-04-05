#!/usr/bin/env bash
set -ex

TMPDIR="tmp"
mkdir -p $TMPDIR

echo "Getting docker-bosh in place"

docker_bosh_pkg_path=./${TMPDIR}/docker.tar.gz
if [ -f "${docker_bosh_pkg_path}" ]; then
    echo "Docker-bosh package already exist, skipping download"
else
    echo "Docker-bosh package doesn't exist, downloading"
    url=https://github.com/cloudfoundry-incubator/docker-boshrelease/releases/download/v31.0.1/docker-31.0.1.tgz
    wget ${url} -O "${docker_bosh_pkg_path}"
fi

cf_cli_pkg_path=./${TMPDIR}/cf-cli.tgz
if [ -f "${cf_cli_pkg_path}" ]; then
    echo "cf cli package already exist, skipping download"
else
    echo "cf cli package doesn't exist, downloading"
    url='https://packages.cloudfoundry.org/stable?release=linux64-binary&version=6.36.0&source=github-rel'
    wget ${url} -O "${cf_cli_pkg_path}"
fi


bosh upload-release ${docker_bosh_pkg_path}


echo "Adding blobs"

bosh add-blob ./${TMPDIR}/kibosh.linux kibosh.linux
bosh add-blob ./${TMPDIR}/loader.linux loader.linux
bosh add-blob ${cf_cli_pkg_path} cf-cli.tgz
bosh add-blob ./${TMPDIR}/delete_all_and_deregister.linux delete_all_and_deregister.linux

bosh create-release --name=kibosh --force

bosh upload-release --name=kibosh

yes | bosh -d kibosh deploy manifests/lite-manifest.yml --no-redact --vars-store=manifests/values.yml

bosh -d kibosh run-errand load-image --keep-alive
bosh -d kibosh run-errand registrar --keep-alive
