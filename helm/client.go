package helm

import (
	"fmt"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/k8s"
	helmstaller "k8s.io/helm/cmd/helm/installer"
	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

type myHelmClient struct {
	privateRegistryServer string
	cluster               k8s.Cluster
	logger                lager.Logger
	chart                 *MyChart
}

//- go:generate counterfeiter ./ MyHelmClient
//^ counterfeiter is generating bad stubs interface. If needing to regenerate, fix above line & then re-fix stubs
type MyHelmClient interface {
	helm.Interface
	Install(*helmstaller.Options) error
	Upgrade(*helmstaller.Options) error
	InstallChart(namespace string, planName string, options ...helm.InstallOption) (*rls.InstallReleaseResponse, error)
	MergeValueBytes(base []byte, override []byte) ([]byte, error)
}

func NewMyHelmClient(chart *MyChart, cluster k8s.Cluster, logger lager.Logger) MyHelmClient {
	return &myHelmClient{
		chart:   chart,
		cluster: cluster,
		logger:  logger,
	}
}

func (c myHelmClient) open() (*kube.Tunnel, helm.Interface, error) {
	config, client := c.cluster.GetClientConfig(), c.cluster.GetClient()
	tunnel, err := portforwarder.New(nameSpace, client, config)
	if err != nil {
		return nil, nil, err
	}

	host := fmt.Sprintf("127.0.0.1:%d", tunnel.Local)
	c.logger.Debug("Tunnel", lager.Data{"host": host})

	return tunnel, helm.NewClient(helm.Host(host)), nil
}

func (c *myHelmClient) Install(opts *helmstaller.Options) error {
	return helmstaller.Install(c.cluster.GetClient(), opts)
}

func (c *myHelmClient) Upgrade(opts *helmstaller.Options) error {
	return helmstaller.Upgrade(c.cluster.GetClient(), opts)
}

func (c myHelmClient) ListReleases(opts ...helm.ReleaseListOption) (*rls.ListReleasesResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()
	return client.ListReleases(opts...)
}

func (c myHelmClient) InstallRelease(chStr, namespace string, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	panic("Not yet implemented")
}

func (c myHelmClient) InstallReleaseFromChart(myChart *chart.Chart, namespace string, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.InstallReleaseFromChart(myChart, namespace, opts...)
}

func (c myHelmClient) InstallChart(namespace string, planName string, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	overrideValues, err := c.MergeValueBytes(c.chart.Values, c.chart.Plans[planName].Values)
	if err != nil {
		return nil, err
	}
	newOpts := append(opts, helm.ValueOverrides(overrideValues))
	return c.InstallReleaseFromChart(c.chart.Chart, namespace, newOpts...)
}

func (c myHelmClient) DeleteRelease(rlsName string, opts ...helm.DeleteOption) (*rls.UninstallReleaseResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.DeleteRelease(rlsName, opts...)
}

func (c myHelmClient) ReleaseStatus(rlsName string, opts ...helm.StatusOption) (*rls.GetReleaseStatusResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.ReleaseStatus(rlsName, opts...)
}

func (c myHelmClient) UpdateRelease(rlsName, chStr string, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
	panic("Not yet implemented")
}

func (c myHelmClient) UpdateReleaseFromChart(rlsName string, chart *chart.Chart, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
	panic("Not yet implemented")
}

func (c myHelmClient) RollbackRelease(rlsName string, opts ...helm.RollbackOption) (*rls.RollbackReleaseResponse, error) {
	panic("Not yet implemented")
}

func (c myHelmClient) ReleaseContent(rlsName string, opts ...helm.ContentOption) (*rls.GetReleaseContentResponse, error) {
	panic("Not yet implemented")
}

func (c myHelmClient) ReleaseHistory(rlsName string, opts ...helm.HistoryOption) (*rls.GetHistoryResponse, error) {
	panic("Not yet implemented")
}

func (c myHelmClient) GetVersion(opts ...helm.VersionOption) (*rls.GetVersionResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.GetVersion(opts...)
}

func (c myHelmClient) RunReleaseTest(rlsName string, opts ...helm.ReleaseTestOption) (<-chan *rls.TestReleaseResponse, <-chan error) {
	panic("Not yet implemented")
}

func (c myHelmClient) PingTiller() error {
	panic("Not yet implemented")
}

func (c myHelmClient) MergeValueBytes(base []byte, override []byte) ([]byte, error) {
	baseVals := map[string]interface{}{}
	err := yaml.Unmarshal(base, &baseVals)
	if err != nil {
		panic(err)
	}
	overrideVals := map[string]interface{}{}
	err = yaml.Unmarshal(override, &overrideVals)
	if err != nil {
		panic(err)
	}

	mergeValueMaps(baseVals, overrideVals)

	merged, err := yaml.Marshal(baseVals)
	if err != nil {
		panic(err)
	}

	return merged, nil
}

// we stole this from Helm cmd/helm/installer/install.mergeValues
func mergeValueMaps(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}
		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			dest[k] = v
			continue
		}
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = nextMap
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValueMaps(destMap, nextMap)
	}
	return dest
}