# kibosh/bosh-release

A BOSH release that deploys kibosh and pushes images to private registry.
This BOSH release is bundled into a tile by
[tile-generator](https://github.com/cf-platform-eng/tile-generator/)
`kibosh` package type. This is not intended to be consumed outside of tile-generator.

Jobs
* kibosh: OSBAPI broker
* load-image: errand to push docker images to a private registry

This release depends on a specific directory structure with respect to how
the helm chart and its associated docker images are laid out on disk. See
`example-chart` and `example-chart-bosh-release` for an example of this
structure.

`example-chart` is a standard helm chart with a small set of changes and constraints
* See root README.md for documentation around `values.yaml`, `plans.yaml`, and the `plans` subdirectory.
* `images` is a directory we introduce into the chart (used by the `load-image` errand) that contains locally exported docker image tgz files.
  Each image is
    - loaded into the local docker daemon
    - re-tagged for the configured private registry
    - pushed to that registry

This release also depends on 
[docker's bosh release](https://github.com/cloudfoundry-incubator/docker-boshrelease/)
to get docker daemon and cli (for pushing images to a private registry) 

In `manifests` there are examples pulling these three release together
into a working errand and broker deployment.

### build

`deploy.sh` will cycle locally using the `lite-manifest.yaml`. This manifest assumes
a default cloud-config.
