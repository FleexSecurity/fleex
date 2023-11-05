package utils

import (
	"io/ioutil"

	"github.com/FleexSecurity/fleex/pkg/models"
	"gopkg.in/yaml.v2"
)

func ReadModuleFile(path string) (*models.Module, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &models.Module{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
