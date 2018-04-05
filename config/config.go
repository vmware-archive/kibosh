package config

import (
	"github.com/kelseyhightower/envconfig"

	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

type ClusterCredentials struct {
	CAData string `envconfig:"CA_DATA"`
	Server string `envconfig:"SERVER"`
	Token  string `envconfig:"TOKEN"`
}

type RegistryConfig struct {
	Server string `envconfig:"REG_SERVER"`
	User   string `envconfig:"REG_USER"`
	Pass   string `envconfig:"REG_PASS"`
	Email  string `envconfig:"REG_EMAIL"`
}

type config struct {
	AdminUsername string `envconfig:"SECURITY_USER_NAME" required:"true"`
	AdminPassword string `envconfig:"SECURITY_USER_PASSWORD" required:"true"`
	ServiceID     string `envconfig:"SERVICE_ID" required:"true"`
	ServiceName   string `envconfig:"SERVICE_NAME" required:"true"`

	Port         int    `envconfig:"PORT" default:"8080"`
	HelmChartDir string `envconfig:"HELM_CHART_DIR" default:"charts"`

	ClusterCredentials *ClusterCredentials
	RegistryConfig     *RegistryConfig
}

func (r RegistryConfig) HasRegistryConfig() bool {
	return r.Server != ""
}

func (r RegistryConfig) GetDockerConfigJson() ([]byte, error) {
	if r.Server == "" || r.Email == "" || r.Pass == "" || r.User == "" {
		return nil, errors.New("environment didn't have a proper registry config")
	}

	dockerConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			r.Server: map[string]interface{}{
				"username": r.User,
				"password": r.Pass,
				"email":    r.Email,
			},
		},
	}
	dockerConfigJson, err := json.Marshal(dockerConfig)
	if err != nil {
		return nil, err
	}

	return dockerConfigJson, nil
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

	match, err := regexp.MatchString(`^[0-9a-z.\-]+$`, c.ServiceName)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, errors.New(fmt.Sprintf("Name [%s] contains invalid characters", c.ServiceName))
	}

	return c, nil
}
