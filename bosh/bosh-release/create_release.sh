#!/usr/bin/env bash

set -e

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

bosh upload-release ${docker_bosh_pkg_path}

cf_cli_bosh_pkg_path=./${TMPDIR}/cf-cli.tar.gz
if [ -f "${cf_cli_bosh_pkg_path}" ]; then
    echo "cf-cli-bosh package already exist, skipping download"
else
    echo "cf-cli-bosh package doesn't exist, downloading"
    url=https://bosh.io/d/github.com/bosh-packages/cf-cli-release
    wget ${url} -O "${cf_cli_bosh_pkg_path}"
fi

bosh upload-release ${cf_cli_bosh_pkg_path}

cp ../../kibosh.linux ./${TMPDIR}
cp ../../loader.linux ./${TMPDIR}
cp ../../bazaar.linux ./${TMPDIR}
cp ../../bazaarcli.linux ./${TMPDIR}

delete_all_and_deregister_path=./${TMPDIR}/delete_all_and_deregister.linux
if [ -f "${delete_all_and_deregister_path}" ]; then
    echo "delete_all_and_deregister.linux already exists, skipping download"
else
    echo "delete_all_and_deregister.linux doesn't exists, downloading"
    url=https://storage.googleapis.com/kibosh-public/delete_all_and_deregister-v0.27.0.linux
    wget ${url} -O "${delete_all_and_deregister_path}"
fi

echo "Adding blobs"

bosh add-blob ./${TMPDIR}/kibosh.linux kibosh.linux
bosh add-blob ./${TMPDIR}/loader.linux loader.linux
bosh add-blob ./${TMPDIR}/bazaar.linux bazaar.linux
bosh add-blob ./${TMPDIR}/bazaarcli.linux bazaarcli.linux
bosh add-blob ./${TMPDIR}/delete_all_and_deregister.linux delete_all_and_deregister.linux

bosh create-release --name=kibosh --force

bosh upload-release --name=kibosh
