package config

import (
	"github.com/kelseyhightower/envconfig"
)

type ClusterCredentials struct {
	CAData string `envconfig:"CA_DATA"`
	Server string `envconfig:"SERVER"`
	Token  string `envconfig:"TOKEN"`
}

type config struct {
	AdminUsername string `envconfig:"SECURITY_USER_NAME" required:"true"`
	AdminPassword string `envconfig:"SECURITY_USER_PASSWORD" required:"true"`
	Port          int    `envconfig:"port" default:"8080"`
	HelmChartDir  string `envconfig:"HELM_CHART_DIR" default:"charts"`
	ServiceID     string `envconfig:"SERVICE_ID" required:"true"`

	ClusterCredentials *ClusterCredentials
}

func Parse() (*config, error) {
	c := &config{}
	err := envconfig.Process("", c)
	if err != nil {
		return nil, err
	}

	clusterCredentials := &ClusterCredentials{}
	err = envconfig.Process("", clusterCredentials)
	if err != nil {
		return nil, err
	}
	c.ClusterCredentials = clusterCredentials

	return c, nil
}
