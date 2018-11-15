# Example script to create a new pivnet release.


### NOTE
If you are using [concourse](https://concourse-ci.org/), use the [pivnet-resource](https://github.com/pivotal-cf/pivnet-resource) to create pivnet releases.


### About
This is an example script that automates the process of creating new releases on [Pivotal Network](https://network.pivotal.io). It will create a release, upload product files, and associate those files with the release.
**This script should be modified to meet your needs. For instance, the create release API endpoint has additional options for metadata that you may need.** See the [PivNet API guide](https://network.pivotal.io/docs/api) for additional details.

### Usage

1. Create a directory and put all product release files in it
2. Modified the values at the top of the script
3. Run `./pivnet-create-release.sh`

### Dependencies

The script has the following dependencies:

- [curl](https://github.com/curl/curl)
- [aws cli](https://aws.amazon.com/cli/)
- [jq](https://stedolan.github.io/jq/)
- [openssl](https://github.com/openssl/openssl) (can be easily modified to use another tool that generates sha256 digest)


