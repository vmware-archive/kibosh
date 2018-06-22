package bazaar

import (
	"github.com/cf-platform-eng/kibosh/config"
	"github.com/kelseyhightower/envconfig"
)

type bazaarConfig struct {
	AdminUsername string `envconfig:"SECURITY_USER_NAME" required:"true"`
	AdminPassword string `envconfig:"SECURITY_USER_PASSWORD" required:"true"`

	Port         int    `envconfig:"PORT" default:"8081"`
	HelmChartDir string `envconfig:"HELM_CHART_DIR" default:"charts"`

	RegistryConfig *config.RegistryConfig
}

func ParseConfig() (*bazaarConfig, error) {
	c := &bazaarConfig{}
	err := envconfig.Process("", c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
