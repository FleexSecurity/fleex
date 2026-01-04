package models

import "time"

type BuildRecipe struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Author      string            `yaml:"author"`
	Version     string            `yaml:"version"`
	OS          OSConfig          `yaml:"os"`
	Files       []FileTransfer    `yaml:"files,omitempty"`
	Steps       []BuildStep       `yaml:"steps"`
	Verify      []VerifyStep      `yaml:"verify,omitempty"`
	Vars        map[string]string `yaml:"vars,omitempty"`
}

type OSConfig struct {
	Supported      []string `yaml:"supported"`
	PackageManager string   `yaml:"package_manager"`
}

type FileTransfer struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Mode        string `yaml:"mode,omitempty"`
}

type BuildStep struct {
	Name       string   `yaml:"name"`
	Commands   []string `yaml:"commands"`
	Retries    int      `yaml:"retries,omitempty"`
	Timeout    int      `yaml:"timeout,omitempty"`
	ContinueOn string   `yaml:"continue_on,omitempty"`
}

type VerifyStep struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Expect  string `yaml:"expect,omitempty"`
}

type BuildOptions struct {
	Recipe      *BuildRecipe
	FleetName   string
	Parallel    int
	NoVerify    bool
	ContinueErr bool
	DryRun      bool
	Verbose     bool
}

type BuildResult struct {
	BoxName  string
	Success  bool
	Steps    []StepResult
	Duration time.Duration
	Error    error
}

type StepResult struct {
	StepName string
	Success  bool
	Output   string
	Retries  int
	Duration time.Duration
}
