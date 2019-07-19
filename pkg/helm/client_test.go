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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
	"os"

	"github.com/ghodss/yaml"

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
			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{},
							Type:  "LoadBalancer",
						},
						Status: api_v1.ServiceStatus{},
					},
				},
			}
			fakeCluster.ListServicesReturns(&serviceList, nil)

			message, statusCode, err := myHelmClient.ReleaseReadiness("myRelease", "myInstance", fakeCluster)

			Expect(err).To(BeNil())
			Expect(*message).To(Equal("service deployment load balancer in progress"))
			Expect(statusCode).To(Equal(hapi_release.Status_PENDING_INSTALL))
		})

		It("waits until pods are running", func() {
			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{},
							Type:  "LoadBalancer",
						},
						Status: api_v1.ServiceStatus{
							LoadBalancer: api_v1.LoadBalancerStatus{
								Ingress: []api_v1.LoadBalancerIngress{
									{IP: "127.0.0.1"},
								},
							},
						},
					},
				},
			}
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := api_v1.PodList{
				Items: []api_v1.Pod{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "pod1"},
						Spec:       api_v1.PodSpec{},
						Status: api_v1.PodStatus{
							Phase: "Pending",
							Conditions: []api_v1.PodCondition{
								{
									Status:  "False",
									Type:    "PodScheduled",
									Reason:  "Unschedulable",
									Message: "0/1 nodes are available: 1 Insufficient memory",
								},
							},
						},
					},
				},
			}
			fakeCluster.ListPodsReturns(&podList, nil)

			message, statusCode, err := myHelmClient.ReleaseReadiness("myRelease", "myInstance", fakeCluster)

			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_PENDING_INSTALL))
			Expect(*message).To(ContainSubstring("0/1 nodes are available: 1 Insufficient memory"))
		})

		It("considers a pod status of Completed as meaning the pod succeeded", func() {
			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{},
							Type:  "LoadBalancer",
						},
						Status: api_v1.ServiceStatus{
							LoadBalancer: api_v1.LoadBalancerStatus{
								Ingress: []api_v1.LoadBalancerIngress{
									{IP: "127.0.0.1"},
								},
							},
						},
					},
				},
			}
			fakeCluster.ListServicesReturns(&serviceList, nil)

			podList := api_v1.PodList{
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
							Phase: "Succeeded",
						},
					},
				},
			}
			fakeCluster.ListPodsReturns(&podList, nil)

			message, statusCode, err := myHelmClient.ReleaseReadiness("myRelease", "myInstance", fakeCluster)

			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_DEPLOYED))
			Expect(message).To(BeNil())
		})

		It("returns error when unable to list pods", func() {
			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{},
							Type:  "LoadBalancer",
						},
						Status: api_v1.ServiceStatus{
							LoadBalancer: api_v1.LoadBalancerStatus{
								Ingress: []api_v1.LoadBalancerIngress{
									{IP: "127.0.0.1"},
								},
							},
						},
					},
				},
			}
			fakeCluster.ListServicesReturns(&serviceList, nil)

			fakeCluster.ListPodsReturns(nil, errors.New("nope"))

			message, statusCode, err := myHelmClient.ReleaseReadiness("myRelease", "myInstance", fakeCluster)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("nope"))
			Expect(message).To(BeNil())
			Expect(statusCode).To(Equal(hapi_release.Status_UNKNOWN))
		})

		It("bubbles up error on list service failure", func() {
			errorMsg := "list services error"
			fakeCluster.ListServicesReturns(&api_v1.ServiceList{}, errors.New(errorMsg))

			message, statusCode, err := myHelmClient.ReleaseReadiness("myRelease", "myInstance", fakeCluster)

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(errorMsg))
			Expect(statusCode).To(Equal(hapi_release.Status_UNKNOWN))
			Expect(message).To(BeNil())
		})
	})
})
