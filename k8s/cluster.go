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

package k8s

import (
	"github.com/cf-platform-eng/kibosh/config"

	api_v1 "k8s.io/api/core/v1"
	v1_beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//go:generate counterfeiter ./ Cluster
type Cluster interface {
	GetClient() kubernetes.Interface
	GetClientConfig() *rest.Config

	CreateNamespace(*api_v1.Namespace) (*api_v1.Namespace, error)
	DeleteNamespace(name string, options *meta_v1.DeleteOptions) error
	ListPods(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.PodList, error)
	GetDeployment(string, string, meta_v1.GetOptions) (*v1_beta1.Deployment, error)
	ListServiceAccounts(string, meta_v1.ListOptions) (*api_v1.ServiceAccountList, error)
	CreateServiceAccount(string, *api_v1.ServiceAccount) (*api_v1.ServiceAccount, error)
	ListClusterRoleBindings(meta_v1.ListOptions) (*rbacv1beta1.ClusterRoleBindingList, error)
	CreateClusterRoleBinding(*rbacv1beta1.ClusterRoleBinding) (*rbacv1beta1.ClusterRoleBinding, error)
	CreateSecret(nameSpace string, secret *api_v1.Secret) (*api_v1.Secret, error)
	ListSecrets(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.SecretList, error)
	ListServices(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.ServiceList, error)
	Patch(nameSpace string, name string, pt types.PatchType, data []byte, subresources ...string) (result *api_v1.ServiceAccount, err error)
}

type cluster struct {
	credentials *config.ClusterCredentials
	client      kubernetes.Interface
	k8sConfig   *rest.Config
}

func NewCluster(kuboConfig *config.ClusterCredentials) (Cluster, error) {
	k8sConfig, err := buildClientConfig(kuboConfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	return &cluster{
		credentials: kuboConfig,
		k8sConfig:   k8sConfig,
		client:      client,
	}, nil
}

func (cluster *cluster) GetClientConfig() *rest.Config {
	return cluster.k8sConfig
}

func buildClientConfig(credentials *config.ClusterCredentials) (*rest.Config, error) {
	tlsClientConfig := rest.TLSClientConfig{
		CAData: []byte(credentials.CAData),
	}

	return &rest.Config{
		Host:            credentials.Server,
		BearerToken:     credentials.Token,
		TLSClientConfig: tlsClientConfig,
	}, nil
}

func (cluster *cluster) GetClient() kubernetes.Interface {
	return cluster.client
}

func (cluster *cluster) CreateNamespace(namespace *api_v1.Namespace) (*api_v1.Namespace, error) {
	return cluster.GetClient().CoreV1().Namespaces().Create(namespace)
}

func (cluster *cluster) DeleteNamespace(name string, options *meta_v1.DeleteOptions) error {
	return cluster.GetClient().CoreV1().Namespaces().Delete(name, options)
}

func (cluster *cluster) ListPods(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.PodList, error) {
	pods, err := cluster.client.CoreV1().Pods(nameSpace).List(listOptions)
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func (cluster *cluster) GetDeployment(nameSpace string, deploymentName string, getOptions meta_v1.GetOptions) (*v1_beta1.Deployment, error) {
	return cluster.GetClient().ExtensionsV1beta1().Deployments(nameSpace).Get(deploymentName, meta_v1.GetOptions{})
}

func (cluster *cluster) ListServiceAccounts(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.ServiceAccountList, error) {
	return cluster.GetClient().CoreV1().ServiceAccounts(nameSpace).List(listOptions)
}

func (cluster *cluster) CreateServiceAccount(nameSpace string, serviceAccount *api_v1.ServiceAccount) (*api_v1.ServiceAccount, error) {
	return cluster.GetClient().CoreV1().ServiceAccounts(nameSpace).Create(serviceAccount)
}

func (cluster *cluster) ListClusterRoleBindings(listOptions meta_v1.ListOptions) (*rbacv1beta1.ClusterRoleBindingList, error) {
	return cluster.GetClient().RbacV1beta1().ClusterRoleBindings().List(listOptions)
}

func (cluster *cluster) CreateClusterRoleBinding(clusterRoleBinding *rbacv1beta1.ClusterRoleBinding) (*rbacv1beta1.ClusterRoleBinding, error) {
	return cluster.GetClient().RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding)
}

func (cluster *cluster) CreateSecret(nameSpace string, secret *api_v1.Secret) (*api_v1.Secret, error) {
	return cluster.GetClient().CoreV1().Secrets(nameSpace).Create(secret)
}

func (cluster *cluster) ListSecrets(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.SecretList, error) {
	return cluster.GetClient().CoreV1().Secrets(nameSpace).List(listOptions)
}

func (cluster *cluster) ListServices(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.ServiceList, error) {
	return cluster.GetClient().CoreV1().Services(nameSpace).List(listOptions)
}

func (cluster *cluster) Patch(nameSpace string, name string, pt types.PatchType, data []byte, subresources ...string) (result *api_v1.ServiceAccount, err error) {
	return cluster.GetClient().CoreV1().ServiceAccounts(nameSpace).Patch(name, pt, data, subresources...)
}
