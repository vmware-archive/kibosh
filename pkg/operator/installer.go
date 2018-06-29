package operator

import (
	"code.cloudfoundry.org/lager"
	"fmt"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

type PksOperator struct {
	Logger         lager.Logger
	registryConfig *config.RegistryConfig

	cluster      k8s.Cluster
	myHelmClient my_helm.MyHelmClient
	operatorsMap map[string]*my_helm.MyChart
}

func NewInstaller(registryConfig *config.RegistryConfig, cluster k8s.Cluster, myHelmClient my_helm.MyHelmClient, logger lager.Logger) *PksOperator {
	operator := &PksOperator{
		Logger:         logger,
		registryConfig: registryConfig,

		cluster:      cluster,
		myHelmClient: myHelmClient,
	}

	return operator
}

func (operator *PksOperator) InstallCharts(operatorCharts []*my_helm.MyChart) {
	for _, operatorChart := range operatorCharts {
		operator.Install(operatorChart)
	}
}

func (operator *PksOperator) Install(chart *my_helm.MyChart) error {

	operator.Logger.Info(fmt.Sprintf("operator to install " + chart.Chartpath))

	namespaceName := chart.String() + "-kibosh-operator"

	ns, err := operator.cluster.GetNamespace(namespaceName, &meta_v1.GetOptions{})
	if err != nil {
		return err
	}
	if ns == nil {
		namespace := api_v1.Namespace{
			Spec: api_v1.NamespaceSpec{},
			ObjectMeta: meta_v1.ObjectMeta{
				Name: namespaceName,
				Labels: map[string]string{
					"kibosh": "installed",
				},
			},
		}
		_, err := operator.cluster.CreateNamespace(&namespace)
		if err != nil {
			return err
		}
	}

	if operator.registryConfig.HasRegistryConfig() {
		privateRegistrySetup := k8s.NewPrivateRegistrySetup(namespaceName, "default", operator.cluster, operator.registryConfig)
		err := privateRegistrySetup.Setup()
		if err != nil {
			return err
		}
	}

	releases, err := operator.myHelmClient.ListReleases()
	if err != nil {
		return err
	}
	if releases != nil && exists(namespaceName, releases) {
		return nil
	}

	_, err = operator.myHelmClient.InstallOperator(chart, namespaceName)
	if err != nil {
		return err
	}

	return nil
}

func exists(name string, releases *rls.ListReleasesResponse) bool {
	for _, release := range releases.Releases {
		if release.Name == name {
			return true
		}
	}
	return false
}
