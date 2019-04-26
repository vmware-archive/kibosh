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

package k8s_test

import (
	"encoding/json"
	"errors"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/cf-platform-eng/kibosh/pkg/config"
	. "github.com/cf-platform-eng/kibosh/pkg/k8s"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	api_v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sAPI "k8s.io/client-go/tools/clientcmd/api"
)

var _ = Describe("Config", func() {
	var creds *config.ClusterCredentials

	BeforeEach(func() {
		creds = &config.ClusterCredentials{
			CAData: []byte("c29tZSByYW5kb20gc3R1ZmY="),
			Server: "127.0.0.1/api",
			Token:  "my-token",
		}
	})

	Context("real delegate", func() {
		It("list pods", func() {
			var url string
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				url = string(r.URL.Path)
			})
			testserver := httptest.NewServer(handler)
			creds.Server = testserver.URL

			cluster, err := NewCluster(creds)

			Expect(err).To(BeNil())

			cluster.ListPods("mynamespace", meta_v1.ListOptions{})

			Expect(url).To(Equal("/api/v1/namespaces/mynamespace/pods"))
		})

		It("loads default config", func() {
			configFile, err := ioutil.TempFile("", "")
			Expect(err).To(BeNil())

			configFile.Write([]byte(`
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: Zm9v
    server: https://127.0.0.1
  name: my_cluster
contexts:
- context:
    cluster: my_cluster
    user: my_cluster_user
  name: my_cluster
current-context: my_cluster
kind: Config
preferences: {}
users:
- name: my_cluster_user
  user:
    token: cGFzc3dvcmQ=
`))

			os.Setenv("KUBECONFIG", configFile.Name())

			cluster, err := NewClusterFromDefaultConfig()

			Expect(err).To(BeNil())
			clientConfig := cluster.GetClientConfig()
			Expect(clientConfig).NotTo(BeNil())
			Expect(clientConfig.BearerToken).To(Equal("cGFzc3dvcmQ="))
		})

		It("load specific config", func() {
			k8sConfig := &k8sAPI.Config{
				Clusters: map[string]*k8sAPI.Cluster{
					"cluster1": {
						CertificateAuthorityData: []byte("my cat"),
						Server:                   "myserver",
					},
					"cluster2": {
						CertificateAuthorityData: []byte("my cat"),
						Server:                   "myserver",
					},
				},
				CurrentContext: "context2",
				Contexts: map[string]*k8sAPI.Context{
					"context1": {
						Cluster:  "cluster1",
						AuthInfo: "auth1",
					},
					"context2": {
						Cluster:  "cluster2",
						AuthInfo: "auth2",
					},
				},
				AuthInfos: map[string]*k8sAPI.AuthInfo{
					"auth1": {
						Token: "my encoded token",
					},
					"auth2": {
						Token: "my encoded 2nd token",
					},
				},
			}

			cluster, err := GetClusterFromK8sConfig(k8sConfig)

			Expect(err).To(BeNil())

			clientConfig := cluster.GetClientConfig()

			Expect(clientConfig).NotTo(BeNil())
			Expect(clientConfig.BearerToken).To(Equal("my encoded 2nd token"))
		})

		It("no current context", func() {
			k8sConfig := &k8sAPI.Config{
				Clusters: map[string]*k8sAPI.Cluster{
					"cluster1": {
						CertificateAuthorityData: []byte("my cat"),
						Server:                   "myserver",
					},
					"cluster2": {
						CertificateAuthorityData: []byte("my cat"),
						Server:                   "myserver",
					},
				},
				Contexts: map[string]*k8sAPI.Context{
					"context1": {
						Cluster:  "cluster1",
						AuthInfo: "auth1",
					},
					"context2": {
						Cluster:  "cluster2",
						AuthInfo: "auth2",
					},
				},
				AuthInfos: map[string]*k8sAPI.AuthInfo{
					"auth1": {
						Token: "my encoded token",
					},
					"auth2": {
						Token: "my encoded 2nd token",
					},
				},
			}

			_, err := GetClusterFromK8sConfig(k8sConfig)

			Expect(err).NotTo(BeNil())
		})
	})

	Context("mock delegate", func() {
		var fakeClusterDelegate k8sfakes.FakeClusterDelegate

		BeforeEach(func() {
			fakeClusterDelegate = k8sfakes.FakeClusterDelegate{}
		})

		It("create namespace is called when namespace doesn't exist", func() {
			statusError := &k8s_errors.StatusError{ErrStatus: meta_v1.Status{
				Reason: meta_v1.StatusReasonNotFound},
			}
			fakeClusterDelegate = k8sfakes.FakeClusterDelegate{}
			fakeClusterDelegate.GetNamespaceReturns(nil, statusError)

			cluster, err := NewUnitTestCluster(&fakeClusterDelegate)
			Expect(err).To(BeNil())

			_, err = cluster.CreateNamespaceIfNotExists(&api_v1.Namespace{})
			Expect(err).To(BeNil())

			Expect(fakeClusterDelegate.CreateNamespaceCallCount()).To((Equal(1)))
		})

		It("create namespace not called when namespace exist", func() {
			fakeClusterDelegate = k8sfakes.FakeClusterDelegate{}
			fakeClusterDelegate.GetNamespaceReturns(&api_v1.Namespace{}, nil)

			cluster, err := NewUnitTestCluster(&fakeClusterDelegate)
			Expect(err).To(BeNil())

			_, err = cluster.CreateNamespaceIfNotExists(&api_v1.Namespace{})
			Expect(err).To(BeNil())

			Expect(fakeClusterDelegate.CreateNamespaceCallCount()).To((Equal(0)))
		})

		It("secrets and services returns externalIPs field when Service Type NodePort is used", func() {
			nodeList := api_v1.NodeList{
				Items: []api_v1.Node{
					{
						ObjectMeta: meta_v1.ObjectMeta{
							Labels: map[string]string{
								"spec.ip": "1.1.1.1",
							},
						},
					},
				},
			}
			secretsList := api_v1.SecretList{
				Items: []api_v1.Secret{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "passwords"},
						Data:       map[string][]byte{"db-password": []byte("abc123")},
						Type:       api_v1.SecretTypeOpaque,
					},
				},
			}
			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{},
							Type:  "NodePort",
						},
					},
				},
			}
			fakeClusterDelegate.ListNodesReturns(&nodeList, nil)
			fakeClusterDelegate.ListServicesReturns(&serviceList, nil)
			fakeClusterDelegate.ListSecretsReturns(&secretsList, nil)
			cluster, err := NewUnitTestCluster(&fakeClusterDelegate)
			Expect(err).To(BeNil())

			creds, err := cluster.GetSecretsAndServices("mynamespaceid")

			services := creds["services"]
			spec := services.([]map[string]interface{})[0]["spec"]
			externalIPs := spec.(api_v1.ServiceSpec).ExternalIPs
			Expect(externalIPs[0]).To(Equal("1.1.1.1"))
			namespace, _ := fakeClusterDelegate.ListSecretsArgsForCall(0)
			Expect(namespace).To(Equal("mynamespaceid"))
		})

		It("get secrets and services filters to only opaque secrets", func() {
			serviceList := api_v1.ServiceList{Items: []api_v1.Service{}}
			fakeClusterDelegate.ListServicesReturns(&serviceList, nil)

			secretsList := api_v1.SecretList{
				Items: []api_v1.Secret{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "passwords"},
						Data:       map[string][]byte{"db-password": []byte("abc123")},
						Type:       api_v1.SecretTypeOpaque,
					}, {
						ObjectMeta: meta_v1.ObjectMeta{Name: "default-token-xyz"},
						Data:       map[string][]byte{"token": []byte("my-token")},
						Type:       api_v1.SecretTypeServiceAccountToken,
					},
				},
			}
			fakeClusterDelegate.ListSecretsReturns(&secretsList, nil)

			cluster, err := NewUnitTestCluster(&fakeClusterDelegate)
			Expect(err).To(BeNil())

			creds, err := cluster.GetSecretsAndServices("mynamespaceid")
			Expect(fakeClusterDelegate.ListSecretsCallCount()).To(Equal(1))

			secrets := creds["secrets"]
			secretsJson, err := json.Marshal(secrets)
			Expect(string(secretsJson)).To(Equal(`[{"data":{"db-password":"abc123"},"name":"passwords"}]`))
		})

		It("bubbles up list secrets errors", func() {
			serviceList := api_v1.ServiceList{Items: []api_v1.Service{}}
			fakeClusterDelegate.ListServicesReturns(&serviceList, nil)

			fakeClusterDelegate.ListSecretsReturns(nil, errors.New("foo failed"))

			cluster, err := NewUnitTestCluster(&fakeClusterDelegate)
			Expect(err).To(BeNil())

			_, err = cluster.GetSecretsAndServices("mynamespaceid")

			Expect(fakeClusterDelegate.ListSecretsCallCount()).To(Equal(1))
		})

		It("returns services (load balancer stuff)", func() {
			secretList := api_v1.SecretList{Items: []api_v1.Secret{}}
			fakeClusterDelegate.ListSecretsReturns(&secretList, nil)

			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{
								{
									Name:     "mysql",
									NodePort: 30092,
									Port:     3306,
									Protocol: "TCP",
								},
							},
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
			fakeClusterDelegate.ListServicesReturns(&serviceList, nil)

			cluster, err := NewUnitTestCluster(&fakeClusterDelegate)
			Expect(err).To(BeNil())

			creds, err := cluster.GetSecretsAndServices("mynamespaceid")

			Expect(fakeClusterDelegate.ListServicesCallCount()).To(Equal(1))

			services := creds["services"]
			name := services.([]map[string]interface{})[0]["name"]
			Expect(name).To(Equal("kibosh-my-mysql-db-instance"))

			spec := services.([]map[string]interface{})[0]["spec"]
			specJson, _ := json.Marshal(spec)
			Expect(string(specJson)).To(Equal(`{"ports":[{"name":"mysql","protocol":"TCP","port":3306,"targetPort":0,"nodePort":30092}]}`))

			status := services.([]map[string]interface{})[0]["status"]
			statusJson, _ := json.Marshal(status)
			Expect(string(statusJson)).To(Equal(`{"loadBalancer":{"ingress":[{"ip":"127.0.0.1"}]}}`))
		})

		It("bubbles up list services errors", func() {
			secretList := api_v1.SecretList{Items: []api_v1.Secret{}}
			fakeClusterDelegate.ListSecretsReturns(&secretList, nil)

			fakeClusterDelegate.ListServicesReturns(nil, errors.New("no services for you"))

			cluster, err := NewUnitTestCluster(&fakeClusterDelegate)
			Expect(err).To(BeNil())

			_, err = cluster.GetSecretsAndServices("mynamespaceid")
			Expect(err).NotTo(BeNil())

		})
	})
})
