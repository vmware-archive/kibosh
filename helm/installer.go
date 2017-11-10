package helm

import (
	"code.cloudfoundry.org/lager"
	"fmt"
	"github.com/cf-platform-eng/kibosh/k8s"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	helmstaller "k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/helm"
	"time"
)

type installer struct {
	cluster k8s.Cluster
	logger  lager.Logger
}

type Installer interface {
	Install() error
	HelmHealthy() bool
}

//todo: the image needs to somehow be increment-able + local, deferring to packaging stories
const (
	nameSpace = "kube-system"
	image     = "gcr.io/kubernetes-helm/tiller:v2.6.1"
)

func NewInstaller(cluster k8s.Cluster, logger lager.Logger) Installer {
	return &installer{
		cluster: cluster,
		logger:  logger,
	}
}

func (i installer) Install() error {
	//stableRepositoryURL: = "https://kubernetes-charts.storage.googleapis.com"
	options := helmstaller.Options{
		Namespace: nameSpace,
		ImageSpec: image,
	}
	i.logger.Debug("Installing helm")
	err := helmstaller.Install(i.cluster.GetClient(), &options)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			i.logger.Debug("Already exists, updating")
			return errors.Wrap(err, "Error installing new helm")
		}
		err := helmstaller.Upgrade(i.cluster.GetClient(), &options)
		if err != nil {
			return errors.Wrap(err, "Error upgrading helm")
		}
	}

	i.logger.Info("Waiting for tiller to become healthy")
	for {
		if i.HelmHealthy() {
			break
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

func (i installer) HelmHealthy() bool {
	tunnel, err := i.setupConnection()
	if err != nil {
		i.logger.Error("Error establishing tunnel", err)
		return false
	}
	defer tunnel.Close()

	host := fmt.Sprintf("127.0.0.1:%d", tunnel.Local)
	i.logger.Debug("Tunnel", lager.Data{"host": host})

	helmClient := helm.NewClient(helm.Host(host))
	_, err = helmClient.ListReleases()
	return err == nil
}

func (i installer) setupConnection() (*Tunnel, error) {

	config, client := i.cluster.GetClientConfig(), i.cluster.GetClient()

	return New(nameSpace, client, config)
}
