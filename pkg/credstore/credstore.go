package credstore

import (
	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/auth"
	"github.com/sirupsen/logrus"
)

//go:generate counterfeiter ./ CredStore
type CredStore interface {
	//store the key/value, return the configured placeholder block?
	Put(key string, credentials interface{}) (interface{}, error)
	Get(key string) (interface{}, error)
	Delete(key string) error
}

type credhubStore struct {
	credHubClient *credhub.CredHub
	logger        *logrus.Logger
}

func NewCredhubStore(credHubURL, uaaURL, uaaClientName, uaaClientSecret string, skipSSLValidation bool, logger *logrus.Logger) (CredStore, error) {
	ch, err := credhub.New(credHubURL,
		credhub.SkipTLSValidation(skipSSLValidation),
		credhub.Auth(auth.UaaClientCredentials(uaaClientName, uaaClientSecret)),
		credhub.AuthURL(uaaURL),
	)
	if err != nil {
		return nil, err
	}

	return &credhubStore{
		credHubClient: ch,
		logger:        logger,
	}, err
}

func (c *credhubStore) Put(key string, credentials interface{}) (interface{}, error) {
	return c.credHubClient.SetCredential(key, "json", credentials)
}

func (c *credhubStore) Get(key string) (interface{}, error) {
	return c.credHubClient.GetLatestValue(key)
}

func (c *credhubStore) Delete(key string) error {
	return c.credHubClient.Delete(key)
}
