# Kibosh

A generic broker bridging the gap between Kubernetes and CF brokered services.

## Dev
#### Setup
Grab a newish version of Go (we used 1.9) 

Install Go depenencies
```bash
go get -u github.com/onsi/ginkgo/ginkgo
go get -u github.com/onsi/gomega
go get -u github.com/maxbrunsfeld/counterfeiter
go get -u github.com/golang/dep/cmd/dep
```

#### Minikube
To set things up in a way that authentication is done the same way as against PKS, run 
```bash
./dev/minikube_auth.sh
```

Which creates a service account with `cluster-admin` and output the token.

For `certificate-authority-data`, encode the minikube certificate:
```bash
cat ~/.minikube/ca.crt | base64
```

#### Dependency vendoring
Run `make bootstrap` from a clean checkout to setup initial dependencies. This will restore
the locked dependency set specified by `Gopkg.toml` (we're not longer checking in `vendor`).

To add a dependency:
```bash
dep ensure -add github.com/pkg/errors
```

To update a dependency:
```bash
dep ensure -update github.com/pkg/errors
```

Dependency vendoring wrt to helm & k8s is trickier. `dep` isn't able to build the
tree without significant help. The `Gopkg.tml` has several overrides needed to get everything
to compile (which work in conjunction with `setup-apimmachinery.sh`).

Updating to a new version of helm/k8s will probably require re-visiting the override & constraint
matrix built. Useful inputs into this process are:
* The k8s Godeps
    - https://github.com/kubernetes/kubernetes/blob/master/Godeps/Godeps.json
* Helm's Glide dependencies and dependency lock file
    - https://github.com/kubernetes/helm/blob/master/glide.yaml
    - https://github.com/kubernetes/helm/blob/master/glide.lock

Also run the make target `cleandep` to wipe out the lock file an any local state when upgrading
helm/k8s, to make sure it can be rebuilt cleanly from the specified constraints.

More dep links:
* Common dep commands:  https://golang.github.io/dep/docs/daily-dep.html
* `Gopks.toml` details: https://github.com/golang/dep/blob/master/docs/Gopkg.toml.md

#### Test
```bash
make test
```

Instructions to manually deploy and verify catalog: 
1) Create a directory and put the kibosh executable into it. 
    ```
    mkdir kibosh-example
    cp <somewhere>/kibosh.linux kibosh-example/kibosh.linux
    ```
1) Create a Manifest.yml file as shown below in that directory.
    ```
    cat > Manifest.yml
    ---
    applications:
    - name: kibosh_manual
      buildpack: binary_buildpack
      command: ./kibosh.linux
      services:
      - kibosh_cups
      env:
        SECURITY_USER_NAME: username
        SECURITY_USER_PASSWORD: password
        SERVICE_ID: f448fdea-25b8-41aa-aed4-539d8ace5e32
        HELM_CHART_DIR: charts
    ```
    The username/password come from XXXX...

1) Create a helm chart in that directory.  
   That helm chart could like like the example-chart in this project, 
   or could be one of these: https://github.com/kubernetes/charts.  The HELM_CHART_DIR
   environment variable in the Manifest.yml file should point to the sub-directory 
   in which you put it. 
   
1) Push the kibosh application (with its helm chart subdirectory) into CF
   ```
   cf push

   ```
1) Validate the data that the catalog endpoint provides
    ```
    curl -k https://example.com/v2/catalog -u 'username:password'
    ```

1) Put the application into the CF Marketplace. 
    ```
    cf create-service-broker kibosh-broker username password https://example.com
    cf enable-service-access spacebears
    cf service-brokers
    cf service-access
    ```

1) Create an instance of the helm application from the marketplace. 
    ```
    cf create-service spacebears default sb-instance
    helm list
    cf create-service-key sb-instance sb-service-key
    cf services
    cf service sb-instance
    ```
1) Create a service key to bind to the service instance. 
    ```
    cf create-service-key sb-instance sb-service-key
    ```

#### Run
```bash
make run
```

### ci
* https://concourse.cfplatformeng.com/teams/main/pipelines/kibosh

## Notes

Inline-style: 
![](SeqDiagram.png)

Diagram source https://www.websequencediagrams.com/ + 
```text
object operator user cf generic_broker k8s app
operator->cf: deploy tile w/ generic_broker+elastic search configured
generic_broker->k8s: create odb kubo cluster
generic_broker->cf: add self to marketplaces (errand?)
user->cf: create elastic search service
cf->generic_broker: create-service API call
generic_broker-> k8s: kubectl create new namespace
generic_broker-> k8s: kubectl to create deployment in namespace
user->cf: bind-service
cf->generic_broker: bind-service
generic_broker-> k8s: kubectl get secrets / kubectl get services
k8s->generic_broker: secrets and services
generic_broker->cf: secrets and services in cf format
cf->app: secrets and services as env vars
```
