package docker

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
)

type ImageValues struct {
	Image    string                 `yaml:"image"`
	ImageTag string                 `yaml:"imageTag"`
	Images   map[string]ImageValues `yaml:"images"`
}

type DockerImageOutput struct {
	Repo string `yaml:"repo"`
	Tag  string `yaml:"tag"`
}

func TagAndPush(privateRegistryServer string) error {
	cmd := exec.Command("docker", "images", "--format", "- repo: {{ json .Repository }}\n  tag: {{ json .Tag }}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	images := []DockerImageOutput{}
	err = yaml.Unmarshal([]byte(output), &images)
	if err != nil {
		return err
	}

	for _, image := range images {
		split := strings.Split(image.Repo, "/")
		imageName := fmt.Sprintf("%s/%s", privateRegistryServer, split[len(split)-1])

		origTag := fmt.Sprintf("%s:%s", image.Repo, image.Tag)
		newTag := fmt.Sprintf("%s:%s", imageName, image.Tag)

		cmd := exec.Command("docker", "tag", origTag, newTag)
		output, err := cmd.CombinedOutput()
		println(string(output))
		if err != nil {
			return err
		}

		cmd = exec.Command("docker", "push", newTag)
		output, err = cmd.CombinedOutput()
		println(string(output))
		if err != nil {
			return err
		}
	}

	return nil
}

func LoadImage(imagePath string) error {
	cmd := exec.Command("docker", "load", "-i", imagePath)
	output, err := cmd.CombinedOutput()
	println(string(output))
	return err
}

func ParseValues(chartPath string) (*ImageValues, error) {
	valuesPath := path.Join(chartPath, "values.yaml")
	bytes, err := ioutil.ReadFile(valuesPath)
	if err != nil {
		return nil, err
	}

	parsedImages := &ImageValues{}
	err = yaml.Unmarshal(bytes, parsedImages)
	if err != nil {
		return nil, err
	}
	if !parsedImages.ValidateImages() {
		return nil, err
	}

	return parsedImages, nil
}

func (i *ImageValues) ValidateImages() bool {
	if !i.validateImage() && len(i.Images) == 0 {
		return false

	}
	for _, image := range i.Images {
		if !image.validateImage() {
			return false
		}
	}
	return true
}

func (i *ImageValues) validateImage() bool {
	if i.Image == "" || i.ImageTag == "" {
		return false
	}
	return true
}
