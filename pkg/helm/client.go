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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"

	"github.com/Sirupsen/logrus"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/ghodss/yaml"
	"github.com/gosuri/uitable"
	"github.com/gosuri/uitable/util/strutil"
	"io"
	api_v1 "k8s.io/api/core/v1"
	helmstaller "k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/timeconv"
	"k8s.io/helm/pkg/tlsutil"
	"regexp"
	"text/tabwriter"
)

type myHelmClient struct {
	cluster   k8s.Cluster
	tlsConf   *config.HelmTLSConfig
	namespace string
	logger    *logrus.Logger
}

//go:generate counterfeiter ./ MyHelmClient
type MyHelmClient interface {
	helm.Interface
	Install(*helmstaller.Options) error
	Upgrade(*helmstaller.Options) error
	Uninstall(*helmstaller.Options) error
	InstallChart(registryConfig *config.RegistryConfig, namespace api_v1.Namespace, chart *MyChart, planName string, installValues []byte) (*rls.InstallReleaseResponse, error)
	InstallOperator(chart *MyChart, namespace string) (*rls.InstallReleaseResponse, error)
	UpdateChart(chart *MyChart, rlsName string, planName string, updateValues []byte) (*rls.UpdateReleaseResponse, error)
	MergeValueBytes(base []byte, override []byte) ([]byte, error)
	HasDifferentTLSConfig() bool
	PrintStatus(out io.Writer, deploymentName string) error
}

func NewMyHelmClient(cluster k8s.Cluster, tlsConf *config.HelmTLSConfig, namespace string, logger *logrus.Logger) MyHelmClient {
	return &myHelmClient{
		cluster:   cluster,
		tlsConf:   tlsConf,
		namespace: namespace,
		logger:    logger,
	}
}

func (c myHelmClient) open() (*kube.Tunnel, helm.Interface, error) {
	conf, client := c.cluster.GetClientConfig(), c.cluster.GetClient()
	tunnel, err := portforwarder.New(c.namespace, client, conf)
	if err != nil {
		return nil, nil, err
	}

	host := fmt.Sprintf("127.0.0.1:%d", tunnel.Local)
	c.logger.Debug("Tunnel", map[string]interface{}{"host": host})

	opts := []helm.Option{
		helm.Host(host),
	}
	if c.tlsConf.HasTillerTLS() {
		tlsOpts := tlsutil.Options{
			CaCertFile:         c.tlsConf.TLSCaCertFile,
			KeyFile:            c.tlsConf.HelmTLSKeyFile,
			CertFile:           c.tlsConf.HelmTLSCertFile,
			InsecureSkipVerify: false,
		}

		tlsCfg, err := tlsutil.ClientConfig(tlsOpts)
		if err != nil {
			return nil, nil, err
		}

		caBytes, err := ioutil.ReadFile(c.tlsConf.TLSCaCertFile)
		//trust our own cert
		tlsCfg.RootCAs.AppendCertsFromPEM(caBytes)
		if err != nil {
			return nil, nil, err
		}

		opts = append(opts, helm.WithTLS(tlsCfg))
	}

	return tunnel, helm.NewClient(opts...), nil
}

func (c *myHelmClient) HasDifferentTLSConfig() bool {
	_, err := c.ListReleases()
	if err != nil {
		_, isUAError := err.(x509.UnknownAuthorityError)
		if isUAError {
			return true
		}

		_, isHostnameError := err.(x509.HostnameError)
		if isHostnameError {
			return true
		}

		opError, isOpError := err.(*net.OpError)
		if isOpError {
			errorType := reflect.TypeOf(opError.Err)
			if errorType.String() == "tls.alert" {
				return true
			}
		}

		_, isRecordHeaderError := err.(tls.RecordHeaderError)
		if isRecordHeaderError {
			return true
		}
	}
	return false
}

func (c *myHelmClient) Install(opts *helmstaller.Options) error {
	return helmstaller.Install(c.cluster.GetClient(), opts)
}

func (c *myHelmClient) Upgrade(opts *helmstaller.Options) error {
	return helmstaller.Upgrade(c.cluster.GetClient(), opts)
}

func (c *myHelmClient) Uninstall(opts *helmstaller.Options) error {
	return helmstaller.Uninstall(c.cluster.GetClient(), opts)
}

func (c myHelmClient) ListReleases(opts ...helm.ReleaseListOption) (*rls.ListReleasesResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()
	releases, err := client.ListReleases(opts...)
	if err != nil {
		return nil, err
	}
	return releases, nil
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

func (c myHelmClient) InstallChart(registryConfig *config.RegistryConfig, namespace api_v1.Namespace, chart *MyChart, planName string, installValues []byte) (*rls.InstallReleaseResponse, error) {
	_, err := c.cluster.CreateNamespace(&namespace)
	if err != nil {
		return nil, err
	}

	namespaceName := namespace.Name
	if registryConfig.HasRegistryConfig() {
		privateRegistrySetup := k8s.NewPrivateRegistrySetup(namespaceName, "default", c.cluster, registryConfig)
		err := privateRegistrySetup.Setup()
		if err != nil {
			return nil, err
		}
	}

	overrideValues, err := c.MergeValueBytes(chart.Values, chart.Plans[planName].Values)
	if err != nil {
		return nil, err
	}
	mergedValues, _ := c.MergeValueBytes(overrideValues, installValues)
	if err != nil {
		return nil, err
	}

	return c.InstallReleaseFromChart(chart.Chart, namespaceName, helm.ReleaseName(namespaceName) /* here he is!*/, helm.ValueOverrides(mergedValues))
}

func (c myHelmClient) InstallOperator(chart *MyChart, namespace string) (*rls.InstallReleaseResponse, error) {
	return c.InstallReleaseFromChart(chart.Chart, namespace, helm.ReleaseName(namespace), helm.ValueOverrides(chart.Values))
}

func (c myHelmClient) UpdateChart(chart *MyChart, rlsName string, planName string, updateValues []byte) (*rls.UpdateReleaseResponse, error) {
	existingChartValues, err := c.MergeValueBytes(chart.Values, chart.Plans[planName].Values)
	if err != nil {
		return nil, err
	}

	mergedValues, err := c.MergeValueBytes(existingChartValues, updateValues)
	if err != nil {
		return nil, err
	}

	return c.UpdateReleaseFromChart(rlsName, chart.Chart, helm.UpdateValueOverrides(mergedValues))
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
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.UpdateReleaseFromChart(rlsName, chart, opts...)
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
		return nil, err
	}
	overrideVals := map[string]interface{}{}
	err = yaml.Unmarshal(override, &overrideVals)
	if err != nil {
		return nil, err
	}

	mergeValueMaps(baseVals, overrideVals)

	merged, err := yaml.Marshal(baseVals)
	if err != nil {
		return nil, err
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

// This is copied from helm client since it was in the main package
// https://github.com/helm/helm/blob/v2.9.1/cmd/helm/status.go
func (c myHelmClient) PrintStatus(out io.Writer, deploymentName string) error {
	res, err := c.ReleaseStatus(deploymentName)
	if err != nil {
		return err
	}

	if res.Info.LastDeployed != nil {
		fmt.Fprintf(out, "LAST DEPLOYED: %s\n", timeconv.String(res.Info.LastDeployed))
	}
	fmt.Fprintf(out, "NAMESPACE: %s\n", res.Namespace)
	fmt.Fprintf(out, "STATUS: %s\n", res.Info.Status.Code)
	fmt.Fprintf(out, "\n")
	if len(res.Info.Status.Resources) > 0 {
		re := regexp.MustCompile("  +")

		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', tabwriter.TabIndent)
		fmt.Fprintf(w, "RESOURCES:\n%s\n", re.ReplaceAllString(res.Info.Status.Resources, "\t"))
		w.Flush()
	}
	if res.Info.Status.LastTestSuiteRun != nil {
		lastRun := res.Info.Status.LastTestSuiteRun
		fmt.Fprintf(out, "TEST SUITE:\n%s\n%s\n\n%s\n",
			fmt.Sprintf("Last Started: %s", timeconv.String(lastRun.StartedAt)),
			fmt.Sprintf("Last Completed: %s", timeconv.String(lastRun.CompletedAt)),
			c.formatTestResults(lastRun.Results))
	}

	if len(res.Info.Status.Notes) > 0 {
		fmt.Fprintf(out, "NOTES:\n%s\n", res.Info.Status.Notes)
	}
	return nil
}

func (c myHelmClient) formatTestResults(results []*release.TestRun) string {
	tbl := uitable.New()
	tbl.MaxColWidth = 50
	tbl.AddRow("TEST", "STATUS", "INFO", "STARTED", "COMPLETED")
	for i := 0; i < len(results); i++ {
		r := results[i]
		n := r.Name
		s := strutil.PadRight(r.Status.String(), 10, ' ')
		i := r.Info
		ts := timeconv.String(r.StartedAt)
		tc := timeconv.String(r.CompletedAt)
		tbl.AddRow(n, s, i, ts, tc)
	}
	return tbl.String()
}
