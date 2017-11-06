package config

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"os"
)

type KuboODBVCAP struct {
	Name  string `json:"name"`
	Label string `json:"label"`

	Credentials Credentials `json:"credentials"`
}

type Credentials struct {
	KubeConfig KubeConfig `json:"kubeconfig"`
}

type KubeConfig struct {
	ApiVersion string    `json:"apiVersion"`
	Clusters   []Cluster `json:"clusters"`
	Users      []User    `json:"users"`
}

type Cluster struct {
	Name        string      `json:"name"`
	ClusterInfo ClusterInfo `json:"cluster"`
}

type ClusterInfo struct {
	Server string `json:"server"`
	CAData string `json:"certificate-authority-data"`
}

type User struct {
	Name            string          `json:"name"`
	UserCredentials UserCredentials `json:"user"`
}

type UserCredentials struct {
	Token string `json:"token"`
}

func ParseVCAPServices(service string) (*KuboODBVCAP, error) {
	services := os.Getenv("VCAP_SERVICES")
	vcapServicesRaw := map[string]interface{}{}
	err := json.Unmarshal([]byte(services), &vcapServicesRaw)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to parse VCAP_SERVICES json")
	}

	kuboODBServices := vcapServicesRaw[service]
	kuboODBServiceWeak := (kuboODBServices.([]interface{}))[0]

	reEncoded, err := json.Marshal(kuboODBServiceWeak)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to re-marshal service json")
	}

	kuboODBService := KuboODBVCAP{}
	err = json.Unmarshal(reEncoded, &kuboODBService)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to unmarshal re-marshaled service json")
	}

	return &kuboODBService, nil
}

func (cluster ClusterInfo) DecodeCAData() ([]byte, error) {
	caData, err := base64.StdEncoding.DecodeString(cluster.CAData)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to decode CA Data")

	}
	return caData, nil
}
