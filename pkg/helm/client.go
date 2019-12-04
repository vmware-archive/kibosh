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
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"reflect"
	"strings"

	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/ghodss/yaml"
	"github.com/gosuri/uitable"
	"github.com/gosuri/uitable/util/strutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helmstaller "k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
	"k8s.io/helm/pkg/tlsutil"
	deploymentutil "k8s.io/kubernetes/pkg/controller/deployment/util"
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
	ResourceReadiness(namespace string, cluster k8s.Cluster) (*string, hapi_release.Status_Code, error)
	Install(*helmstaller.Options) error
	Upgrade(*helmstaller.Options) error
	Uninstall(*helmstaller.Options) error
	InstallChart(registryConfig *config.RegistryConfig, namespace api_v1.Namespace, releaseName string, chart *MyChart, planName string, installValues []byte, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error)
	InstallOperator(chart *MyChart, namespace string) (*rls.InstallReleaseResponse, error)
	UpdateChart(chart *MyChart, rlsName string, planName string, updateValues []byte) (*rls.UpdateReleaseResponse, error)
	HasDifferentTLSConfig() bool
	PrintStatus(out io.Writer, deploymentName string) error
	RenderTemplatedValues(releaseOptions chartutil.ReleaseOptions, inputValues []byte, chart chart.Chart) ([]byte, error)
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

	var servername string
	if c.tlsConf.HasTillerTLS() {
		if c.tlsConf.TillerTLSCertFile != "" {

			pemData, err := ioutil.ReadFile(c.tlsConf.TillerTLSCertFile)
			if err != nil {
				c.logger.Error(err)
			}
			block, _ := pem.Decode(pemData)
			if block == nil {
				c.logger.Error("pem decode")
			} else {
				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					c.logger.Error(err)
				}
				servername = cert.Subject.CommonName
			}
		}
		tlsOpts := tlsutil.Options{
			CaCertFile:         c.tlsConf.TLSCaCertFile,
			KeyFile:            c.tlsConf.HelmTLSKeyFile,
			CertFile:           c.tlsConf.HelmTLSCertFile,
			InsecureSkipVerify: false,
			ServerName:         servername,
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
		return nil, errors.Wrap(err, "Error opening tunnel")
	}
	defer tunnel.Close()
	releases, err := client.ListReleases(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error listing releases on the underlying client")
	}
	return releases, nil
}

func (c myHelmClient) InstallRelease(chStr, namespace string, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	panic("Not yet implemented")
}

func (c myHelmClient) InstallReleaseWithContext(ctx context.Context, chStr, namespace string, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
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

func (c myHelmClient) InstallReleaseFromChartWithContext(ctx context.Context, chart *chart.Chart, namespace string, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.InstallReleaseFromChartWithContext(ctx, chart, namespace, opts...)
}

func (c myHelmClient) InstallChart(registryConfig *config.RegistryConfig, namespace api_v1.Namespace, releaseName string, chart *MyChart, planName string, installValues []byte, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	err := c.cluster.CreateNamespaceIfNotExists(&namespace)
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

	releaseOptions := chartutil.ReleaseOptions{
		Name:      releaseName,
		Namespace: namespaceName,
		IsInstall: true,
		IsUpgrade: false,
	}
	var finalValues []byte
	if planName != "" {
		planOverrideValues, err := MergeValueBytes(chart.TransformedValues, chart.Plans[planName].Values)
		if err != nil {
			return nil, err
		}
		renderedValues, err := c.RenderTemplatedValues(releaseOptions, planOverrideValues, chart.Chart)
		if err != nil {
			return nil, err
		}
		finalValues, err = MergeValueBytes(renderedValues, installValues)
		if err != nil {
			return nil, err
		}
	} else {
		renderedValues, err := c.RenderTemplatedValues(releaseOptions, chart.TransformedValues, chart.Chart)
		if err != nil {
			return nil, err
		}
		finalValues, err = MergeValueBytes(renderedValues, installValues)
	}

	mergedOpts := append(opts, helm.ReleaseName(releaseName), helm.ValueOverrides(finalValues))
	return c.InstallReleaseFromChart(&chart.Chart, namespaceName, mergedOpts...)
}

func (c myHelmClient) RenderTemplatedValues(releaseOptions chartutil.ReleaseOptions, inputValues []byte, chartToInstall chart.Chart) ([]byte, error) {
	ephemeralTemplateName := "templates/ephemeral_kibosh_yaml_template.yaml"
	chartToInstall.Templates = append(chartToInstall.Templates, &chart.Template{
		Name: ephemeralTemplateName,
		Data: inputValues,
	})
	rendered, err := renderutil.Render(&chartToInstall, &chart.Config{}, renderutil.Options{ReleaseOptions: releaseOptions})
	if err != nil {
		return nil, err
	}
	outputValues := rendered[fmt.Sprintf("%s/%s", chartToInstall.Metadata.Name, ephemeralTemplateName)]
	outputValuesBytes := []byte(outputValues)

	return outputValuesBytes, nil
}

func (c myHelmClient) InstallOperator(chart *MyChart, namespace string) (*rls.InstallReleaseResponse, error) {
	return c.InstallReleaseFromChart(&chart.Chart, namespace, helm.ReleaseName(namespace), helm.ValueOverrides(chart.TransformedValues))
}

func (c myHelmClient) UpdateChart(chart *MyChart, rlsName string, planName string, updateValues []byte) (*rls.UpdateReleaseResponse, error) {
	planOverrideValues, err := MergeValueBytes(chart.TransformedValues, chart.Plans[planName].Values)
	if err != nil {
		return nil, err
	}
	releaseOptions := chartutil.ReleaseOptions{
		Name:      rlsName,
		IsInstall: false,
		IsUpgrade: true,
	}
	renderedValues, err := c.RenderTemplatedValues(releaseOptions, planOverrideValues, chart.Chart)
	if err != nil {
		return nil, err
	}
	updateOverrideValues, err := MergeValueBytes(renderedValues, updateValues)

	updateOverrideYaml := map[string]interface{}{}
	err = yaml.Unmarshal(updateOverrideValues, &updateOverrideYaml)
	if err != nil {
		return nil, err
	}

	updatedValuesYaml := map[string]interface{}{}
	err = yaml.Unmarshal(updateValues, &updatedValuesYaml)
	if err != nil {
		return nil, err
	}

	//finalValues should only contain keys from updatedValuesYaml
	v := filterValues(updateOverrideYaml, updatedValuesYaml)
	finalValues, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	c.logger.Infof("Updated helm release with these values: %+v", string(finalValues))
	return c.UpdateReleaseFromChart(rlsName, &chart.Chart, helm.UpdateValueOverrides(finalValues), helm.ReuseValues(true))
}

func (c myHelmClient) DeleteRelease(rlsName string, opts ...helm.DeleteOption) (*rls.UninstallReleaseResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.DeleteRelease(rlsName, opts...)
}

func (c myHelmClient) ResourceReadiness(namespace string, cluster k8s.Cluster) (*string, hapi_release.Status_Code, error) {
	msg, servicesReady, err := c.servicesReady(namespace, cluster)
	if err != nil {
		return msg, hapi_release.Status_UNKNOWN, err
	}
	if !servicesReady {
		return msg, hapi_release.Status_PENDING_INSTALL, nil
	}

	msg, podsReady, err := c.podsReady(namespace, cluster)
	if err != nil {
		return msg, hapi_release.Status_UNKNOWN, err
	}
	if !podsReady {
		return msg, hapi_release.Status_PENDING_INSTALL, nil
	}

	msg, volReady, err := c.volumesReady(namespace, cluster)
	if err != nil {
		return msg, hapi_release.Status_UNKNOWN, err
	}
	if !volReady {
		return msg, hapi_release.Status_PENDING_INSTALL, nil
	}

	msg, deployReady, err := c.deploymentsReady(namespace, cluster)
	if err != nil {
		return msg, hapi_release.Status_UNKNOWN, err
	}

	if !deployReady {
		return msg, hapi_release.Status_PENDING_INSTALL, nil
	}

	return nil, hapi_release.Status_DEPLOYED, nil
}

func (c myHelmClient) volumesReady(namespace string, cluster k8s.Cluster) (*string, bool, error) {
	persistentVolumeClaimList, err := cluster.ListPersistentVolumes(namespace, meta_v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}

	for _, volumeClaim := range persistentVolumeClaimList.Items {
		if volumeClaim.Status.Phase != api_v1.ClaimBound {
			var message string
			message = fmt.Sprintf("PersistentVolumeClaim is not ready: %s/%s", volumeClaim.GetNamespace(), volumeClaim.GetName())
			return &message, false, err
		}
	}

	return nil, true, err
}

func (c myHelmClient) deploymentsReady(namespace string, cluster k8s.Cluster) (*string, bool, error) {
	deployments, err := cluster.ListDeployments(namespace, meta_v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}
	for _, deployment := range deployments.Items {
		if deployment.ReplicaSets.Status.ReadyReplicas < *deployment.Deployment.Spec.Replicas-deploymentutil.MaxUnavailable(*deployment.Deployment) {
			var message string
			message = fmt.Sprintf("Deployment is not ready: %s/%s", deployment.Deployment.GetNamespace(), deployment.Deployment.GetName())
			return &message, false, nil
		}
	}
	return nil, true, nil
}

func (c myHelmClient) ReleaseStatus(rlsName string, opts ...helm.StatusOption) (*rls.GetReleaseStatusResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.ReleaseStatus(rlsName, opts...)
}

func (c myHelmClient) servicesReady(namespace string, cluster k8s.Cluster) (*string, bool, error) {
	services, err := cluster.ListServices(namespace, meta_v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}

	servicesReady := true
	for _, service := range services.Items {
		if service.Spec.Type == "LoadBalancer" {
			if len(service.Status.LoadBalancer.Ingress) < 1 {
				servicesReady = false
			}
		}
	}
	var message string
	if !servicesReady {
		message = "service deployment load balancer in progress"
	}
	return &message, servicesReady, nil
}

func (c myHelmClient) podsReady(namespace string, cluster k8s.Cluster) (*string, bool, error) {
	podList, err := cluster.ListPods(namespace, meta_v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}

	podsReady := true
	var message string
	for _, pod := range podList.Items {

		if c.podIsJob(pod) {
			if pod.Status.Phase != "Succeeded" {
				podsReady = false
				for _, condition := range pod.Status.Conditions {
					if condition.Message != "" {
						message = message + condition.Message + "\n"
					}
				}
			}
		} else {
			if pod.Status.Phase != "Running" {
				podsReady = false
				for _, condition := range pod.Status.Conditions {
					if condition.Message != "" {
						message = message + condition.Message + "\n"
					}
				}
			}
		}
	}

	message = strings.TrimSpace(message)

	return &message, podsReady, nil
}

func (c myHelmClient) podIsJob(pod api_v1.Pod) bool {
	for key := range pod.ObjectMeta.Labels {
		if key == "job-name" {
			return true
		}
	}
	return false
}

func (c myHelmClient) UpdateRelease(rlsName, chStr string, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
	panic("Not yet implemented")
}

func (c myHelmClient) UpdateReleaseWithContext(ctx context.Context, rlsName, chStr string, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
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

func (c myHelmClient) UpdateReleaseFromChartWithContext(ctx context.Context, rlsName string, chart *chart.Chart, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.UpdateReleaseFromChartWithContext(ctx, rlsName, chart, opts...)
}

func (c myHelmClient) RollbackRelease(rlsName string, opts ...helm.RollbackOption) (*rls.RollbackReleaseResponse, error) {
	panic("Not yet implemented")
}

func (c myHelmClient) ReleaseContent(rlsName string, opts ...helm.ContentOption) (*rls.GetReleaseContentResponse, error) {
	tunnel, client, err := c.open()
	if err != nil {
		return nil, err
	}
	defer tunnel.Close()

	return client.ReleaseContent(rlsName, opts...)
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

func MergeValueBytes(base []byte, override []byte) ([]byte, error) {
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

func (c myHelmClient) PrintStatus(out io.Writer, deploymentName string) error {
	panic("Not yet implemented")
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

func filterValues(fullVals map[string]interface{}, filterKeys map[string]interface{}) map[string]interface{} {

	for k, v := range filterKeys {
		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			filterKeys[k] = fullVals[k]
		} else {
			valsMap := fullVals[k].(map[string]interface{})
			filterKeys[k] = filterValues(valsMap, nextMap)
		}
	}
	return filterKeys
}
