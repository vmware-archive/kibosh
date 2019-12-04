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

package helm

import (
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helmstaller "k8s.io/helm/cmd/helm/installer"
)

type installer struct {
	maxWait time.Duration
	config  *config.Config
	cluster k8s.Cluster
	client  MyHelmClient
	logger  *logrus.Logger
}

//go:generate counterfeiter ./ Installer
type Installer interface {
	Install() error
	SetMaxWait(duration time.Duration)
}

const (
	TillerTag string = "v2.16.1"
)

const (
	deploymentName = "tiller-deploy"
)

type InstallerFactory func(c *config.Config, cluster k8s.Cluster, client MyHelmClient, logger *logrus.Logger) Installer

func InstallerFactoryDefault(c *config.Config, cluster k8s.Cluster, client MyHelmClient, logger *logrus.Logger) Installer {
	return NewInstaller(c, cluster, client, logger)
}

func NewInstaller(c *config.Config, cluster k8s.Cluster, client MyHelmClient, logger *logrus.Logger) Installer {
	return &installer{
		maxWait: 60 * time.Second,
		config:  c,
		cluster: cluster,
		client:  client,
		logger:  logger,
	}
}

func (i *installer) Install() error {
	i.logger.Debug(fmt.Sprintf("Installing helm with Tiller version %s", TillerTag))

	tillerImage := "gcr.io/kubernetes-helm/tiller:" + TillerTag
	if i.config.RegistryConfig.HasRegistryConfig() {
		privateRegistrySetup := k8s.NewPrivateRegistrySetup(i.config.TillerNamespace, k8s.ServiceAccountName, i.cluster, i.config.RegistryConfig)
		err := privateRegistrySetup.Setup()
		if err != nil {
			return err
		}

		tillerImage = fmt.Sprintf("%s/tiller:%s", i.config.RegistryConfig.Server, TillerTag)
		if i.config.TillerSHA != "" {
			tillerImage = fmt.Sprintf("%s@sha256:%s", tillerImage, i.config.TillerSHA)
		}
	}

	var err error
	if i.config.HelmTLSConfig.HasTillerTLS() {
		i.logger.Debug("Installing with TLS")
		err = i.installWithTLS(tillerImage)
	} else {
		i.logger.Debug("Installing insecure")
		err = i.installInsecure(tillerImage)
	}
	if err != nil {
		return err
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

	if err != nil {
		i.logger.Debug(fmt.Sprintf(
			"Error checking helm healthy. Not necessarily an 'error' Error: %s", err.Error()),
		)
	}

	return err == nil
}

func (i *installer) isNewerVersion(existingImage string, newImage string) bool {
	existingVersionSplit := strings.Split(existingImage, ":")
	if len(existingVersionSplit) < 2 {
		return true
	}
	existingVersion := strings.Split(existingVersionSplit[1], "@")[0]

	newVersionSplit := strings.Split(newImage, ":")
	if len(newVersionSplit) < 2 {
		return true
	}
	newVersion := strings.Split(newVersionSplit[1], "@")[0]

	return semver.MustParse(newVersion).GreaterThan(semver.MustParse(existingVersion))
}

func (i *installer) installWithTLS(tillerImage string) error {
	if i.client.HasDifferentTLSConfig() {
		i.logger.Debug("Uninstalling to remove existing TLS")
		err := i.client.Uninstall(&helmstaller.Options{
			Namespace: i.config.TillerNamespace,
		})
		if err != nil {
			return errors.Wrap(err, "Error uninstalling previous helm")
		}
		//todo: wait for deletion!?
	}

	options := helmstaller.Options{
		Namespace:                    i.config.TillerNamespace,
		ImageSpec:                    tillerImage,
		ServiceAccount:               k8s.ServiceAccountName,
		AutoMountServiceAccountToken: true,
		VerifyTLS:                    true,
		TLSCertFile:                  i.config.HelmTLSConfig.TillerTLSCertFile,
		TLSKeyFile:                   i.config.HelmTLSConfig.TillerTLSKeyFile,
		TLSCaCertFile:                i.config.HelmTLSConfig.TLSCaCertFile,
	}
	return i.doInstall(tillerImage, options)
}

func (i *installer) installInsecure(tillerImage string) error {
	options := helmstaller.Options{
		Namespace:                    i.config.TillerNamespace,
		ImageSpec:                    tillerImage,
		ServiceAccount:               k8s.ServiceAccountName,
		AutoMountServiceAccountToken: true,
	}

	return i.doInstall(tillerImage, options)
}

func (i *installer) doInstall(tillerImage string, options helmstaller.Options) error {
	err := i.client.Install(&options)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "Error installing new helm")
		}

		obj, err := i.cluster.GetDeployment(i.config.TillerNamespace, deploymentName, meta_v1.GetOptions{})
		if err != nil {
			return err
		}
		existingImage := obj.Spec.Template.Spec.Containers[0].Image
		if existingImage == tillerImage {
			return nil
		}
		if !i.isNewerVersion(existingImage, tillerImage) {
			return nil
		}
		err = i.client.Upgrade(&options)
		if err != nil {
			return errors.Wrap(err, "Error upgrading helm")
		}
	}

	return nil
}
