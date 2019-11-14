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
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	api_v1 "k8s.io/api/core/v1"
	v1_beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	k8sAPI "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	deploymentutil "k8s.io/kubernetes/pkg/controller/deployment/util"
)

//go:generate counterfeiter ./ Cluster
type Cluster interface {
	ClusterDelegate

	CreateNamespaceIfNotExists(*api_v1.Namespace) error
	NamespaceExists(namespaceName string) (bool, error)
	GetSecretsAndServices(namespace string) (map[string][]map[string]interface{}, error)
	SecretExists(namespaceName string, secretName string) (bool, error)
	CreateOrUpdateSecret(namespaceName string, secret *api_v1.Secret) (*api_v1.Secret, error)
	GetIngress(namespace string) ([]map[string]interface{}, error)
}

//go:generate counterfeiter ./ ClusterDelegate
type ClusterDelegate interface {
	GetClient() kubernetes.Interface
	GetClientConfig() *rest.Config

	GetDeployment(string, string, meta_v1.GetOptions) (*v1_beta1.Deployment, error)
	ListPods(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.PodList, error)
	CreateNamespace(*api_v1.Namespace) (*api_v1.Namespace, error)
	DeleteNamespace(name string, options *meta_v1.DeleteOptions) error
	GetNamespace(name string, options *meta_v1.GetOptions) (*api_v1.Namespace, error)
	GetNamespaces() (*api_v1.NamespaceList, error)
	ListServiceAccounts(string, meta_v1.ListOptions) (*api_v1.ServiceAccountList, error)
	CreateServiceAccount(string, *api_v1.ServiceAccount) (*api_v1.ServiceAccount, error)
	ListClusterRoleBindings(meta_v1.ListOptions) (*rbacv1beta1.ClusterRoleBindingList, error)
	CreateClusterRoleBinding(*rbacv1beta1.ClusterRoleBinding) (*rbacv1beta1.ClusterRoleBinding, error)
	CreateSecret(nameSpace string, secret *api_v1.Secret) (*api_v1.Secret, error)
	UpdateSecret(nameSpace string, secret *api_v1.Secret) (*api_v1.Secret, error)
	GetSecret(nameSpace string, name string, getOptions meta_v1.GetOptions) (*api_v1.Secret, error)
	ListNodes(listOptions meta_v1.ListOptions) (*api_v1.NodeList, error)
	ListSecrets(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.SecretList, error)
	ListServices(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.ServiceList, error)
	Patch(nameSpace string, name string, pt types.PatchType, data []byte, subresources ...string) (result *api_v1.ServiceAccount, err error)
	ListPersistentVolumes(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.PersistentVolumeClaimList, error)
	ListDeployments(nameSpace string, listOptions meta_v1.ListOptions) (*DeploymentList, error)
	ListIngress(nameSpace string, listOptions meta_v1.ListOptions) (*v1_beta1.IngressList, error)
}

type cluster struct {
	ClusterDelegate
}

type clusterDelegate struct {
	credentials    *config.ClusterCredentials
	client         kubernetes.Interface
	internalClient internalclientset.Interface
	k8sConfig      *rest.Config
}

type Deployment struct {
	ReplicaSets *appsv1.ReplicaSet
	Deployment  *appsv1.Deployment
}

type DeploymentList struct {
	Items []Deployment
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

	internalClient, err := internalclientset.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	cd := clusterDelegate{
		credentials:    kuboConfig,
		k8sConfig:      k8sConfig,
		client:         client,
		internalClient: internalClient,
	}

	return &cluster{
		ClusterDelegate: &cd,
	}, nil
}

func NewUnitTestCluster(cd ClusterDelegate) (Cluster, error) {
	return &cluster{
		ClusterDelegate: cd,
	}, nil
}

func NewClusterFromDefaultConfig() (Cluster, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	k8sConfig, err := loadingRules.Load()
	if err != nil {
		return nil, err
	}
	if k8sConfig.CurrentContext == "" {
		return nil, errors.New("the default Kubernetes config has no current context")
	}

	return GetClusterFromK8sConfig(k8sConfig)
}

func GetClusterFromK8sConfig(k8sConfig *k8sAPI.Config) (Cluster, error) {
	if k8sConfig.CurrentContext == "" {
		return nil, errors.New("the default Kubernetes config has no current context")
	}

	context := k8sConfig.Contexts[k8sConfig.CurrentContext]
	authInfo := k8sConfig.AuthInfos[context.AuthInfo]
	token := authInfo.Token
	server := k8sConfig.Clusters[context.Cluster].Server
	cert := k8sConfig.Clusters[context.Cluster].CertificateAuthorityData

	var creds *config.ClusterCredentials
	creds = &config.ClusterCredentials{
		CAData: cert,
		Server: server,
		Token:  token,
	}

	return NewCluster(creds)
}

func (cluster *cluster) CreateNamespaceIfNotExists(namespace *api_v1.Namespace) error {

	exists, err := cluster.NamespaceExists(namespace.Name)
	if err != nil {
		return err
	}

	if !exists {
		_, err = cluster.CreateNamespace(namespace)
	}
	return err
}

func (cluster *cluster) NamespaceExists(namespaceName string) (bool, error) {

	_, err := cluster.GetNamespace(namespaceName, nil)
	if err != nil {
		statusError, ok := err.(*k8s_errors.StatusError)
		if ok {
			if statusError.ErrStatus.Reason != meta_v1.StatusReasonNotFound {
				return false, err
			}
		} else {
			return false, err
		}

		return false, nil
	}

	return true, nil

}

func (cluster *cluster) GetSecretsAndServices(namespace string) (map[string][]map[string]interface{}, error) {
	secrets, err := cluster.ListSecrets(namespace, meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	secretsMap := []map[string]interface{}{}
	for _, secret := range secrets.Items {
		if secret.Type == api_v1.SecretTypeOpaque {
			credentialSecrets := map[string]string{}
			for key, val := range secret.Data {
				credentialSecrets[key] = string(val)
			}
			credential := map[string]interface{}{
				"name": secret.Name,
				"data": credentialSecrets,
			}
			secretsMap = append(secretsMap, credential)
		}
	}

	services, err := cluster.ListServices(namespace, meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	servicesMap := []map[string]interface{}{}
	for _, service := range services.Items {
		if service.Spec.Type == "NodePort" {
			nodes, _ := cluster.ListNodes(meta_v1.ListOptions{})
			for _, node := range nodes.Items {
				service.Spec.ExternalIPs = append(service.Spec.ExternalIPs, node.ObjectMeta.Labels["spec.ip"])
			}
		}
		credentialService := map[string]interface{}{
			"name":     service.ObjectMeta.Name,
			"metadata": service.ObjectMeta,
			"spec":     service.Spec,
			"status":   service.Status,
		}
		servicesMap = append(servicesMap, credentialService)
	}

	servicesAndSecrets := map[string][]map[string]interface{}{
		"secrets":  secretsMap,
		"services": servicesMap,
	}

	return servicesAndSecrets, nil

}

func (cluster *cluster) GetIngress(namespace string) ([]map[string]interface{}, error) {
	ingress, err := cluster.ListIngress(namespace, meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var ingressMap []map[string]interface{}
	for _, service := range ingress.Items {
		credentialService := map[string]interface{}{
			"name":     service.ObjectMeta.Name,
			"metadata": service.ObjectMeta,
			"spec":     service.Spec,
			"status":   service.Status,
		}
		ingressMap = append(ingressMap, credentialService)
	}
	return ingressMap, nil
}

func (cluster *cluster) SecretExists(namespaceName string, secretName string) (bool, error) {
	_, err := cluster.GetSecret(namespaceName, secretName, meta_v1.GetOptions{})
	if err != nil {
		statusError, ok := err.(*errors2.StatusError)
		if !ok {
			return false, err
		}
		if statusError.ErrStatus.Reason != meta_v1.StatusReasonNotFound {
			return false, err
		}

		return false, nil
	} else {
		return true, nil
	}
}

func (cluster *cluster) CreateOrUpdateSecret(namespaceName string, secret *api_v1.Secret) (*api_v1.Secret, error) {
	exists, err := cluster.SecretExists(namespaceName, secret.Name)
	if err != nil {
		return nil, err
	}

	if exists {
		return cluster.UpdateSecret(namespaceName, secret)
	} else {
		return cluster.CreateSecret(namespaceName, secret)
	}
}

func (cluster *clusterDelegate) GetClientConfig() *rest.Config {
	return cluster.k8sConfig
}

func buildClientConfig(credentials *config.ClusterCredentials) (*rest.Config, error) {
	tlsClientConfig := rest.TLSClientConfig{
		CAData: credentials.CAData,
	}

	return &rest.Config{
		Host:            credentials.Server,
		BearerToken:     credentials.Token,
		TLSClientConfig: tlsClientConfig,
	}, nil
}

func (cluster *clusterDelegate) GetClient() kubernetes.Interface {
	return cluster.client
}

func (cluster *clusterDelegate) CreateNamespace(namespace *api_v1.Namespace) (*api_v1.Namespace, error) {
	return cluster.GetClient().CoreV1().Namespaces().Create(namespace)
}

func (cluster *clusterDelegate) DeleteNamespace(name string, options *meta_v1.DeleteOptions) error {
	return cluster.GetClient().CoreV1().Namespaces().Delete(name, options)
}

func (cluster *clusterDelegate) GetNamespace(name string, options *meta_v1.GetOptions) (*api_v1.Namespace, error) {
	if options == nil {
		return cluster.GetClient().CoreV1().Namespaces().Get(name, meta_v1.GetOptions{})
	} else {
		return cluster.GetClient().CoreV1().Namespaces().Get(name, *options)
	}
}

func (cluster *clusterDelegate) ListPods(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.PodList, error) {
	pods, err := cluster.client.CoreV1().Pods(nameSpace).List(listOptions)
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func (cluster *clusterDelegate) GetDeployment(nameSpace string, deploymentName string, getOptions meta_v1.GetOptions) (*v1_beta1.Deployment, error) {
	return cluster.GetClient().ExtensionsV1beta1().Deployments(nameSpace).Get(deploymentName, meta_v1.GetOptions{})
}

func (cluster *clusterDelegate) ListNodes(listOptions meta_v1.ListOptions) (*api_v1.NodeList, error) {
	return cluster.GetClient().CoreV1().Nodes().List(listOptions)
}

func (cluster *clusterDelegate) ListServiceAccounts(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.ServiceAccountList, error) {
	return cluster.GetClient().CoreV1().ServiceAccounts(nameSpace).List(listOptions)
}

func (cluster *clusterDelegate) CreateServiceAccount(nameSpace string, serviceAccount *api_v1.ServiceAccount) (*api_v1.ServiceAccount, error) {
	return cluster.GetClient().CoreV1().ServiceAccounts(nameSpace).Create(serviceAccount)
}

func (cluster *clusterDelegate) ListClusterRoleBindings(listOptions meta_v1.ListOptions) (*rbacv1beta1.ClusterRoleBindingList, error) {
	return cluster.GetClient().RbacV1beta1().ClusterRoleBindings().List(listOptions)
}

func (cluster *clusterDelegate) CreateClusterRoleBinding(clusterRoleBinding *rbacv1beta1.ClusterRoleBinding) (*rbacv1beta1.ClusterRoleBinding, error) {
	return cluster.GetClient().RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding)
}

func (cluster *clusterDelegate) CreateSecret(nameSpace string, secret *api_v1.Secret) (*api_v1.Secret, error) {
	return cluster.GetClient().CoreV1().Secrets(nameSpace).Create(secret)
}

func (cluster *clusterDelegate) UpdateSecret(nameSpace string, secret *api_v1.Secret) (*api_v1.Secret, error) {
	return cluster.GetClient().CoreV1().Secrets(nameSpace).Update(secret)
}

func (cluster *clusterDelegate) GetSecret(nameSpace string, name string, getOptions meta_v1.GetOptions) (*api_v1.Secret, error) {
	return cluster.GetClient().CoreV1().Secrets(nameSpace).Get(name, getOptions)
}

func (cluster *clusterDelegate) ListSecrets(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.SecretList, error) {
	return cluster.GetClient().CoreV1().Secrets(nameSpace).List(listOptions)
}

func (cluster *clusterDelegate) ListServices(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.ServiceList, error) {
	return cluster.GetClient().CoreV1().Services(nameSpace).List(listOptions)
}

func (cluster *clusterDelegate) Patch(nameSpace string, name string, pt types.PatchType, data []byte, subresources ...string) (result *api_v1.ServiceAccount, err error) {
	return cluster.GetClient().CoreV1().ServiceAccounts(nameSpace).Patch(name, pt, data, subresources...)
}

func (cluster *clusterDelegate) ListPersistentVolumes(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.PersistentVolumeClaimList, error) {
	list, err := cluster.GetClient().CoreV1().PersistentVolumeClaims(nameSpace).List(listOptions)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (cluster *clusterDelegate) GetNamespaces() (*api_v1.NamespaceList, error) {
	namespaceList, err := cluster.GetClient().CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return namespaceList, nil
}

func (cluster *clusterDelegate) ListDeployments(nameSpace string, listOptions meta_v1.ListOptions) (*DeploymentList, error) {
	list, err := cluster.GetClient().AppsV1().Deployments(nameSpace).List(listOptions)
	if err != nil {
		return nil, err
	}

	deployments := DeploymentList{}

	for _, item := range list.Items {
		// Find RS associated with deployment
		newReplicaSet, err := deploymentutil.GetNewReplicaSet(&item, cluster.GetClient().AppsV1())
		if err != nil || newReplicaSet == nil {
			return nil, err
		}
		newDeployment := Deployment{
			newReplicaSet,
			&item,
		}
		deployments.Items = append(deployments.Items, newDeployment)
	}
	return &deployments, nil
}

func (cluster *clusterDelegate) ListIngress(nameSpace string, listOptions meta_v1.ListOptions) (*v1_beta1.IngressList, error) {
	list, err := cluster.GetClient().ExtensionsV1beta1().Ingresses(nameSpace).List(listOptions)
	if err != nil {
		return nil, err
	}
	return list, nil
}
