// kibosh
//
// Copyright (c) 2017-Present Pivotal Software, Inc. All Rights Reserved.
//
// This program and the accompanying materials are made available under the terms of the under the Apache License,
// Version 2.0 (the "License”); you may not use this file except in compliance with the License. You may
// obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the
// License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing permissions and
// limitations under the License.

package helm_test

import (
	"errors"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
	"os"

	. "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
)

var _ = Describe("Client", func() {
	var myHelmClient MyHelmClient
	var chartPath string
	var fakeCluster *k8sfakes.FakeCluster

	BeforeEach(func() {
		fakeCluster = &k8sfakes.FakeCluster{}

		myHelmClient = NewMyHelmClient(fakeCluster, nil, "my-kibosh-namespace", nil)

		var err error
		chartPath, err = ioutil.TempDir("", "chart-")
		Expect(err).To(BeNil())
	})

	BeforeEach(func() {
		os.RemoveAll(chartPath)
	})

	It("merge values bytres overrides", func() {
		base := []byte(`
foo: bar
`)
		override := []byte(`
foo: not bar
`)

		mergedBytes, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).To(BeNil())

		merged := map[string]interface{}{}
		err = yaml.Unmarshal(mergedBytes, &merged)
		Expect(err).To(BeNil())
		Expect(merged).To(Equal(map[string]interface{}{
			"foo": "not bar",
		}))
	})

	It("keeps non-specified base values", func() {
		base := []byte(`
foo: bar
baz: qux
`)
		override := []byte(`
foo: not bar
`)

		mergedBytes, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).To(BeNil())

		merged := map[string]interface{}{}
		err = yaml.Unmarshal(mergedBytes, &merged)
		Expect(err).To(BeNil())
		Expect(merged).To(Equal(map[string]interface{}{
			"foo": "not bar",
			"baz": "qux",
		}))
	})

	It("add override values not in base", func() {
		base := []byte(`
foo: bar
`)
		override := []byte(`
foo: not bar
baz: qux
`)

		mergedBytes, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).To(BeNil())

		merged := map[string]interface{}{}
		err = yaml.Unmarshal(mergedBytes, &merged)
		Expect(err).To(BeNil())
		Expect(merged).To(Equal(map[string]interface{}{
			"foo": "not bar",
			"baz": "qux",
		}))
	})

	It("nested override", func() {
		base := []byte(`
images:
  thing1:
    image: "my-first-image"
    imageTag: "5.7.14"
  thing2:
    image: "my-second-image"
    imageTag: "1.2.3"
`)
		override := []byte(`
images:
  thing1:
    image: "example.com/my-first-image"
`)

		mergedBytes, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).To(BeNil())

		merged := map[string]interface{}{}
		err = yaml.Unmarshal(mergedBytes, &merged)
		Expect(err).To(BeNil())

		Expect(merged).To(Equal(map[string]interface{}{
			"images": map[string]interface{}{
				"thing1": map[string]interface{}{
					"image":    "example.com/my-first-image",
					"imageTag": "5.7.14",
				},
				"thing2": map[string]interface{}{
					"image":    "my-second-image",
					"imageTag": "1.2.3",
				},
			},
		}))
	})

	It("returns an error when the override file is invalid", func() {
		base := []byte(`
foo: bar
`)
		override := []byte(`
- foo: "bar2"
`)
		_, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).ToNot(BeNil())
	})

	Context("Readiness checks", func() {

		It("waits until load balancer servers have ingress", func() {
			serviceList := serviceTemplate(false)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).To(BeNil())
			Expect(*message).To(Equal("service deployment load balancer in progress"))
			Expect(statusCode).To(Equal(hapi_release.Status_PENDING_INSTALL))
		})

		It("waits until pods are running", func() {
			serviceList := serviceTemplate(true)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := podTemplate("Pending")

			errMsg := "0/1 nodes are available: 1 Insufficient memory"
			condition := []api_v1.PodCondition{
				{
					Message: errMsg,
				},
			}
			podList.Items[0].Status.Conditions = condition
			fakeCluster.ListPodsReturns(&podList, nil)

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_PENDING_INSTALL))
			Expect(*message).To(ContainSubstring(errMsg))
		})

		It("wait until volume claims are bound", func() {
			serviceList := serviceTemplate(true)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := podTemplate("Succeeded")
			fakeCluster.ListPodsReturns(&podList, nil)

			volumeClaimList := PVCTemplate("Available")

			fakeCluster.ListPersistentVolumesReturns(&volumeClaimList, nil)

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_PENDING_INSTALL))
			Expect(*message).To(ContainSubstring("PersistentVolumeClaim is not ready:"))
		})

		It("wait until deployments are ready", func() {
			serviceList := serviceTemplate(true)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := podTemplate("Succeeded")
			fakeCluster.ListPodsReturns(&podList, nil)

			volumeClaimList := PVCTemplate("Bound")
			fakeCluster.ListPersistentVolumesReturns(&volumeClaimList, nil)

			deploymentsList := deploymentTemplate(false)

			fakeCluster.ListDeploymentsReturns(&deploymentsList, nil)

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_PENDING_INSTALL))
			Expect(*message).To(ContainSubstring("Deployment is not ready:"))
		})

		It("considers a pod status of Completed as meaning the pod succeeded", func() {
			serviceList := serviceTemplate(true)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := podTemplate("Succeeded")
			fakeCluster.ListPodsReturns(&podList, nil)

			volumeClaimList := PVCTemplate("Bound")
			fakeCluster.ListPersistentVolumesReturns(&volumeClaimList, nil)

			deploymentsList := deploymentTemplate(true)
			fakeCluster.ListDeploymentsReturns(&deploymentsList, nil)

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_DEPLOYED))
			Expect(message).To(BeNil())
		})

		It("Considers a volume claims in phase bound as succeeded ", func() {
			serviceList := serviceTemplate(true)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := podTemplate("Succeeded")
			fakeCluster.ListPodsReturns(&podList, nil)

			volumeClaimList := PVCTemplate("Bound")
			fakeCluster.ListPersistentVolumesReturns(&volumeClaimList, nil)

			deploymentsList := deploymentTemplate(true)
			fakeCluster.ListDeploymentsReturns(&deploymentsList, nil)

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_DEPLOYED))
			Expect(message).To(BeNil())
		})

		It("Consider a Deployment as ready if it meets maxUnavailable setting", func() {
			serviceList := serviceTemplate(true)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := podTemplate("Succeeded")
			fakeCluster.ListPodsReturns(&podList, nil)

			volumeClaimList := PVCTemplate("Bound")
			fakeCluster.ListPersistentVolumesReturns(&volumeClaimList, nil)
			deploymentsList := deploymentTemplate(true)

			fakeCluster.ListDeploymentsReturns(&deploymentsList, nil)

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_DEPLOYED))
			Expect(message).To(BeNil())
		})

		It("returns an error when unable to list services", func() {
			errorMsg := "list services error"
			fakeCluster.ListServicesReturns(&api_v1.ServiceList{}, errors.New(errorMsg))

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(errorMsg))
			Expect(statusCode).To(Equal(hapi_release.Status_UNKNOWN))
			Expect(message).To(BeNil())
		})

		It("returns error when unable to list pods", func() {
			serviceList := serviceTemplate(true)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			fakeCluster.ListPodsReturns(nil, errors.New("nope"))

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("nope"))
			Expect(message).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_UNKNOWN))
		})

		It("returns error when unable to list persistent volume claims", func() {
			serviceList := serviceTemplate(true)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := podTemplate("Succeeded")
			fakeCluster.ListPodsReturns(&podList, nil)

			volumeClaimList := PVCTemplate("Available")
			errMessage := "bad volume list"
			fakeCluster.ListPersistentVolumesReturns(&volumeClaimList, errors.New(errMessage))

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).ToNot(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_UNKNOWN))
			Expect(message).To(BeNil())
			Expect(err.Error()).To(ContainSubstring(errMessage))
		})

		It("returns error when unable to list deployments", func() {
			serviceList := serviceTemplate(true)
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := podTemplate("Succeeded")
			fakeCluster.ListPodsReturns(&podList, nil)

			volumeClaimList := PVCTemplate("Bound")
			fakeCluster.ListPersistentVolumesReturns(&volumeClaimList, nil)

			errMessage := "deployment list error"

			fakeCluster.ListDeploymentsReturns(nil, errors.New(errMessage))

			message, statusCode, err := myHelmClient.ResourceReadiness("myNamespace", fakeCluster)

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(errMessage))
			Expect(statusCode).To(Equal(hapi_release.Status_UNKNOWN))
			Expect(message).To(BeNil())
		})

	})
})

func deploymentTemplate(readyStatus bool) k8s.DeploymentList {
	readyReplicas := int32(1)
	if readyStatus {
		readyReplicas = 3
	}
	unavailableMax := intstr.FromInt(1)
	replicaCount := int32(3)

	return k8s.DeploymentList{
		Items: []k8s.Deployment{
			{
				ReplicaSets: &appsv1.ReplicaSet{
					ObjectMeta: meta_v1.ObjectMeta{
						Name: "replicaset1",
						Labels: map[string]string{
							"job-name": "test",
						},
					},
					Spec: appsv1.ReplicaSetSpec{},
					Status: appsv1.ReplicaSetStatus{
						ReadyReplicas: readyReplicas,
					},
				},
				Deployment: &appsv1.Deployment{
					ObjectMeta: meta_v1.ObjectMeta{
						Name: "deployment1",
						Labels: map[string]string{
							"job-name": "test",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							RollingUpdate: &appsv1.RollingUpdateDeployment{
								MaxUnavailable: &unavailableMax,
							},
						},
						Replicas: &replicaCount,
					},
					Status: appsv1.DeploymentStatus{},
				},
			},
		},
	}
}

func serviceTemplate(ready bool) api_v1.ServiceList {
	var ingress []api_v1.LoadBalancerIngress

	if ready {
		ipAddress := api_v1.LoadBalancerIngress{
			IP: "127.0.0.1",
		}
		ingress = append(ingress, ipAddress)
	}
	return api_v1.ServiceList{
		Items: []api_v1.Service{
			{
				ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
				Spec: api_v1.ServiceSpec{
					Ports: []api_v1.ServicePort{},
					Type:  "LoadBalancer",
				},
				Status: api_v1.ServiceStatus{
					LoadBalancer: api_v1.LoadBalancerStatus{
						Ingress: ingress,
					},
				},
			},
		},
	}
}

func podTemplate(phase api_v1.PodPhase) api_v1.PodList {

	return api_v1.PodList{
		Items: []api_v1.Pod{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "pod1",
					Labels: map[string]string{
						"job-name": "test",
					},
				},
				Spec: api_v1.PodSpec{},
				Status: api_v1.PodStatus{
					Phase: phase,
				},
			},
		},
	}
}

func PVCTemplate(phase api_v1.PersistentVolumeClaimPhase) api_v1.PersistentVolumeClaimList {

	return api_v1.PersistentVolumeClaimList{
		Items: []api_v1.PersistentVolumeClaim{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "volumeClaim1",
					Labels: map[string]string{
						"job-name": "test",
					},
				},
				Spec: api_v1.PersistentVolumeClaimSpec{},
				Status: api_v1.PersistentVolumeClaimStatus{
					Phase: phase,
				},
			},
		},
	}
}
