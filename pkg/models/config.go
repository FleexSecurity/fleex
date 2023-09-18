package models

type Config struct {
	Providers map[string]Provider `json:"providers"`
	SSHKeys   SSHKeys             `json:"ssh_keys"`
	Settings  Settings            `json:"settings"`
}

type Provider struct {
	Token    string   `json:"token"`
	Region   string   `json:"region"`
	Size     string   `json:"size"`
	Image    string   `json:"image"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Tags     []string `json:"tags,omitempty"`
}

type SSHKeys struct {
	PublicFile  string `json:"public_file"`
	PrivateFile string `json:"private_file"`
}

type Settings struct {
	Provider string `json:"provider"`
}
