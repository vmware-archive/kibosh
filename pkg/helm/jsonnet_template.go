package helm

import (
	"encoding/json"
	"strings"

	"github.com/google/go-jsonnet"
)

func RenderJsonnetTemplate(template string, data map[string]interface{}) (string, error) {
	ssTemplateBytes, err := json.Marshal(data)
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
