# pks-generic-broker

### Dev
Install Go depenencies
```bash
go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega
go get github.com/maxbrunsfeld/counterfeiter
```


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
