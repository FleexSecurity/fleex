package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [fleet-name]",
	Short: "Show detailed fleet status",
	Long: `Display detailed status of a fleet or all instances.

Examples:
  fleex status           # Show all instances grouped by fleet
  fleex status myfleet   # Show specific fleet details
  fleex status --summary # Show summary only`,
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		summaryOnly, _ := cmd.Flags().GetBool("summary")
		providerFlag, _ := cmd.Flags().GetString("provider")

		if providerFlag != "" {
			globalConfig.Settings.Provider = providerFlag
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		newController := controller.NewController(globalConfig)

		boxes, err := newController.Service.GetBoxes()
		if err != nil {
			utils.Log.Fatal(err)
		}

		if len(boxes) == 0 {
			fmt.Println("No instances found.")
			return
		}

		var fleetFilter string
		if len(args) > 0 {
			fleetFilter = args[0]
		}

		fleets := make(map[string][]struct {
			ID     string
			Label  string
			Status string
			IP     string
		})

		for _, box := range boxes {
			if fleetFilter != "" && !utils.MatchesFleetName(box.Label, fleetFilter) {
				continue
			}

			fleetName := extractFleetName(box.Label)
			fleets[fleetName] = append(fleets[fleetName], struct {
				ID     string
				Label  string
				Status string
				IP     string
			}{
				ID:     box.ID,
				Label:  box.Label,
				Status: box.Status,
				IP:     box.IP,
			})
		}

		if len(fleets) == 0 {
			if fleetFilter != "" {
				fmt.Printf("No instances found for fleet '%s'\n", fleetFilter)
			} else {
				fmt.Println("No instances found.")
			}
			return
		}

		fmt.Printf("\n=== FLEET STATUS ===\n")
		fmt.Printf("Provider: %s\n\n", globalConfig.Settings.Provider)

		if summaryOnly {
			printSummary(fleets)
			return
		}

		for fleetName, instances := range fleets {
			running := 0
			total := len(instances)

			for _, inst := range instances {
				if inst.Status == "running" || inst.Status == "active" {
					running++
				}
			}

			fmt.Printf("Fleet: %s (%d/%d running)\n", fleetName, running, total)

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Label", "Status", "IP"})
			table.SetBorder(false)

			for _, inst := range instances {
				status := inst.Status
				if status == "running" || status == "active" {
					status = "RUNNING"
				} else {
					status = strings.ToUpper(status)
				}
				table.Append([]string{inst.Label, status, inst.IP})
			}

			table.Render()
			fmt.Println()
		}

		printSummary(fleets)
	},
}

func extractFleetName(label string) string {
	parts := strings.Split(label, "-")
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	return label
}

func printSummary(fleets map[string][]struct {
	ID     string
	Label  string
	Status string
	IP     string
}) {
	totalInstances := 0
	totalRunning := 0

	for _, instances := range fleets {
		totalInstances += len(instances)
		for _, inst := range instances {
			if inst.Status == "running" || inst.Status == "active" {
				totalRunning++
			}
		}
	}

	fmt.Println("=== SUMMARY ===")
	fmt.Printf("Total Fleets:    %d\n", len(fleets))
	fmt.Printf("Total Instances: %d\n", totalInstances)
	fmt.Printf("Running:         %d\n", totalRunning)
	fmt.Printf("Other:           %d\n", totalInstances-totalRunning)

	if globalConfig != nil {
		provider := globalConfig.Settings.Provider
		if provInfo, ok := globalConfig.Providers[provider]; ok {
			hourlyCost := getHourlyCost(provider, provInfo.Size)
			fmt.Printf("\nEstimated hourly cost: $%.4f\n", hourlyCost*float64(totalRunning))
		}
	}
}

func getHourlyCost(provider, size string) float64 {
	costs := map[string]map[string]float64{
		"linode": {
			"g6-nanode-1":   0.0075,
			"g6-standard-1": 0.018,
			"g6-standard-2": 0.036,
		},
		"digitalocean": {
			"s-1vcpu-1gb": 0.00744,
			"s-1vcpu-2gb": 0.01488,
			"s-2vcpu-2gb": 0.02679,
		},
		"vultr": {
			"vc2-1c-1gb": 0.006,
			"vc2-1c-2gb": 0.012,
			"vc2-2c-4gb": 0.024,
		},
	}

	if provCosts, ok := costs[provider]; ok {
		if cost, ok := provCosts[size]; ok {
			return cost
		}
	}
	return 0.01
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolP("summary", "s", false, "Show summary only")
	statusCmd.Flags().StringP("provider", "p", "", "Cloud provider")
}
