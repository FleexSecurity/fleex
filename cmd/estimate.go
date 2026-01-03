package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
)

type ProviderPricing struct {
	Name       string
	HourlyCost float64
	Size       string
}

var providerPricing = map[string][]ProviderPricing{
	"linode": {
		{Name: "Nanode 1GB", HourlyCost: 0.0075, Size: "g6-nanode-1"},
		{Name: "Linode 2GB", HourlyCost: 0.018, Size: "g6-standard-1"},
		{Name: "Linode 4GB", HourlyCost: 0.036, Size: "g6-standard-2"},
	},
	"digitalocean": {
		{Name: "s-1vcpu-1gb", HourlyCost: 0.00744, Size: "s-1vcpu-1gb"},
		{Name: "s-1vcpu-2gb", HourlyCost: 0.01488, Size: "s-1vcpu-2gb"},
		{Name: "s-2vcpu-2gb", HourlyCost: 0.02679, Size: "s-2vcpu-2gb"},
	},
	"vultr": {
		{Name: "vc2-1c-1gb", HourlyCost: 0.006, Size: "vc2-1c-1gb"},
		{Name: "vc2-1c-2gb", HourlyCost: 0.012, Size: "vc2-1c-2gb"},
		{Name: "vc2-2c-4gb", HourlyCost: 0.024, Size: "vc2-2c-4gb"},
	},
}

var toolEstimates = map[string]float64{
	"nuclei":    0.5,
	"httpx":     0.2,
	"subfinder": 0.3,
	"masscan":   0.1,
	"nmap":      1.0,
	"ffuf":      0.8,
	"puredns":   0.4,
	"amass":     2.0,
	"default":   0.5,
}

var estimateCmd = &cobra.Command{
	Use:   "estimate",
	Short: "Estimate scan cost before running",
	Long: `Estimate the cost and time for a distributed scan.

Examples:
  fleex estimate -t targets.txt -i 10
  fleex estimate -t domains.txt --tool nuclei -i 50
  fleex estimate -t ips.txt --tool masscan -p digitalocean`,
	Run: func(cmd *cobra.Command, args []string) {
		targetsFile, _ := cmd.Flags().GetString("targets")
		instances, _ := cmd.Flags().GetInt("instances")
		tool, _ := cmd.Flags().GetString("tool")
		provider, _ := cmd.Flags().GetString("provider")
		duration, _ := cmd.Flags().GetFloat64("duration")

		if targetsFile == "" {
			utils.Log.Fatal("--targets flag is required")
		}

		if provider == "" && globalConfig != nil {
			provider = globalConfig.Settings.Provider
		}
		if provider == "" {
			provider = "linode"
		}

		targetCount := countLines(targetsFile)
		if targetCount == 0 {
			utils.Log.Fatal("No targets found in file")
		}

		pricing := getProviderPricing(provider)

		if duration == 0 {
			duration = estimateDuration(targetCount, instances, tool)
		}

		totalCost := pricing.HourlyCost * float64(instances) * duration
		bufferCost := totalCost * 1.12

		fmt.Println("\n=== COST ESTIMATE ===\n")
		fmt.Printf("Targets:     %d\n", targetCount)
		fmt.Printf("Instances:   %d\n", instances)
		fmt.Printf("Tool:        %s\n", tool)
		fmt.Printf("Provider:    %s\n", provider)
		fmt.Printf("Instance:    %s @ $%.5f/hour\n", pricing.Size, pricing.HourlyCost)
		fmt.Println()
		fmt.Printf("Duration:    ~%.1f minutes\n", duration*60)
		fmt.Printf("Rate:        ~%.0f targets/min\n", float64(targetCount)/(duration*60))
		fmt.Println()
		fmt.Printf("Base cost:   $%.2f\n", totalCost)
		fmt.Printf("With buffer: $%.2f (+12%%)\n", bufferCost)
		fmt.Println()

		if bufferCost < 1 {
			fmt.Println("Cost level: LOW")
		} else if bufferCost < 10 {
			fmt.Println("Cost level: MODERATE")
		} else if bufferCost < 50 {
			fmt.Println("Cost level: HIGH")
		} else {
			fmt.Println("Cost level: VERY HIGH - Consider reducing instances")
		}

		fmt.Println("\nTo proceed, run:")
		fmt.Printf("  fleex spawn -n scan -c %d && fleex scan -n scan -i %s -o results.txt\n", instances, targetsFile)
	},
}

func countLines(filename string) int {
	file, err := os.Open(filename)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}
	return count
}

func getProviderPricing(provider string) ProviderPricing {
	if prices, ok := providerPricing[provider]; ok {
		if globalConfig != nil {
			configSize := globalConfig.Providers[provider].Size
			for _, p := range prices {
				if p.Size == configSize {
					return p
				}
			}
		}
		return prices[0]
	}
	return ProviderPricing{Name: "Unknown", HourlyCost: 0.01, Size: "default"}
}

func estimateDuration(targets, instances int, tool string) float64 {
	rate, ok := toolEstimates[tool]
	if !ok {
		rate = toolEstimates["default"]
	}

	targetsPerInstance := float64(targets) / float64(instances)
	minutes := targetsPerInstance * rate
	hours := minutes / 60

	if hours < 0.01 {
		hours = 0.01
	}

	return hours
}

func init() {
	rootCmd.AddCommand(estimateCmd)

	estimateCmd.Flags().StringP("targets", "t", "", "File with targets to scan")
	estimateCmd.Flags().IntP("instances", "i", 10, "Number of instances to use")
	estimateCmd.Flags().StringP("tool", "", "nuclei", "Tool to use (nuclei, masscan, httpx, etc)")
	estimateCmd.Flags().StringP("provider", "p", "", "Cloud provider")
	estimateCmd.Flags().Float64P("duration", "d", 0, "Override estimated duration (hours)")
}
