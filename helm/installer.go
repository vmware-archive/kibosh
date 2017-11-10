package helm

import (
	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/k8s"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	helmstaller "k8s.io/helm/cmd/helm/installer"
	"time"
	"k8s.io/helm/pkg/helm"
)

type installer struct {
	cluster k8s.Cluster
	client helm.Interface
	logger  lager.Logger
}

type Installer interface {
	Install() error
}

//todo: the image needs to somehow be increment-able + local, deferring to packaging stories
const (
	nameSpace = "kube-system"
	image     = "gcr.io/kubernetes-helm/tiller:v2.6.1"
)

func NewInstaller(cluster k8s.Cluster, client helm.Interface, logger lager.Logger) Installer {
	return &installer{
		cluster: cluster,
		client: client,
		logger:  logger,
	}
}

func (i installer) Install() error {
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
		if i.helmHealthy() {
			break
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

func (i installer) helmHealthy() bool {
	_, err := i.client.ListReleases()

	return err == nil
}
