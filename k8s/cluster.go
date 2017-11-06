package k8s

import (
	"github.com/cf-platform-eng/kibosh/config"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Cluster interface {
	//todo: ListPods is just to validate that API working, delete as appropriate
	ListPods() (*v1.PodList, error)
}

type cluster struct {
	kuboConfig *config.KuboODBVCAP
	clientSet  *kubernetes.Clientset
}

func NewCluster(kuboConfig *config.KuboODBVCAP) (Cluster, error) {
	newK8s := cluster{
		kuboConfig: kuboConfig,
	}

	var err error
	newK8s.clientSet, err = buildClientConfig(kuboConfig)
	if err != nil {
		return nil, err
	}

	return &newK8s, nil
}

func buildClientConfig(kuboConfig *config.KuboODBVCAP) (*kubernetes.Clientset, error) {
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

func (cluster cluster) ListPods() (*v1.PodList, error) {
	//todo: ListPods is just to validate that API working, delete as appropriate
	pods, err := cluster.clientSet.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pods, nil
}
