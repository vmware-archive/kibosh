package helm

import (
	"code.cloudfoundry.org/lager"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	helmstaller "k8s.io/helm/cmd/helm/installer"
)

type installer struct {
	client kubernetes.Interface
	logger lager.Logger
}

type Installer interface {
	Install() error
}

func NewInstaller(client kubernetes.Interface, logger lager.Logger) Installer {
	return &installer{
		client: client,
		logger: logger,
	}
}

func (i installer) Install() error {
	options := helmstaller.Options{

	}
	i.logger.Debug("Installing helm")
	err := helmstaller.Install(i.client, &options)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			i.logger.Debug("Already exists, updating")
			return errors.Wrap(err, "Error installing new helm")
		}
		err := helmstaller.Upgrade(i.client, &options)
		if err != nil {
			return errors.Wrap(err, "Error upgrading helm")
		}
	}
	return nil
}
