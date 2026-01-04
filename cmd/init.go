package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/sshutils"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Fleex initialization wizard",
	Run: func(cmd *cobra.Command, args []string) {
		wizard, _ := cmd.Flags().GetBool("wizard")
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		addProvider, _ := cmd.Flags().GetString("add-provider")

		configDir, err := utils.GetConfigDir()
		if err != nil {
			utils.Log.Fatal(err)
		}

		fleexPath := filepath.Join(configDir, "fleex")

		if addProvider != "" {
			runAddProvider(addProvider, fleexPath)
			return
		}

		if _, err := os.Stat(fleexPath); !os.IsNotExist(err) {
			if !overwrite {
				utils.Log.Fatal("Fleex folder already exists. Use --overwrite to replace it")
			}
		}

		if err := os.MkdirAll(fleexPath, 0755); err != nil {
			utils.Log.Fatal(err)
		}

		reader := bufio.NewReader(os.Stdin)

		if wizard {
			runWizard(reader, fleexPath)
		} else {
			runQuickSetup(reader, fleexPath)
		}
	},
}

func runWizard(reader *bufio.Reader, fleexPath string) {
	fmt.Println("\n=== FLEEX SETUP WIZARD ===\n")

	fmt.Println("Select your primary use case:")
	fmt.Println("  1. Bug Bounty Hunting")
	fmt.Println("  2. Penetration Testing")
	fmt.Println("  3. Security Research")
	fmt.Println("  4. Continuous Scanning")
	fmt.Print("\nChoice [1-4]: ")
	useCaseChoice, _ := reader.ReadString('\n')
	useCaseChoice = strings.TrimSpace(useCaseChoice)

	fmt.Println("\nWhich cloud providers do you have accounts with?")
	fmt.Println("(Enter comma-separated numbers, e.g., 1,2)")
	fmt.Println("  1. Linode")
	fmt.Println("  2. DigitalOcean")
	fmt.Println("  3. Vultr")
	fmt.Println("  4. Custom VMs only")
	fmt.Print("\nChoice: ")
	providerChoice, _ := reader.ReadString('\n')
	providerChoice = strings.TrimSpace(providerChoice)

	config := models.Config{
		Providers: make(map[string]models.Provider),
		SSHKeys:   models.SSHKeys{},
		Settings:  models.Settings{},
	}

	providers := strings.Split(providerChoice, ",")
	var primaryProvider string

	for _, p := range providers {
		p = strings.TrimSpace(p)
		switch p {
		case "1":
			fmt.Println("\n--- Linode Configuration ---")
			config.Providers["linode"] = configureProvider(reader, "linode")
			if primaryProvider == "" {
				primaryProvider = "linode"
			}
		case "2":
			fmt.Println("\n--- DigitalOcean Configuration ---")
			config.Providers["digitalocean"] = configureProvider(reader, "digitalocean")
			if primaryProvider == "" {
				primaryProvider = "digitalocean"
			}
		case "3":
			fmt.Println("\n--- Vultr Configuration ---")
			config.Providers["vultr"] = configureProvider(reader, "vultr")
			if primaryProvider == "" {
				primaryProvider = "vultr"
			}
		case "4":
			fmt.Println("\n--- Custom VMs Configuration ---")
			config.CustomVMs = configureCustomVMs(reader)
			if primaryProvider == "" {
				primaryProvider = "custom"
			}
		}
	}

	if primaryProvider == "" {
		primaryProvider = "linode"
		config.Providers["linode"] = models.Provider{
			Region:   "us-east",
			Size:     "g6-nanode-1",
			Port:     22,
			Username: "root",
		}
	}

	config.Settings.Provider = primaryProvider

	sshPath := filepath.Join(fleexPath, "ssh")
	if err := os.MkdirAll(sshPath, 0700); err != nil {
		utils.Log.Fatal(err)
	}

	hostname, _ := os.Hostname()
	userInfo, _ := user.Current()
	email := fmt.Sprintf("%s@%s", userInfo.Username, hostname)

	fmt.Println("\n--- SSH Key Generation ---")
	fmt.Printf("Generating SSH key pair with email: %s\n", email)

	if err := sshutils.GenerateSSHKeyPair(4096, email, sshPath); err != nil {
		utils.Log.Fatal(err)
	}

	config.SSHKeys.PublicFile = filepath.Join(sshPath, "id_rsa.pub")
	config.SSHKeys.PrivateFile = filepath.Join(sshPath, "id_rsa")

	configPath := filepath.Join(fleexPath, "config.json")
	configFile, err := os.Create(configPath)
	if err != nil {
		utils.Log.Fatal(err)
	}
	defer configFile.Close()

	encoder := json.NewEncoder(configFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		utils.Log.Fatal(err)
	}

	createDefaultWorkflows(fleexPath)
	createDefaultBuilds(fleexPath)

	fmt.Println("\n=== SETUP COMPLETE ===")
	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Printf("SSH keys generated in: %s\n", sshPath)
	fmt.Println("\nQuick start commands:")
	fmt.Println("  fleex ls                           # List running instances")
	fmt.Println("  fleex spawn -n myfleet -c 5        # Spawn 5 instances")
	fmt.Println("  fleex spawn -n myfleet -c 5 --build security-tools  # Spawn and provision")
	fmt.Println("  fleex build list                   # View available build recipes")
	fmt.Println("  fleex workflow list                # View available workflows")
}

func runQuickSetup(reader *bufio.Reader, fleexPath string) {
	fmt.Println("\n=== FLEEX QUICK SETUP ===\n")
	fmt.Println("For interactive wizard, use: fleex init --wizard")
	fmt.Println()

	fmt.Print("Enter your preferred provider (linode/digitalocean/vultr) [linode]: ")
	provider, _ := reader.ReadString('\n')
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "linode"
	}

	fmt.Print("Enter API token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	config := models.Config{
		Providers: map[string]models.Provider{
			provider: getDefaultProviderConfig(provider, token),
		},
		Settings: models.Settings{
			Provider: provider,
		},
	}

	sshPath := filepath.Join(fleexPath, "ssh")
	if err := os.MkdirAll(sshPath, 0700); err != nil {
		utils.Log.Fatal(err)
	}

	hostname, _ := os.Hostname()
	userInfo, _ := user.Current()
	email := fmt.Sprintf("%s@%s", userInfo.Username, hostname)

	if err := sshutils.GenerateSSHKeyPair(4096, email, sshPath); err != nil {
		utils.Log.Fatal(err)
	}

	config.SSHKeys.PublicFile = filepath.Join(sshPath, "id_rsa.pub")
	config.SSHKeys.PrivateFile = filepath.Join(sshPath, "id_rsa")

	configPath := filepath.Join(fleexPath, "config.json")
	configFile, err := os.Create(configPath)
	if err != nil {
		utils.Log.Fatal(err)
	}
	defer configFile.Close()

	encoder := json.NewEncoder(configFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		utils.Log.Fatal(err)
	}

	createDefaultWorkflows(fleexPath)
	createDefaultBuilds(fleexPath)

	fmt.Println("\nSetup complete!")
	fmt.Printf("Configuration: %s\n", configPath)
}

func configureProvider(reader *bufio.Reader, provider string) models.Provider {
	fmt.Print("API Token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	defaults := getDefaultProviderConfig(provider, token)

	fmt.Printf("Region [%s]: ", defaults.Region)
	region, _ := reader.ReadString('\n')
	region = strings.TrimSpace(region)
	if region != "" {
		defaults.Region = region
	}

	fmt.Printf("Instance size [%s]: ", defaults.Size)
	size, _ := reader.ReadString('\n')
	size = strings.TrimSpace(size)
	if size != "" {
		defaults.Size = size
	}

	fmt.Printf("SSH Port [%d]: ", defaults.Port)
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	if portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			defaults.Port = port
		}
	}

	fmt.Printf("SSH Username [%s]: ", defaults.Username)
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	if username != "" {
		defaults.Username = username
	}

	fmt.Print("SSH Password (optional): ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)
	if password != "" {
		defaults.Password = password
	}

	return defaults
}

func configureCustomVMs(reader *bufio.Reader) []models.CustomVM {
	var vms []models.CustomVM

	fmt.Println("Enter custom VMs (empty IP to finish):")

	for i := 1; ; i++ {
		fmt.Printf("\nVM #%d\n", i)
		fmt.Print("  Public IP: ")
		ip, _ := reader.ReadString('\n')
		ip = strings.TrimSpace(ip)
		if ip == "" {
			break
		}

		fmt.Print("  SSH Port [22]: ")
		portStr, _ := reader.ReadString('\n')
		portStr = strings.TrimSpace(portStr)
		port := 22
		if portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}

		fmt.Print("  Username [root]: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)
		if username == "" {
			username = "root"
		}

		fmt.Print("  Instance ID (e.g., vm-1): ")
		instanceID, _ := reader.ReadString('\n')
		instanceID = strings.TrimSpace(instanceID)
		if instanceID == "" {
			instanceID = fmt.Sprintf("custom-vm-%d", i)
		}

		vms = append(vms, models.CustomVM{
			Provider:   "custom",
			InstanceID: instanceID,
			PublicIP:   ip,
			SSHPort:    port,
			Username:   username,
		})
	}

	return vms
}

func getDefaultProviderConfig(provider, token string) models.Provider {
	switch provider {
	case "digitalocean":
		return models.Provider{
			Token:    token,
			Region:   "nyc1",
			Size:     "s-1vcpu-1gb",
			Image:    "ubuntu-22-04-x64",
			Port:     22,
			Username: "root",
		}
	case "vultr":
		return models.Provider{
			Token:    token,
			Region:   "ewr",
			Size:     "vc2-1c-1gb",
			Image:    "387",
			Port:     22,
			Username: "root",
		}
	default:
		return models.Provider{
			Token:    token,
			Region:   "us-east",
			Size:     "g6-nanode-1",
			Image:    "linode/ubuntu22.04",
			Port:     22,
			Username: "root",
		}
	}
}

func createDefaultBuilds(fleexPath string) {
	buildsPath := filepath.Join(fleexPath, "builds")
	if err := os.MkdirAll(buildsPath, 0755); err != nil {
		return
	}

	securityTools := `name: security-tools
description: Common security tools for bug bounty
author: fleex
version: "1.0.0"
os:
  supported:
    - ubuntu
    - debian
vars:
  GO_VERSION: "1.21.5"
  USERNAME: root
steps:
  - name: System Update
    commands:
      - DEBIAN_FRONTEND=noninteractive apt-get update -qq
      - DEBIAN_FRONTEND=noninteractive apt-get -o Dpkg::Options::=--force-confold -o Dpkg::Options::=--force-confdef upgrade -y -qq
    retries: 2
    timeout: 600
  - name: Install Base Packages
    commands:
      - DEBIAN_FRONTEND=noninteractive apt-get install -y -qq git curl wget jq make gcc libpcap-dev
    retries: 2
  - name: Install Go
    commands:
      - wget -q https://go.dev/dl/go{vars.GO_VERSION}.linux-amd64.tar.gz -O /tmp/go.tar.gz
      - rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz
      - echo 'export PATH=$PATH:/usr/local/go/bin:/root/go/bin' >> ~/.bashrc
  - name: Install ProjectDiscovery Tools
    commands:
      - /usr/local/go/bin/go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest
      - /usr/local/go/bin/go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest
      - /usr/local/go/bin/go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest
    retries: 1
  - name: Install Masscan
    commands:
      - git clone --depth 1 https://github.com/robertdavidgraham/masscan /tmp/masscan
      - cd /tmp/masscan && make -j4 && cp bin/masscan /usr/local/bin/
    retries: 1
verify:
  - name: Check nuclei
    command: /root/go/bin/nuclei --version
  - name: Check httpx
    command: /root/go/bin/httpx --version
  - name: Check masscan
    command: masscan --version
`

	reconTools := `name: recon-tools
description: Reconnaissance and enumeration tools
author: fleex
version: "1.0.0"
os:
  supported:
    - ubuntu
    - debian
vars:
  USERNAME: op
steps:
  - name: Install Amass
    commands:
      - /bin/su -l {vars.USERNAME} -c 'source /etc/profile.d/go.sh && go install -v github.com/owasp-amass/amass/v4/...@master'
    retries: 1
  - name: Install ffuf
    commands:
      - /bin/su -l {vars.USERNAME} -c 'source /etc/profile.d/go.sh && go install -v github.com/ffuf/ffuf/v2@latest'
  - name: Install puredns
    commands:
      - /bin/su -l {vars.USERNAME} -c 'source /etc/profile.d/go.sh && go install -v github.com/d3mondev/puredns/v2@latest'
  - name: Install massdns
    commands:
      - git clone --depth 1 https://github.com/blechschmidt/massdns /tmp/massdns
      - cd /tmp/massdns && make && cp bin/massdns /usr/local/bin/
verify:
  - name: Check ffuf
    command: /home/{vars.USERNAME}/go/bin/ffuf --version
    expect: ffuf
  - name: Check puredns
    command: /home/{vars.USERNAME}/go/bin/puredns --version
    expect: puredns
`

	baseSetup := `name: base-setup
description: Base system setup with essential packages
author: fleex
version: "1.0.0"
os:
  supported:
    - ubuntu
    - debian
vars:
  USERNAME: op
steps:
  - name: System Update
    commands:
      - export DEBIAN_FRONTEND=noninteractive
      - apt-get update -qq
      - apt-get upgrade -y -qq
    retries: 2
    timeout: 600
  - name: Install Essential Packages
    commands:
      - apt-get install -y -qq git curl wget jq htop tmux vim net-tools
    retries: 2
  - name: Configure Swap
    commands:
      - fallocate -l 2G /swap
      - chmod 600 /swap
      - mkswap /swap
      - swapon /swap
      - echo '/swap none swap sw 0 0' >> /etc/fstab
    continue_on: error
`

	builds := map[string]string{
		"security-tools": securityTools,
		"recon-tools":    reconTools,
		"base-setup":     baseSetup,
	}

	for name, content := range builds {
		buildFile := filepath.Join(buildsPath, name+".yaml")
		if err := os.WriteFile(buildFile, []byte(content), 0644); err != nil {
			continue
		}
	}
}

func createDefaultWorkflows(fleexPath string) {
	workflowsPath := filepath.Join(fleexPath, "workflows")
	if err := os.MkdirAll(workflowsPath, 0755); err != nil {
		return
	}

	quickScan := `name: quick-scan
description: Quick vulnerability scan with nuclei (high/critical only)
author: fleex

steps:
  - name: nuclei
    command: "nuclei -l {INPUT} -severity high,critical -o {OUTPUT}"
    timeout: "1h"

output:
  aggregate: concat
  deduplicate: true
`

	fullRecon := `name: full-recon
description: Complete recon pipeline (httpx -> nuclei)
author: fleex

setup:
  - "nuclei -update-templates -silent"

steps:
  - name: httpx
    command: "httpx -l {INPUT} -silent -o {OUTPUT}"

  - name: nuclei
    command: "nuclei -l {INPUT} -severity medium,high,critical -o {OUTPUT}"
    timeout: "2h"

output:
  aggregate: concat
  deduplicate: true
`

	subdomainEnum := `name: subdomain-enum
description: Subdomain enumeration and probing
author: fleex

steps:
  - name: subfinder
    command: "subfinder -dL {INPUT} -silent -o {OUTPUT}"

  - name: httpx
    command: "httpx -l {INPUT} -silent -o {OUTPUT}"

output:
  aggregate: sort-unique
  deduplicate: true
`

	portScan := `name: port-scan
description: Fast port scanning with masscan
author: fleex

vars:
  PORTS: "1-65535"
  RATE: "10000"

steps:
  - name: masscan
    command: "masscan -iL {INPUT} -p{vars.PORTS} --rate={vars.RATE} -oG {OUTPUT}"
    timeout: "30m"

output:
  aggregate: concat
  deduplicate: false
`

	megScan := `name: meg-scan
description: Fast endpoint discovery with meg
author: fleex

vars:
  ENDPOINTS: "/tmp/endpoints.txt"
  CONCURRENCY: "200"

setup:
  - "go install github.com/tomnomnom/meg@latest"

steps:
  - name: meg
    command: "meg -c {vars.CONCURRENCY} -v {vars.ENDPOINTS} {INPUT} {OUTPUT}"
    timeout: "1h"

output:
  aggregate: concat
  deduplicate: false
`

	ffufFuzz := `name: ffuf-fuzz
description: Directory fuzzing with ffuf
author: fleex

vars:
  WORDLIST: "/usr/share/wordlists/dirb/common.txt"

steps:
  - name: ffuf
    command: "ffuf -u FUZZ -w {INPUT}:URL -w {vars.WORDLIST}:WORD -of csv -o {OUTPUT}"
    timeout: "2h"

output:
  aggregate: concat
  deduplicate: true
`

	workflows := map[string]string{
		"quick-scan":     quickScan,
		"full-recon":     fullRecon,
		"subdomain-enum": subdomainEnum,
		"port-scan":      portScan,
		"meg-scan":       megScan,
		"ffuf-fuzz":      ffufFuzz,
	}

	for name, content := range workflows {
		workflowFile := filepath.Join(workflowsPath, name+".yaml")
		if err := os.WriteFile(workflowFile, []byte(content), 0644); err != nil {
			continue
		}
	}
}

func runAddProvider(providerName string, fleexPath string) {
	validProviders := []string{"linode", "digitalocean", "vultr"}
	isValid := false
	for _, p := range validProviders {
		if providerName == p {
			isValid = true
			break
		}
	}
	if !isValid {
		utils.Log.Fatalf("Invalid provider: %s. Valid providers: linode, digitalocean, vultr", providerName)
	}

	configPath := filepath.Join(fleexPath, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		utils.Log.Fatal("Configuration file not found. Run 'fleex init' first")
	}

	configFile, err := os.ReadFile(configPath)
	if err != nil {
		utils.Log.Fatal(err)
	}

	var config models.Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		utils.Log.Fatal(err)
	}

	if config.Providers == nil {
		config.Providers = make(map[string]models.Provider)
	}

	if _, exists := config.Providers[providerName]; exists {
		utils.Log.Fatalf("Provider '%s' already exists in configuration", providerName)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n--- Adding %s Provider ---\n", providerName)

	newProvider := configureProvider(reader, providerName)
	config.Providers[providerName] = newProvider

	file, err := os.Create(configPath)
	if err != nil {
		utils.Log.Fatal(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		utils.Log.Fatal(err)
	}

	fmt.Printf("\nProvider '%s' added successfully\n", providerName)
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolP("wizard", "w", false, "Run interactive setup wizard")
	initCmd.Flags().BoolP("overwrite", "o", false, "Overwrite existing configuration")
	initCmd.Flags().StringP("url", "u", "", "Config folder url (deprecated)")
	initCmd.Flags().StringP("email", "e", "", "Email for SSH key generation")
	initCmd.Flags().String("add-provider", "", "Add a new provider to existing config (linode, digitalocean, vultr)")
	_ = time.Now()
}
