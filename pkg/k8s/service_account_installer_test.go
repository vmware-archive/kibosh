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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
	"github.com/cf-platform-eng/kibosh/pkg/test"
	api_v1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Service Account Installer", func() {
	var logger lager.Logger
	var cluster k8sfakes.FakeCluster
	var installer k8s.ServiceAccountInstaller

	BeforeEach(func() {
		logger = lager.NewLogger("test")
		k8sClient := test.FakeK8sInterface{}
		cluster = k8sfakes.FakeCluster{}
		cluster.GetClientReturns(&k8sClient)

		installer = k8s.NewServiceAccountInstaller(&cluster, logger)
	})

	Context("ensure account", func() {
		BeforeEach(func() {
			//ensure we skip the second step, binding
			cluster.ListClusterRoleBindingsReturns(&rbacv1beta1.ClusterRoleBindingList{
				Items: []rbacv1beta1.ClusterRoleBinding{{}},
			}, nil)
		})

		It("Creates the account", func() {
			cluster.ListServiceAccountsReturns(&api_v1.ServiceAccountList{
				Items:    []api_v1.ServiceAccount{},
				ListMeta: meta_v1.ListMeta{},
				TypeMeta: meta_v1.TypeMeta{},
			}, nil)

			err := installer.Install()
			Expect(err).To(BeNil())

			Expect(cluster.CreateServiceAccountCallCount()).To(Equal(1))
			nameSpace, listOptions := cluster.ListServiceAccountsArgsForCall(0)
			Expect(listOptions.LabelSelector).To(Equal("kibosh=tiller-service-account"))
			Expect(nameSpace).To(Equal("kube-system"))

			nameSpace, serviceAccount := cluster.CreateServiceAccountArgsForCall(0)
			Expect(nameSpace).To(Equal("kube-system"))
			Expect(serviceAccount.Labels["kibosh"]).To(Equal("tiller-service-account"))
			Expect(serviceAccount.Name).To(Equal("tiller"))

		})

		It("Skips creation if the service account exists", func() {
			cluster.ListServiceAccountsReturns(&api_v1.ServiceAccountList{
				Items: []api_v1.ServiceAccount{
					{},
				},
				ListMeta: meta_v1.ListMeta{},
				TypeMeta: meta_v1.TypeMeta{},
			}, nil)

			err := installer.Install()
			Expect(err).To(BeNil())

			Expect(cluster.CreateServiceAccountCallCount()).To(Equal(0))
		})
	})

	Context("ensure role", func() {
		BeforeEach(func() {
			cluster.ListServiceAccountsReturns(&api_v1.ServiceAccountList{
				Items: []api_v1.ServiceAccount{{}},
			}, nil)
		})

		It("Creates the service role binding", func() {
			cluster.ListClusterRoleBindingsReturns(&rbacv1beta1.ClusterRoleBindingList{
				Items: []rbacv1beta1.ClusterRoleBinding{},
			}, nil)

			err := installer.Install()
			Expect(err).To(BeNil())

			listOptions := cluster.ListClusterRoleBindingsArgsForCall(0)
			Expect(listOptions.LabelSelector).To(Equal("kibosh=tiller-service-admin-binding"))

			Expect(cluster.CreateClusterRoleBindingCallCount()).To(Equal(1))
			clusterRoleBinding := cluster.CreateClusterRoleBindingArgsForCall(0)
			Expect(clusterRoleBinding.Name).To(Equal("tiller-cluster-admin"))
			Expect(clusterRoleBinding.Labels["kibosh"]).To(Equal("tiller-service-admin-binding"))
			Expect(clusterRoleBinding.RoleRef.Name).To(Equal("cluster-admin"))
			Expect(clusterRoleBinding.RoleRef.Kind).To(Equal("ClusterRole"))
			Expect(clusterRoleBinding.RoleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
			Expect(clusterRoleBinding.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(clusterRoleBinding.Subjects[0].Name).To(Equal("tiller"))
			Expect(clusterRoleBinding.Subjects[0].Namespace).To(Equal("kube-system"))
		})

		It("Skips creation if the role exists", func() {
			cluster.ListClusterRoleBindingsReturns(&rbacv1beta1.ClusterRoleBindingList{
				Items: []rbacv1beta1.ClusterRoleBinding{
					{},
				},
			}, nil)

			err := installer.Install()
			Expect(err).To(BeNil())

			Expect(cluster.CreateClusterRoleBindingCallCount()).To(Equal(0))
		})
	})

})
