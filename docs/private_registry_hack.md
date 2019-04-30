# Using local docker private registry

This is temporary work around until we fully integrate kibosh to work with various private registry options. It uses
a private
[helm based docker-registry](https://github.com/kubernetes/charts/tree/master/stable/docker-registry)
pushed to the cluster itself.



### Installation
* Ensure your local `kubectl` is talking to the pks cluster
* Grab the latest helm binary (helm will talk to the same cluster your local kubernetes is configured to talk to)

### Adding the docker registry

Add a service account for tiller:
```bash
echo "apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system" > tiller_rbac.yaml

kubectl create -f tiller_rbac.yaml
```

Add helm to the cluster
```bash
helm init --service-account tiller
```

Install the registry chart
```bash
git clone https://github.com/kubernetes/charts/
cd charts
helm install --set persistence.storageClass="standard" --set service.type="NodePort" ./stable/docker-registry
```

### Whitelist the registry for the cluster

Get the external port for ingress to the deployment
```bash
kubectl get services -l "app=docker-registry" -o jsonpath="{.items[0].spec.ports[0].nodePort}"
```

Get the IP of one of the *workers* on the service deployment from the OpsMan VM
```bash
bosh -d service-instance_some_random_guid vms
```

To get the pull to work in k8s, either trust the registry's CA certificate
* [the way harbor does](https://docs.pivotal.io/partners/vmware-harbor/integrating-pks.html)
* Or whitelist the worker nodes' [insecure-registries](https://github.com/cloudfoundry-incubator/docker-boshrelease/blob/master/jobs/docker/spec#L70-L71)
    - Download the manifest and add a property to the worker nodes that whitelists one of the worker node's private IP
      with the
      For example:
      ```yaml
        jobs:
        ...
        - name: docker
          release: docker
          properties:
            insecure_registries:
            - 192.168.20.21:30141
        ...
      ```
    - Redeploy the changed manifest

#### Adding your image

Use `kubectl` to setup a local proxy to talk to the docker registry (it's not otherwise reachable from your local laptop)

```bash
export POD_NAME=$(kubectl get pods --namespace default -l "app=docker-registry" -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward $POD_NAME 8080:5000
```

This will leave the proxy running in the foreground

For mac
* Add `docker.for.mac.host.internal:8080` to insecure registries (docker system icon -> daemon -> insecure registries)
* If the address does not resolve try Docker Edge, that's what we tested with.
* See https://github.com/docker/for-mac/issues/1160 for some of the networking fun around why it's using that domain

Then build or tag and push
```bash
# building image locally
docker build . -t docker.for.mac.host.internal:8080/my-debug

# or re-tag the image if you already have it
docker tag cfplatformeng/debug docker.for.mac.host.internal:8080/my-debug

# finally, in both push it
docker push docker.for.mac.host.internal:8080/my-debug
```

Get the `ClusterIP` used by the registry:

```bash
kubectl get services -l "app=docker-registry" -o jsonpath="{.items[0].spec.clusterIP}"
```

Change your your helm chart to point to that `ClusterIP`

```yaml
---
spec:
  containers:
  - image: <woker ip from bosh deployment : NodePort>/my-debug
```
