package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/ghodss/yaml"
)

func main() {

	argsWithoutProgramName := os.Args[1:]
	if len(argsWithoutProgramName) != 2 {
		os.Stderr.WriteString("expecting two args: namespace followed by template path")
		os.Exit(1)
	}
	namespace := argsWithoutProgramName[0]
	filename := argsWithoutProgramName[1]

	cluster, err := k8s.NewClusterFromDefaultConfig()
	exitOnErr(err)

	servicesAndSecrets, err := cluster.GetSecretsAndServices(namespace)
	exitOnErr(err)

	println(fmt.Sprintf("servicesAndSecrets %v", servicesAndSecrets))
	rawTemplate, err := ioutil.ReadFile(filename)
	exitOnErr(err)

	bind := &helm.Bind{}
	err = yaml.Unmarshal(rawTemplate, bind)

	renderedTemplate, err := helm.RenderJsonnetTemplate(string(bind.Template), servicesAndSecrets)
	exitOnErr(err)

	var bindCredentials map[string]interface{}
	err = json.Unmarshal([]byte(renderedTemplate), &bindCredentials)
	exitOnErr(err)

	println("Rendered template")
	println(renderedTemplate)
}

func exitOnErr(err error) {
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}
