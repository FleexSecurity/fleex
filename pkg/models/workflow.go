package models

type Workflow struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Author      string            `yaml:"author"`
	Vars        map[string]string `yaml:"vars"`
	Files       []FileTransfer    `yaml:"files,omitempty"`
	Setup       []string          `yaml:"setup,omitempty"`
	Steps       []WorkflowStep    `yaml:"steps"`
	Output      WorkflowOutput    `yaml:"output,omitempty"`
}

type WorkflowStep struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Timeout string `yaml:"timeout,omitempty"`
}

type WorkflowOutput struct {
	Aggregate   string `yaml:"aggregate,omitempty"`
	Deduplicate bool   `yaml:"deduplicate,omitempty"`
}

type WorkflowOptions struct {
	Workflow     *Workflow
	FleetName    string
	Input        string
	Output       string
	ChunksFolder string
	Delete       bool
	DryRun       bool
	Verbose      bool
}

type WorkflowResult struct {
	BoxName     string
	Success     bool
	StepResults []WorkflowStepResult
	Error       error
}

type WorkflowStepResult struct {
	StepName string
	Success  bool
	Output   string
}
