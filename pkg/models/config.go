package models

type Config struct {
	Providers map[string]Provider `json:"providers"`
	CustomVMs []CustomVM          `json:"custom_vms"`
	SSHKeys   SSHKeys             `json:"ssh_keys"`
	Settings  Settings            `json:"settings"`
}

type Provider struct {
	Token    string   `json:"token,omitempty"`
	Region   string   `json:"region,omitempty"`
	Size     string   `json:"size,omitempty"`
	Image    string   `json:"image,omitempty"`
	Port     int      `json:"port,omitempty"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type CustomVM struct {
	Provider   string   `json:"provider"`
	InstanceID string   `json:"instance_id"`
	PublicIP   string   `json:"public_ip"`
	SSHPort    int      `json:"ssh_port"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	KeyPath    string   `json:"key_path"`
	Tags       []string `json:"tags"`
}

type SSHKeys struct {
	PublicFile  string `json:"public_file"`
	PrivateFile string `json:"private_file"`
}

type Settings struct {
	Provider string `json:"provider"`
}

type VMInfo struct {
	Provider string
	IP       string
	Port     int
	Username string
	Password string
	KeyPath  string
}

func GetVMInfo(provider, name string, config *Config) *VMInfo {
	if providerConfig, exists := config.Providers[provider]; exists {
		return &VMInfo{
			Provider: provider,
			Port:     providerConfig.Port,
			Username: providerConfig.Username,
			Password: providerConfig.Password,
			KeyPath:  config.SSHKeys.PrivateFile,
		}
	}

	for _, customVM := range config.CustomVMs {
		if customVM.InstanceID == name {
			return &VMInfo{
				Provider: provider,
				IP:       customVM.PublicIP,
				Port:     customVM.SSHPort,
				Username: customVM.Username,
				Password: customVM.Password,
				KeyPath:  customVM.KeyPath,
			}
		}
	}

	return nil
}
