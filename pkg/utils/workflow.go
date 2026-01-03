package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/models"
	"gopkg.in/yaml.v2"
)

func ReadWorkflowFile(nameOrPath string) (*models.Workflow, error) {
	var path string

	if FileExists(nameOrPath) {
		path = nameOrPath
	} else {
		configDir, err := GetConfigDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(configDir, "fleex", "workflows", nameOrPath+".yaml")
		if !FileExists(path) {
			return nil, fmt.Errorf("workflow not found: %s", nameOrPath)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	workflow := &models.Workflow{}
	if err := yaml.Unmarshal(data, workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

func ListWorkflows() ([]string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	workflowsPath := filepath.Join(configDir, "fleex", "workflows")

	if !FileExists(workflowsPath) {
		return []string{}, nil
	}

	files, err := os.ReadDir(workflowsPath)
	if err != nil {
		return nil, err
	}

	var workflows []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".yaml") || strings.HasSuffix(f.Name(), ".yml") {
			workflows = append(workflows, strings.TrimSuffix(strings.TrimSuffix(f.Name(), ".yaml"), ".yml"))
		}
	}
	return workflows, nil
}

func SaveWorkflow(workflow *models.Workflow) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	workflowsPath := filepath.Join(configDir, "fleex", "workflows")

	if !FileExists(workflowsPath) {
		if err := os.MkdirAll(workflowsPath, 0755); err != nil {
			return err
		}
	}

	data, err := yaml.Marshal(workflow)
	if err != nil {
		return err
	}

	path := filepath.Join(workflowsPath, workflow.Name+".yaml")
	return os.WriteFile(path, data, 0644)
}

func GetWorkflowsDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "fleex", "workflows"), nil
}

func ReplaceWorkflowVars(text string, vars map[string]string) string {
	for k, v := range vars {
		placeholder := fmt.Sprintf("{vars.%s}", k)
		text = strings.ReplaceAll(text, placeholder, v)
	}
	text = strings.ReplaceAll(text, "{INPUT}", vars["INPUT"])
	text = strings.ReplaceAll(text, "{OUTPUT}", vars["OUTPUT"])
	return text
}
