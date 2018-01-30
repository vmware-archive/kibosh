package helm

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/k8s"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	helmstaller "k8s.io/helm/cmd/helm/installer"
)

type installer struct {
	maxWait time.Duration
	cluster k8s.Cluster
	client  MyHelmClient
	logger  lager.Logger
}

type Installer interface {
	Install() error
	SetMaxWait(duration time.Duration)
}

//todo: the image needs to somehow be increment-able + local, deferring to packaging stories
const (
	nameSpace = "kube-system"
	image     = "gcr.io/kubernetes-helm/tiller:v2.8.0"
)

func NewInstaller(cluster k8s.Cluster, client MyHelmClient, logger lager.Logger) Installer {
	return &installer{
		maxWait: 60 * time.Second,
		cluster: cluster,
		client:  client,
		logger:  logger,
	}
}

func (i *installer) Install() error {
	options := helmstaller.Options{
		Namespace:      nameSpace,
		ImageSpec:      image,
		ServiceAccount: "tiller",
	}
	i.logger.Debug("Installing helm")
	err := i.client.Install(&options)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			i.logger.Debug("Already exists, updating")
			return errors.Wrap(err, "Error installing new helm")
		}
		err := i.client.Upgrade(&options)
		if err != nil {
			return errors.Wrap(err, "Error upgrading helm")
		}
	}

	i.logger.Info("Waiting for tiller to become healthy")
	waited := time.Duration(0)
	for {
		if i.helmHealthy() {
			break
		}
		if waited >= i.maxWait {
			return errors.New("Didn't become healthy within max time")
		}
		willWait := i.maxWait / 10
		waited = waited + willWait
		time.Sleep(willWait)
	}
	return nil
}

func (i *installer) SetMaxWait(wait time.Duration) {
	i.maxWait = wait
}

func (i *installer) helmHealthy() bool {
	_, err := i.client.ListReleases()
	//	if err != nil {
	//		println("-----------------------")
	//		println(err.Error())
	//		println("-----------------------")
	//	} else {
	//		println("-----------------------")
	//		println("Shit finally worked")
	//	}
	//
	return err == nil
}
