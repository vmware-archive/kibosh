# Kibosh

A generic broker bridging the gap between Kubernetes and CF brokered services.

## Dev
#### Setup
Grab a newish version of Go (we used 1.9) 

Install Go depenencies
```bash
go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega
go get github.com/maxbrunsfeld/counterfeiter
```

#### Dependency vendoring
Dependency management is kind of a horror show here. For background, see
* https://github.com/kubernetes/client-go/blob/master/INSTALL.md
* https://github.com/kubernetes/client-go/issues/83

To vendor a new helm version:
* Cleanup existing k8s libraries from gopath
* Make sure your *don't* have other k8s repos (especially `client-go`) checked out or you'll likely run into conflicts
    - remove all `$GOPATH/src/k8s.io`
* Checkout helm and jump through the hoops to get it to compile
    - `make bootstrap` (this does a glide install strip vendor and deletes/moves several libraries)
* Swear a few times & then cross fingers
* Import deps
    - `govendor add +external`

To change dependencies, see [govendor](https://github.com/kardianos/govendor) docs for specific commands.

(Tried `dep`, but it added 10s of megabytes of golang.org/x/... to vendor)

#### Test
```bash
make test
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
