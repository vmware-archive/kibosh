package main

import (
	"encoding/json"
	"fmt"
	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

func main() {

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

	servicesAndSecrets, err := cluster.GetSecretsAndServices(namespace)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	println(fmt.Sprintf("servicesAndSecrets %v", servicesAndSecrets))
	rawTemplate, err := ioutil.ReadFile(filename)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	bind := &helm.Bind{}
	err = yaml.Unmarshal(rawTemplate, bind)

	renderedTemplate, err := helm.RenderJsonnetTemplate(string(bind.Template), servicesAndSecrets)
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
