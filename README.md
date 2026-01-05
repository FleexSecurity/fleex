<p align="center">
  <img src="static/Fleex-Banner.png" alt="Fleex">
</p>

<p align="center">
  <b>Distributed workload execution across cloud VPS fleets</b>
</p>

<p align="center">
  <a href="https://github.com/FleexSecurity/fleex/releases"><img src="https://img.shields.io/github/v/release/FleexSecurity/fleex?color=blue&style=flat-square"></a>
  <a href="https://github.com/FleexSecurity/fleex/actions"><img src="https://img.shields.io/github/actions/workflow/status/FleexSecurity/fleex/release_binary.yml?style=flat-square"></a>
  <a href="https://goreportcard.com/report/github.com/FleexSecurity/fleex"><img src="https://goreportcard.com/badge/github.com/FleexSecurity/fleex?style=flat-square"></a>
  <a href="https://github.com/FleexSecurity/fleex/blob/main/LICENSE"><img src="https://img.shields.io/github/license/FleexSecurity/fleex?style=flat-square"></a>
  <a href="https://github.com/FleexSecurity/fleex/issues"><img src="https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat-square"></a>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#usage">Usage</a> •
  <a href="#documentation">Documentation</a> •
  <a href="#supported-providers">Providers</a>
</p>

---

Fleex spawns fleets of VPS instances across multiple cloud providers, distributes workloads via SSH, and aggregates results. Scale your security tools like **masscan**, **nuclei**, **httpx**, **ffuf**, and **puredns** across hundreds of machines in minutes.

## Features

- **Multi-Provider Support** - Linode, DigitalOcean, Vultr, and custom VMs
- **Fleet Management** - Spawn, scale, and destroy fleets with simple commands
- **Distributed Scanning** - Automatically split input files and distribute across fleet
- **Build System** - Provision instances with pre-configured tool recipes
- **Workflow Pipelines** - Chain multiple tools together in distributed pipelines
- **Result Aggregation** - Automatically collect and merge results from all instances
- **Cost Estimation** - Estimate costs before running scans
- **Snapshot Support** - Create reusable images from provisioned instances

## Installation

### Using Go

```bash
go install github.com/FleexSecurity/fleex@latest
```

### From Source

```bash
git clone https://github.com/FleexSecurity/fleex.git
cd fleex
go build -o fleex .
```

### Pre-built Binaries

Download from [Releases](https://github.com/FleexSecurity/fleex/releases)

## Quick Start

### 1. Initialize Configuration

```bash
# Interactive setup wizard
fleex init --wizard

# Or quick setup
fleex init
```

### 2. Spawn a Fleet

```bash
# Spawn 10 instances named "scan"
fleex spawn -n scan -c 10

# Spawn with automatic provisioning
fleex spawn -n scan -c 10 --build security-tools
```

### 3. Run a Distributed Scan

```bash
# Run nuclei across fleet
fleex scan -n scan -i targets.txt -c "nuclei -l {vars.INPUT} -o {vars.OUTPUT}"

# Or use a workflow
fleex workflow run -w quick-scan -n scan -i targets.txt -o results.txt
```

### 4. Clean Up

```bash
# Delete fleet when done
fleex delete -n scan
```

## Usage

### Fleet Management

```bash
fleex spawn -n <name> -c <count>    # Create fleet
fleex ls                             # List all instances
fleex status -n <name>               # Detailed fleet status
fleex delete -n <name>               # Delete fleet
```

### Build & Provision

```bash
fleex build list                     # List available recipes
fleex build show <recipe>            # Show recipe details
fleex build run -r <recipe> -n <fleet>  # Provision fleet
fleex build verify -r <recipe> -n <fleet>  # Verify installation
```

### Scanning

```bash
# Direct command
fleex scan -n <fleet> -i input.txt -c "command {vars.INPUT} -o {vars.OUTPUT}"

# Using module file
fleex scan -n <fleet> -m module.yaml

# Using workflow
fleex scan -n <fleet> -w <workflow> -i input.txt -o output.txt
```

### Horizontal vs Vertical Scaling

Fleex supports two scaling modes:

| Mode | Splits | Use Case |
|------|--------|----------|
| **Horizontal** (default) | Target list | Scan many targets across fleet |
| **Vertical** | Wordlist | Attack single target with distributed wordlist |

```bash
# Horizontal: split 1000 domains across 10 machines (100 each)
fleex scan -n scan -i domains.txt -c "subfinder -d {vars.INPUT} -o {vars.OUTPUT}"

# Vertical: all machines attack tesla.com with different wordlist chunks
fleex scan -n scan --vertical --split-var WORDLIST \
  -c "puredns bruteforce {vars.WORDLIST} tesla.com -o {vars.OUTPUT}" \
  -p WORDLIST:wordlist.txt -p OUTPUT:results.txt -o results.txt
```

Workflows support per-step scale modes with step references:

```yaml
steps:
  - name: Subdomain enumeration
    id: subfinder
    command: subfinder -d {INPUT} -o {OUTPUT}

  - name: HTTP probing
    command: httpx -l {subfinder.OUTPUT} -o {OUTPUT}

  - name: Directory fuzzing
    scale-mode: vertical
    split-var: WORDLIST
    command: ffuf -u {INPUT}/FUZZ -w {vars.WORDLIST} -o {OUTPUT}
```

### Remote Operations

```bash
fleex ssh -n <box-name>              # SSH into instance
fleex scp -n <fleet> -s local -d /remote/path  # Copy files
fleex run -n <fleet> -c "command"    # Execute on all instances
```

### Utilities

```bash
fleex estimate -n <fleet> -h 2       # Estimate cost for 2 hours
fleex images list                    # List available images
fleex images create -n <box> -l <label>  # Create snapshot
```

## Configuration

Configuration is stored in `~/.config/fleex/config.json`:

```json
{
  "providers": {
    "linode": {
      "token": "your-api-token",
      "region": "us-east",
      "size": "g6-nanode-1",
      "image": "linode/ubuntu22.04",
      "port": 22,
      "username": "root"
    }
  },
  "ssh_keys": {
    "public_file": "~/.config/fleex/ssh/id_rsa.pub",
    "private_file": "~/.config/fleex/ssh/id_rsa"
  },
  "settings": {
    "provider": "linode"
  }
}
```

### Adding Providers

```bash
fleex init --add-provider digitalocean
fleex init --add-provider vultr
```

## Supported Providers

| Provider | Status | Notes |
|----------|--------|-------|
| [Linode](https://www.linode.com) | Full Support | Recommended |
| [DigitalOcean](https://www.digitalocean.com) | Full Support | |
| [Vultr](https://www.vultr.com) | Full Support | |
| Custom VMs | Full Support | Bring your own servers |

## Documentation

<a href="https://fleexsecurity.github.io/fleex-docs/">
  <img src="static/Fleex-docs.png" alt="Fleex Documentation" width="400">
</a>

Full documentation available at [fleexsecurity.github.io/fleex-docs](https://fleexsecurity.github.io/fleex-docs/)

## Project Structure

```
fleex/
├── cmd/           # CLI commands (Cobra)
├── pkg/
│   ├── controller/  # Business logic
│   ├── services/    # Provider implementations
│   ├── provider/    # Provider interface
│   ├── models/      # Data structures
│   ├── sshutils/    # SSH/SCP operations
│   ├── utils/       # Utilities
│   └── ui/          # Terminal UI components
└── configs/       # Default configurations
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Support the Project

If you find Fleex useful, consider supporting the developers:

<a href="https://www.buymeacoffee.com/xm1k3">
  <img src="https://www.buymeacoffee.com/assets/img/custom_images/purple_img.png" alt="Buy Me A Coffee">
</a>

### Cloud Provider Referrals

Support the project by using our referral links:

<p>
  <a href="https://www.digitalocean.com/?refcode=91982e64054b&utm_campaign=Referral_Invite&utm_medium=Referral_Program&utm_source=badge">
    <img src="static/Referrals/Digitalocean-referral.png" alt="DigitalOcean" height="40">
  </a>
  <a href="https://www.linode.com/?r=172cb6708bc78a41c5014cc2da0f2ab0d7abbe7b">
    <img src="static/Referrals/Linode-referral.png" alt="Linode" height="40">
  </a>
  <a href="https://vultr.com/?ref=8969285-8H">
    <img src="static/Referrals/Vultr-referral.png" alt="Vultr" height="40">
  </a>
</p>

## Authors

<table>
  <tr>
    <td align="center">
      <a href="https://github.com/sw33tLie">
        <img width="75px;" src="https://avatars.githubusercontent.com/u/47645560?v=4" alt="sw33tLie"/>
        <br />
        <b>sw33tLie</b>
      </a>
      <br />
      <a href="https://twitter.com/sw33tLie"><img src="https://img.shields.io/twitter/follow/sw33tLie?style=social"></a>
    </td>
    <td align="center">
      <a href="https://github.com/xm1k3">
        <img width="75px;" src="https://avatars.githubusercontent.com/u/73166077?v=4" alt="xm1k3"/>
        <br />
        <b>xm1k3</b>
      </a>
      <br />
      <a href="https://twitter.com/xm1k3_"><img src="https://img.shields.io/twitter/follow/xm1k3_?style=social"></a>
    </td>
  </tr>
</table>

## License

Fleex is distributed under the [Apache-2.0 License](LICENSE).
