package k8s

import (
	"github.com/cf-platform-eng/kibosh/config"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	api_v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

type Cluster interface {
	//todo: ListPods is just to validate that API working, delete as appropriate
	ListPods() (*api_v1.PodList, error)
	GetClient() kubernetes.Interface
}

type cluster struct {
	kuboConfig *config.KuboODBVCAP
	client     kubernetes.Interface
}

func NewCluster(kuboConfig *config.KuboODBVCAP) (Cluster, error) {
	newK8s := cluster{
		kuboConfig: kuboConfig,
	}

	var err error
	newK8s.client, err = buildClientConfig(kuboConfig)
	if err != nil {
		return nil, err
	}

	return &newK8s, nil
}

func buildClientConfig(kuboConfig *config.KuboODBVCAP) (kubernetes.Interface, error) {
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
	k8sConfig := &rest.Config{
		Host:            server,
		BearerToken:     token,
		TLSClientConfig: tlsClientConfig,
	}

	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	return clientSet, err
}

func (cluster cluster) GetClient() kubernetes.Interface {
	return cluster.client
}

func (cluster cluster) ListPods() (*api_v1.PodList, error) {
	//todo: ListPods is just to validate that API working, delete as appropriate
	pods, err := cluster.client.CoreV1().Pods("").List(meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pods, nil
}
