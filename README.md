# pks-generic-broker

A generic broker bridging the gap between Kubernetes and CF brokered services.

## Dev
#### Setup
Install Go depenencies
```bash
go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega
go get github.com/maxbrunsfeld/counterfeiter
```

#### Test
```bash
make test
```

#### Run
```bash
make run
```

### ci
* https://concourse.cfplatformeng.com/teams/main/pipelines/pks-generic-broker

## Notes


```
+-----------+ +-------+                                         +-----+                                        +-----------------+                                  +-----+ +-----+
| operator  | | user  |                                         | cf  |                                        | generic_broker  |                                  | k8s | | app |
+-----------+ +-------+                                         +-----+                                        +-----------------+                                  +-----+ +-----+
      |           |                                                |                                                    |                                              |       |
      | deploy tile w/ generic_broker+elastic search configured    |                                                    |                                              |       |
      |----------------------------------------------------------->|                                                    |                                              |       |
      |           |                                                |                                                    |                                              |       |
      |           |                                                |                                                    | create odb kubo cluster                      |       |
      |           |                                                |                                                    |--------------------------------------------->|       |
      |           |                                                |                                                    |                                              |       |
      |           |                                                |                 add self to marketplaces (errand?) |                                              |       |
      |           |                                                |<---------------------------------------------------|                                              |       |
      |           |                                                |                                                    |                                              |       |
      |           | create elastic search service                  |                                                    |                                              |       |
      |           |----------------------------------------------->|                                                    |                                              |       |
      |           |                                                |                                                    |                                              |       |
      |           |                                                | create-service API call                            |                                              |       |
      |           |                                                |--------------------------------------------------->|                                              |       |
      |           |                                                |                                                    |                                              |       |
      |           |                                                |                                                    | kubectl create new namespace                 |       |
      |           |                                                |                                                    |--------------------------------------------->|       |
      |           |                                                |                                                    |                                              |       |
      |           |                                                |                                                    | kubectl to create deployment in namespace    |       |
      |           |                                                |                                                    |--------------------------------------------->|       |
      |           |                                                |                                                    |                                              |       |
      |           |                                                |                                                    |                         secrets and services |       |
      |           |                                                |                                                    |<---------------------------------------------|       |
      |           |                                                |                                                    |                                              |       |
      |           | bind-service                                   |                                                    |                                              |       |
      |           |----------------------------------------------->|                                                    |                                              |       |
      |           |                                                |                                                    |                                              |       |
      |           |                                                | give me creds                                      |                                              |       |
      |           |                                                |--------------------------------------------------->|                                              |       |
      |           |                                                |                                                    |                                              |       |
      |           |                                                |          secrets and services in credentials block |                                              |       |
      |           |                                                |<---------------------------------------------------|                                              |       |
      |           |                                                |                                                    |                                              |       |
      |           |                                                | secrets and services in credentials block          |                                              |       |
      |           |                                                |---------------------------------------------------------------------------------------------------------->|
      |           |                                                |                                                    |                                              |       |
```

Diagram source http://textart.io/sequence + 
```text
object operator user cf generic_broker k8s app
operator->cf: deploy tile w/ generic_broker+elastic search configured
generic_broker->k8s: create odb kubo cluster
generic_broker->cf: add self to marketplaces (errand?)
user->cf: create elastic search service
cf->generic_broker: create-service API call
generic_broker-> k8s: kubectl create new namespace
generic_broker-> k8s: kubectl to create deployment in namespace
k8s->generic_broker: secrets and services
user->cf: bind-service
cf->generic_broker: give me creds
generic_broker->cf: secrets and services in credentials block
cf->app: secrets and services in credentials block
```
