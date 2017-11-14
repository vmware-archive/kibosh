package k8s

import (
	"github.com/cf-platform-eng/kibosh/config"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	api_v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

//go:generate counterfeiter ./ Cluster
type Cluster interface {
	GetClient() kubernetes.Interface
	GetClientConfig() *rest.Config

	CreateNamespace(*api_v1.Namespace) (*api_v1.Namespace, error)
	ListPods() (*api_v1.PodList, error)
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
