package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

func ReadBuildFile(nameOrPath string) (*models.BuildRecipe, error) {
	var path string

	if FileExists(nameOrPath) {
		path = nameOrPath
	} else {
		configDir, err := GetConfigDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(configDir, "fleex", "builds", nameOrPath+".yaml")
		if !FileExists(path) {
			return nil, fmt.Errorf("build recipe not found: %s", nameOrPath)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	recipe := &models.BuildRecipe{}
	if err := yaml.Unmarshal(data, recipe); err != nil {
		return nil, err
	}

	return recipe, nil
}

func ListBuildRecipes() ([]string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	buildsPath := filepath.Join(configDir, "fleex", "builds")

	if !FileExists(buildsPath) {
		return []string{}, nil
	}

	files, err := os.ReadDir(buildsPath)
	if err != nil {
		return nil, err
	}

	var recipes []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".yaml") || strings.HasSuffix(f.Name(), ".yml") {
			recipes = append(recipes, strings.TrimSuffix(strings.TrimSuffix(f.Name(), ".yaml"), ".yml"))
		}
	}
	return recipes, nil
}

func SaveBuildRecipe(recipe *models.BuildRecipe) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	buildsPath := filepath.Join(configDir, "fleex", "builds")

	if !FileExists(buildsPath) {
		if err := os.MkdirAll(buildsPath, 0755); err != nil {
			return err
		}
	}

	data, err := yaml.Marshal(recipe)
	if err != nil {
		return err
	}

	path := filepath.Join(buildsPath, recipe.Name+".yaml")
	return os.WriteFile(path, data, 0644)
}

func ReplaceBuildVars(text string, vars map[string]string) string {
	for k, v := range vars {
		placeholder := fmt.Sprintf("{vars.%s}", k)
		text = strings.ReplaceAll(text, placeholder, v)
	}
	return text
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := homedir.Dir()
		if err == nil {
			path = strings.Replace(path, "~", home, 1)
		}
	}
	return path
}

func GetBuildsDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "fleex", "builds"), nil
}
