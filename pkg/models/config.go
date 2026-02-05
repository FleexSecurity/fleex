package models

import (
	"strings"
)

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
		if matchesFleetName(customVM.InstanceID, name) {
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

// matchesFleetName determines if a label matches the given fleet name.
// If name ends with -{number}, only exact matches are returned.
// Otherwise, prefix matching is used.
// Note: this is a local copy to avoid circular import with utils package.
func matchesFleetName(label, name string) bool {
	if label == name {
		return true
	}

	parts := strings.Split(name, "-")
	if len(parts) >= 2 {
		lastPart := parts[len(parts)-1]
		if len(lastPart) > 0 {
			isNumber := true
			for _, c := range lastPart {
				if c < '0' || c > '9' {
					isNumber = false
					break
				}
			}
			if isNumber {
				return false
			}
		}
	}

	return strings.HasPrefix(label, name)
}
