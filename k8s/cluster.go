package k8s

import (
	"github.com/cf-platform-eng/kibosh/config"

	api_v1 "k8s.io/api/core/v1"
	v1_beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//go:generate counterfeiter ./ Cluster
type Cluster interface {
	GetClient() kubernetes.Interface
	GetClientConfig() *rest.Config

	CreateNamespace(*api_v1.Namespace) (*api_v1.Namespace, error)
	ListPods() (*api_v1.PodList, error)
	GetDeployment(string, string, meta_v1.GetOptions) (*v1_beta1.Deployment, error)
	ListServiceAccounts(string, meta_v1.ListOptions) (*api_v1.ServiceAccountList, error)
	CreateServiceAccount(string, *api_v1.ServiceAccount) (*api_v1.ServiceAccount, error)
	ListClusterRoleBindings(meta_v1.ListOptions) (*rbacv1beta1.ClusterRoleBindingList, error)
	CreateClusterRoleBinding(*rbacv1beta1.ClusterRoleBinding) (*rbacv1beta1.ClusterRoleBinding, error)
	ListSecrets(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.SecretList, error)
	ListServices(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.ServiceList, error)
}

type cluster struct {
	kuboConfig *config.KuboODBVCAP
	client     kubernetes.Interface
	k8sConfig  *rest.Config
}

func NewCluster(kuboConfig *config.KuboODBVCAP) (Cluster, error) {
	k8sConfig, err := buildClientConfig(kuboConfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	return &cluster{
		kuboConfig: kuboConfig,
		k8sConfig:  k8sConfig,
		client:     client,
	}, nil
}

func (cluster *cluster) GetClientConfig() *rest.Config {
	return cluster.k8sConfig
}

func buildClientConfig(kuboConfig *config.KuboODBVCAP) (*rest.Config, error) {
	user := kuboConfig.Credentials.KubeConfig.Users[0]
	cluster := kuboConfig.Credentials.KubeConfig.Clusters[0]

	token := user.UserCredentials.Token
	server := cluster.ClusterInfo.Server
	caData, err := cluster.ClusterInfo.DecodeCAData()
	if err != nil {
		return nil, err
	}

	tlsClientConfig := rest.TLSClientConfig{
		CAData: caData,
	}

	return &rest.Config{
		Host:            server,
		BearerToken:     token,
		TLSClientConfig: tlsClientConfig,
	}, nil
}

func (cluster *cluster) GetClient() kubernetes.Interface {
	return cluster.client
}

func (cluster *cluster) CreateNamespace(namespace *api_v1.Namespace) (*api_v1.Namespace, error) {
	return cluster.GetClient().CoreV1().Namespaces().Create(namespace)
}

func (cluster *cluster) ListPods() (*api_v1.PodList, error) {
	//todo: ListPods is just to validate that API working, delete as appropriate
	pods, err := cluster.client.CoreV1().Pods("").List(meta_v1.ListOptions{})
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

func (cluster *cluster) ListSecrets(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.SecretList, error) {
	return cluster.GetClient().CoreV1().Secrets(nameSpace).List(listOptions)
}

func (cluster *cluster) ListServices(nameSpace string, listOptions meta_v1.ListOptions) (*api_v1.ServiceList, error) {
	return cluster.GetClient().CoreV1().Services(nameSpace).List(listOptions)
}
