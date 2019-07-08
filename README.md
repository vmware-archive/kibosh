# Kibosh

An [open service broker](https://github.com/openservicebrokerapi/servicebroker)
bridging the gap between Kubernetes deployments and CF marketplace.

When deployed with a Helm chart and added to the marketplace,
* `cf create-service` calls to Kibosh will create the collection of Kubernetes resources described by the chart.
* `cf bind-service` calls to Kibosh will expose back any services and secrets created by the chart

### Overriding/Setting values as defined in values.yaml via 'cf create-service' or 'cf update-service'.  

The format of the json string is a nested format.  
Also refer to the cf cli for an example of a valid JSON object.

**Example:** Setting the mysqlUser on `cf create-service` for the MySQL chart

values.yaml

```## Create a database user
##
# mysqlUser:
# mysqlPassword:
```

`cf create-service mysql medium mysql-kibosh-service -c '{"mysqlUser":"admin"}'`

**Example:** Setting the resources.requests.memory on `cf update-service` for the MySQL chart

values.yaml

```## Configure resource requests and limits
## ref: http://kubernetes.io/docs/user-guide/compute-resources/
##
resources:
  requests:
    memory: 256Mi
    cpu: 100m
```

`cf update-service mysql-kibosh-service  -c '{"resources": {"requests": {"memory": "256Mi"}}}'`

For some in depth discussion, see this blog post:
[Use Kubernetes Helm Packages to Build Pivotal Cloud Foundry tiles](https://content.pivotal.io/blog/use-kubernetes-helm-packages-to-build-pivotal-cloud-foundry-tiles-kibosh-a-new-service-broker-makes-it-simple)

![](docs/kibosh_logo_100.png)

## Configuration
### Changes required in Chart
#### Plans (`cf marketplace`)  
Kibosh requires that helm chart has additional file that describes plan in `plans.yaml` at root level

```yaml
- name: "small"
description: "default (small) plan for mysql"
file: "small.yaml"
- name: "medium"
description: "medium sized plan for mysql"
file: "medium.yaml"
```

* `file` is a filename that exists in the `plans` subdirectory of the chart.
* File names should consist of only lowercase letters, digits, `.`, or `-`.
* The standard `values.yaml` file in the helm chart sets the defaults.
* Each plan's yaml file is a set of values overriding the defaults present in `values.yaml`.  

Copy any key/value pairs to override from `values.yaml` into a new plan file and change their value.  
See kibosh-sample's [sample-charts](https://github.com/cf-platform-eng/kibosh-sample/tree/master/sample-charts) for a few examples.

In order to successfully pull private images, we're imposing some requirements
on the `values.yaml` file structure

* **Single-image** charts should use this structure:
    ```yaml
    ---
    image: "my-image"
    imageTag: "5.7.14"
    ```
* **Multi-image** charts shoud use this structure:
    ```yaml
    ---
    images:
      thing1:
        image: "my-first-image"
        imageTag: "5.7.14"
      thing2:
        image: "my-second-image"
        imageTag: "1.2.3"
    ```

### Plan-Specific Clusters
_This feature is experimental and the syntax will likely change in the future_

By default, Kibosh will create all deployments in the same cluster. It's also possible for each plan to 
target a different cluster. In `plans.yaml`, the plan specifies a credentials file:
```yaml
---
- name: "small"
  description: "default (small) plan for mysql"
  file: "small.yaml"
  credentials: "small-creds.yaml"
```
    
The contents of this file mirror what would appear in the `.kube/config` file. For example, `small-creds.yaml`
would contain:

```yaml
---
apiVersion: v1
clusters:
  - cluster:
      certificate-authority-data: bXktY2VydA==
      server: https://pks.example.com
    name: my-cluster
contexts:
  - context:
      cluster: my-cluster
      user: my-user
    name: my-cluster
current-context: my-cluster
kind: Config
preferences: {}
users:
  - name: my-user
    user:
      token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

### Bind Templates

Developers and libraries often have specific assumptions around how the bind
environment variable should be structured. For example,
[Spring Cloud Connectors](https://cloud.spring.io/spring-cloud-connectors/) will
automatically support a service if the 
[right structure is present](https://cloud.spring.io/spring-cloud-connectors/spring-cloud-cloud-foundry-connector.html#_mysql).

Chart authors can transform what the Kibosh broker returns by writing a
[Jsonnet](https://jsonnet.org) template and putting it in the `bind.yaml` file
in the root of the Chart. For example, the `bind.yaml` following will transform the bind
response into a readily consumable structure:

```yaml
template: |
  {
    hostname: $.services[0].status.loadBalancer.ingress[0].ip,
    name: $.services[0].name,
    jdbcUrl: "jdbc:mysql://" + self.hostname + "/my_db?user=" + self.username + "&password=" + self.password + "&useSSL=false",
    uri: "mysql://" + self.username + ":" + self.password + "@" + self.hostname + ":" + self.port + "/my_db?reconnect=true",
    password: $.secrets[0].data['mysql-root-password'],
    port: 3306,
    username: "root"
  }
```

The template is executed in an environment has top-level `services` and `secrets`,
which are json marshalled versions of the services and secrets in the namespace
generated for the service. 

To test your bind template, use the template-tester binary from the [github release.](https://github.com/cf-platform-eng/kibosh/releases/latest)
It takes the namespace in which you have already deployed your helm chart and the file that has the Jsonnet template descrited above.

```bash
template-tester mynamespaceid bind.yaml
```

### CredHub Integration
Kibosh can be configured to store binding credentials in [CredHub](https://docs.cloudfoundry.org/credhub/).

The CF runtime CredHub does not expose an external url, so testing with CredHub can be done by
* Running a proxy app on the platform
* Pushing Kibosh as an app
* Running Kibosh in a BOSH release

To run a proxy, `cf push` the app located in [docs/credhub_proxy](docs/credhub_proxy). The proxy
will then be available at `https://credhub-proxy.<cf apps domain>`. 

Then add the following set of environment variables to configure the Kibosh process:
```bash
CH_CRED_HUB_URL: https://credhub-proxy.[apps-domain]
CH_UAA_URL: https://uaa.[system-domain]
CH_UAA_CLIENT_NAME: my-uaa-client
CH_UAA_CLIENT_SECRET: my-uaa-secret
CH_SKIP_SSL_VALIDATION: true
```

#### Setting up the Client
Firstly, the client needs to created and given the correct scope via [uaac](https://github.com/cloudfoundry/cf-uaac)
```bash
uaac target uaa.[system-domain] --skip-ssl-validation
uaac token client get admin

uaac client add my-uaa-client \
    --access_token_validity 1209600 \
    --authorized_grant_types client_credentials,refresh_token \
    -s my-uaa-secret \
    --scope openid,oauth.approvals,credhub.read,credhub.write \
    --authorities oauth.login,credhub.read,credhub.write
```

Secondly, get a token for the client "Credhub Admin Client Client Credentials" in Ops Manager.

```bash
uaac token client get credhub_admin_client
# get the token by viewing the context
uaac context
```

Finally, give our client access to modify creds in CredHub. This can be done via curl.
```
curl -k "https://credhub-proxy.[apps-domain]/api/v2/permissions" \
  -X POST \
  -d '{
     "path": "/c/kibosh/*",
     "actor": "uaa-client:my-uaa-client",
     "operations": ["read", "write", "delete", "read_acl", "write_acl"]
  }' \
  -H "authorization: bearer [TOKEN]" \
  -H 'content-type: application/json'
```

### Other Requirements

* When defining a `Service`, to expose this back to any applications that are bound,
  `type: LoadBalancer` is a current requirement.
   `NodePort` is also an option and Kibosh will add externalIPs and nodePort to bind json, but `NodePort` does carry significant risks and probably should **not** be used in production: is not robust to cluster scaling events, upgrades or other IP changes.
* Resizing disks has limitiations. To support upgrade:
    - You can't resize a persistent volume claim (currently behind an [alpha feature gate](https://kubernetes.io/docs/reference/feature-gates/))
* Selectors are [immutable](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#selector)
    - This means that *chart name cannot change* (the name is generally used in selectors)

### Private registries
When the environment settings for a private registry are present (`REG_SERVER`, `REG_USER`, `REG_PASS`), 
then Kibosh will transform images to pull them from the private registry. It assumes
the image is already present (see the Kibosh deployment). It will patch
the default service account in the instance namespaces to add in the registry credentials.

Be sure that `REG_SERVER` contains any required path information. For example, in gcp `gcr.io/my-project-name`

## Contributing to Kibosh

We welcome comments, questions, and contributions from community members. Please consider
the following ways to contribute:

* File Github issues for questions, bugs and new features and comment and vote on the ones that you are interested in.
* If you want to contribute code, please make your code changes on a fork of this repository and submit a
pull request to the master branch of Kibosh. We strongly suggest that you first file an issue to
let us know of your intent, or comment on the issue you are planning to address.

## Deploying
To manually deploy the BOSH release, get the latest BOSH release (`kibosh-release-X.X.XX.tgz`)
from the  [Github releases](https://github.com/cf-platform-eng/kibosh/releases) and upload
to your director.

Build a manifest by starting from the example bosh-lite manifest 
[lite-manifest.yml](bosh/bosh-release/manifests/lite-bazaar-manifest.yml)
and customize the cloud specific settings (`az`, `vm_type`, etc). This manifest
uses a set of [input variables](https://bosh.io/docs/cli-int/).
See
[values-sample.yml](bosh/bosh-release/manifests/values-sample.yml)
for example values.

## Dev
### Setup

Install Go dependencies
```bash
go get -u github.com/onsi/ginkgo/ginkgo
go get -u github.com/onsi/gomega
go get -u github.com/maxbrunsfeld/counterfeiter
go get -u github.com/golang/dep/cmd/dep
go get -u github.com/gosuri/uitable
```

### Run
Run `make bootstrap` from a clean checkout to setup initial dependencies. This will restore
the locked dependency set specified by `Gopkg.toml` (we're no longer checking in `vendor`).

#### For a remote K8s Cluster
Copy `local_dev.sh.template` to `local_dev.sh` (which is in `.gitignore`) and 
configure the values (`cluster.certificate-authority-data`, `cluster.server`, and `user.token`)
for a working cluster. Then run:

```bash
./local_dev.sh
```

#### For minikube
Make sure minikube is running:

```bash
minikube start --vm-driver=hyperkit
```

Use `local_dev_minikube` to set up all the secrets and start kibosh:

```bash
local_dev_minikube.sh
```

### Securing tiller
In production, tiller **should be secured**. It's probably good practice to use secure tiller
in your local environment as well (at least some of the time) to catch issues.

To generate a set of credentials, run [tiller_ssl.sh](docs/tiller-ssl/tiller_ssl.sh) from inside
`docs/tiller-ssl/`. This will create a CA cert, a cert/key pair for Tiller, and a client cert/key pair.
If debugging using the helm cli, include the tls flags. For example:
```bash
helm ls --all --tls-verify --tls-ca-cert docs/tiller-ssl/ca.cert.pem --tls-cert docs/tiller-ssl/tiller.cert.pem --tls-key docs/tiller-ssl/tiller.key.pem
```

See [Helm's tiller_ssl.md](https://github.com/helm/helm/blob/master/docs/tiller_ssl.md) for more details. 

### Charts
The Kibosh code loads charts from the `HELM_CHART_DIR`, which defaults to `charts`.
This directory can either be a single chart (with all the changes described in the
configuration, eg `plans.yaml` and `./plans`), or, directory where each
subdirectory is a chart. The multiple charts feature isn't yet supported by tile-generator.
```
charts
├── mariadb
│   ├── Chart.yaml
    ├── plans
    │   ├── medium.yaml
    │   └── small.yaml
    ├── plans.yaml
    ├── templates
...
└── mysql
    ├── Chart.yaml
    ├── plans
    │   └── default.yaml
...
```

We have modified [some example charts](https://github.com/cf-platform-eng/kibosh-sample/tree/master/sample-charts) from stable helm repository.
 
### Test
```bash
make test
```

To generate the test-doubles, after any interface change run: 
```bash
make generate
```

For manual testing, there is a [Python test harness](docs/dev-testing).

## CI
* https://concourse.cfplatformeng.com/teams/main/pipelines/kibosh

The pipeline is backed by a cluster in the shared GKE account. The default admin
user in GKE has a password while Kibosh is configured to use a token. To create a user
in the cluster and fetch the token, do something like:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kibosh-concourse-ci
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: kibosh-concourse-ci
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: kibosh-concourse-ci
    namespace: kube-system
```

```bash
kubectl create -f [above contents in file].yml
kubectl get secrets --namespace=kube-system | grep "kibosh-concourse-ci"
kubectl get secret --namespace=kube-system kibosh-concourse-ci-token-pfnqs -o yaml
```

### Dependency vendoring

To add a dependency:
```bash
dep ensure -add github.com/pkg/errors
```

To update a dependency:
```bash
dep ensure -update github.com/pkg/errors
```

Dependency vendoring with respect to helm & k8s is trickier. `dep` isn't able to build the
tree without significant help. The `Gopkg.tml` has several overrides needed to get everything
to compile.

Updating to a new version of helm/k8s will probably require re-visiting the override & constraint
matrix built. Useful inputs into this process are:
* The k8s Godeps
    - https://github.com/kubernetes/kubernetes/blob/master/Godeps/Godeps.json
* Helm's Glide dependencies and dependency lock file
    - https://github.com/kubernetes/helm/blob/master/glide.yaml
    - https://github.com/kubernetes/helm/blob/master/glide.lock
* Draft's `Gopkg.toml` file (they're doing the same thing we are, pulling in Helm as a library)
    - https://github.com/cf-platform-eng/kibosh
* This [helm tracker issue](https://github.com/kubernetes/helm/issues/3031) also has some useful context

Also run the make target `cleandep` to wipe out the lock file an any local state when upgrading
helm/k8s, to make sure it can be rebuilt cleanly from the specified constraints.

More dep links:
* Common dep commands: https://golang.github.io/dep/docs/daily-dep.html
* `Gopks.toml` details: https://github.com/golang/dep/blob/master/docs/Gopkg.toml.md

## Bazaar
Kibosh can also manage multiple charts more dynamically (without redeployment).
This allows customers to add any available helm chart to their cf marketplace with 
minimal effort and cycle time.

There is also a corresponding cli (`bazaarcli`) to manage these charts.

```bash
./bazaarcli.mac -t http://bazaar.v3.pcfdev.io -u admin -p 'monkey123' list
./bazaarcli.mac -t http://bazaar.v3.pcfdev.io -u admin -p 'monkey123' save ~/workspace/kibosh-sample/sample-charts/mysql-0.8.2.tgz
./bazaarcli.mac -t http://bazaar.v3.pcfdev.io -u admin -p 'monkey123' save ~/workspace/kibosh-sample/sample-charts/rabbitmq-1.1.9.tgz
./bazaarcli.mac -t http://bazaar.v3.pcfdev.io -u admin -p 'monkey123' list

cf enable-service-access mysql
cf enable-service-access rabbitmq
cf marketplace

./bazaarcli.mac -t http://bazaar.v3.pcfdev.io -u admin -p 'monkey123' delete rabbitmq
cf marketplace
```

To deploy in this way, start from the example bosh-lite manifest 
[lite-bazaar-manifest.yml](bosh/bosh-release/manifests/lite-bazaar-manifest.yml)
and customize the cloud specific settings (`az`, `vm_type`, etc). See the deploying section
for more details.

Alternatively, you can also deploy Bazaar as a [tile](https://github.com/cf-platform-eng/kibosh-sample/tree/master/bazaar-tile).  Follow the README on that page.

## Notes

Inline-style: 
![](docs/SeqDiagram.png)

<details>
  <summary>Sequence diagram source</summary>
  via https://www.websequencediagrams.com/ 

        title Kibosh

        operator->cf: deploy tile with kibosh and helm chart
        kibosh->cf: add offering to marketplaces via errand
        user->cf: cf create-service
        cf->kibosh: OSBAPI api provision call
        kibosh-> k8s: deploy chart
        user->cf: cf bind-service
        cf->kibosh: OSBAPI api bind call
        kibosh-> k8s: k8s api to get secrets & services
        k8s->kibosh: secrets and services
        kibosh->cf: secrets and services as credentials json
        cf->app: secrets and services as env vars
</details>

###  MVP architecture
MVP architecture, including Kibosh packaged by
[tile-generator](https://github.com/cf-platform-eng/tile-generator/)

![](docs/mvp_architecture.jpg)

