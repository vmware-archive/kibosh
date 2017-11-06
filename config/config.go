package config

import (
	"github.com/kelseyhightower/envconfig"
)

type config struct {
	AdminUsername string `envconfig:"SECURITY_USER_NAME" required:"true"`
	AdminPassword string `envconfig:"SECURITY_USER_PASSWORD" required:"true"`
	Port          int    `envconfig:"port" default:"8080"`
	HelmChartDir  string `envconfig:"HELM_CHART_DIR" default:"helm"`
	ServiceID     string `envconfig:"SERVICE_ID" required:"true"`

	KuboODBVCAP *KuboODBVCAP
}

func Parse() (*config, error) {
	c := &config{}
	err := envconfig.Process("", c)
	if err != nil {
		return nil, err
	}

	c.KuboODBVCAP, err = ParseVCAPServices("kubo-odb")
	if err != nil {
		return nil, err
	}

	return c, nil
}
