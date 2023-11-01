package models

type Module struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Author      string            `yaml:"author"`
	Vars        map[string]string `yaml:"vars"`
	Commands    []string          `yaml:"commands"`
}
