package main

import (
	"encoding/json"
	"fmt"
	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/google/go-jsonnet"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
)

func main() {
	println("----------------hello")

	argsWithoutProgramName := os.Args[1:]
	if len(argsWithoutProgramName) != 2 {
		os.Stderr.WriteString("single arg expected the path to parse")
		os.Exit(1)
	}
	namespace := argsWithoutProgramName[0]
	filename := argsWithoutProgramName[1]

	cluster, err := k8s.NewClusterFromDefaultConfig()
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	secrets, err := cluster.ListSecrets(namespace, meta_v1.ListOptions{})
	secretsMap := []map[string]interface{}{}
	for _, secret := range secrets.Items {
		if secret.Type == api_v1.SecretTypeOpaque {
			credentialSecrets := map[string]string{}
			for key, val := range secret.Data {
				credentialSecrets[key] = string(val)
			}
			credential := map[string]interface{}{
				"name": secret.Name,
				"data": credentialSecrets,
			}
			secretsMap = append(secretsMap, credential)
		}
	}

	services, err := cluster.ListServices(namespace, meta_v1.ListOptions{})
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	servicesMap := []map[string]interface{}{}
	for _, service := range services.Items {
		if service.Spec.Type == "NodePort" {
			nodes, _ := cluster.ListNodes(meta_v1.ListOptions{})
			for _, node := range nodes.Items {
				service.Spec.ExternalIPs = append(service.Spec.ExternalIPs, node.ObjectMeta.Labels["spec.ip"])
			}
		}
		credentialService := map[string]interface{}{
			"name":   service.ObjectMeta.Name,
			"spec":   service.Spec,
			"status": service.Status,
		}
		servicesMap = append(servicesMap, credentialService)
	}

	servicesAndSecrets := map[string]interface{}{
		"secrets":  secretsMap,
		"services": servicesMap,
	}

	println(fmt.Sprintf("servicesAndSecrets %v", servicesAndSecrets))

	println("⭐⭐⭐⭐⭐⭐⭐⭐⭐⭐⭐⭐⭐⭐⭐⭐⭐")

	rawTemplate, err := ioutil.ReadFile(filename)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	bind := &helm.Bind{}
	err = yaml.Unmarshal(rawTemplate, bind)

	renderedTemplate, err := getRenderedTemplate(string(bind.Template), servicesAndSecrets)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
	var bindCredentials map[string]interface{}
	err = json.Unmarshal([]byte(renderedTemplate), &bindCredentials)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	println("Rendered template")
	println(renderedTemplate)
}

func getRenderedTemplate(template string, servicesAndSecrets map[string]interface{}) (string, error) {
	ssTemplateBytes, err := json.Marshal(servicesAndSecrets)
	if err != nil {
		return "", err
	}

	ssTemplate := string(ssTemplateBytes)

	i := strings.LastIndex(ssTemplate, "}")
	fullTemplate := ssTemplate[0:i] + `,"template": ` + template + "}"
	vm := jsonnet.MakeVM()
	renderedTemplate, err := vm.EvaluateSnippetMulti("", fullTemplate)
	if err != nil {
		return "", err
	}

	return renderedTemplate["template"], nil
}
