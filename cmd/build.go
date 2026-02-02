package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FleexSecurity/fleex/pkg/controller"
	"github.com/FleexSecurity/fleex/pkg/models"
	"github.com/FleexSecurity/fleex/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build and provision fleet instances with tools",
	Long: `Build provisions fleet instances with specified tools and configurations.

Examples:
  fleex build list                              # List available build recipes
  fleex build show security-tools               # Show recipe details
  fleex build run -r security-tools -n pwn      # Build existing fleet
  fleex build verify -r security-tools -n pwn   # Verify installation`,
}

var buildListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available build recipes",
	Run: func(cmd *cobra.Command, args []string) {
		recipes, err := utils.ListBuildRecipes()
		if err != nil {
			utils.Log.Fatal(err)
		}

		if len(recipes) == 0 {
			fmt.Println("No build recipes found. Run 'fleex init' to create default recipes.")
			return
		}

		fmt.Println("\n=== AVAILABLE BUILD RECIPES ===\n")
		fmt.Printf("%-25s %-50s\n", "NAME", "DESCRIPTION")
		fmt.Printf("%-25s %-50s\n", strings.Repeat("-", 25), strings.Repeat("-", 50))

		for _, name := range recipes {
			recipe, err := utils.ReadBuildFile(name)
			if err != nil {
				continue
			}
			fmt.Printf("%-25s %-50s\n", name, recipe.Description)
		}
		fmt.Println()
	},
}

var buildShowCmd = &cobra.Command{
	Use:   "show [recipe-name]",
	Short: "Show build recipe details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		recipeName := args[0]

		recipe, err := utils.ReadBuildFile(recipeName)
		if err != nil {
			utils.Log.Fatal("Recipe not found: ", recipeName)
		}

		fmt.Printf("\n=== %s ===\n\n", strings.ToUpper(recipe.Name))
		fmt.Printf("Description: %s\n", recipe.Description)
		fmt.Printf("Author:      %s\n", recipe.Author)
		fmt.Printf("Version:     %s\n", recipe.Version)

		if len(recipe.OS.Supported) > 0 {
			fmt.Printf("OS:          %s\n", strings.Join(recipe.OS.Supported, ", "))
		}

		if len(recipe.Vars) > 0 {
			fmt.Println("\nVariables:")
			for k, v := range recipe.Vars {
				if v == "" {
					fmt.Printf("  %s: (required)\n", k)
				} else {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}
		}

		if len(recipe.Files) > 0 {
			fmt.Println("\nFiles:")
			for _, f := range recipe.Files {
				fmt.Printf("  %s -> %s\n", f.Source, f.Destination)
			}
		}

		fmt.Println("\nSteps:")
		for i, step := range recipe.Steps {
			retries := ""
			if step.Retries > 0 {
				retries = fmt.Sprintf(" (retries: %d)", step.Retries)
			}
			fmt.Printf("  %d. %s%s\n", i+1, step.Name, retries)
			for _, cmd := range step.Commands {
				if len(cmd) > 60 {
					fmt.Printf("     $ %s...\n", cmd[:60])
				} else {
					fmt.Printf("     $ %s\n", cmd)
				}
			}
		}

		if len(recipe.Verify) > 0 {
			fmt.Println("\nVerification:")
			for _, v := range recipe.Verify {
				fmt.Printf("  - %s: %s\n", v.Name, v.Command)
			}
		}
		fmt.Println()
	},
}

var buildRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run build on a fleet",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		recipeName, _ := cmd.Flags().GetString("recipe")
		recipeFile, _ := cmd.Flags().GetString("file")
		fleetName, _ := cmd.Flags().GetString("name")
		sizeFlag, _ := cmd.Flags().GetString("size")
		parallel, _ := cmd.Flags().GetInt("parallel")
		noVerify, _ := cmd.Flags().GetBool("no-verify")
		continueErr, _ := cmd.Flags().GetBool("continue")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		verbose, _ := cmd.Flags().GetBool("verbose")

		if recipeName == "" && recipeFile == "" {
			utils.Log.Fatal("Either --recipe or --file is required")
		}

		if fleetName == "" {
			utils.Log.Fatal("--name flag is required")
		}

		var recipe *models.BuildRecipe
		var err error

		if recipeFile != "" {
			recipe, err = utils.ReadBuildFile(recipeFile)
		} else {
			recipe, err = utils.ReadBuildFile(recipeName)
		}

		if err != nil {
			utils.Log.Fatal("Failed to load recipe: ", err)
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		providerName := globalConfig.Settings.Provider
		if sizeFlag != "" {
			providerInfo := globalConfig.Providers[providerName]
			providerInfo.Size = sizeFlag
			globalConfig.Providers[providerName] = providerInfo
		}

		newController := controller.NewController(globalConfig)

		snapshot, _ := cmd.Flags().GetBool("snapshot")

		fleet := newController.GetFleet(fleetName)
		fleetExisted := len(fleet) > 0

		if len(fleet) == 0 {
			fmt.Printf("Spawning 1 instance for fleet '%s'...\n", fleetName)
			newController.SpawnFleet(fleetName, 1, false, false)
			fleet = newController.GetFleet(fleetName)
		}

		if len(fleet) == 0 {
			utils.Log.Fatal("Failed to get fleet after spawn")
		}

		createSnapshot := func() {
			now := time.Now()
			snapshotName := fmt.Sprintf("fleex-%s-%s", recipe.Name, now.Format("02-01-2006-15-04"))
			fmt.Printf("Creating snapshot '%s' from %s (ID: %s)...\n", snapshotName, fleet[0].Label, fleet[0].ID)

			boxID, _ := strconv.Atoi(fleet[0].ID)
			err := newController.Service.CreateImage(boxID, snapshotName)

			if err != nil {
				utils.Log.Error("Failed to create snapshot: ", err)
			} else {
				fmt.Printf("Snapshot '%s' created successfully\n", snapshotName)
			}
		}

		if fleetExisted && snapshot {
			fmt.Printf("Fleet '%s' already exists. Creating snapshot...\n", fleetName)
			createSnapshot()
			return
		}

		fmt.Printf("Building fleet '%s' (%d instances) with recipe '%s'...\n", fleetName, len(fleet), recipe.Name)

		opts := models.BuildOptions{
			Recipe:      recipe,
			FleetName:   fleetName,
			Parallel:    parallel,
			NoVerify:    noVerify,
			ContinueErr: continueErr,
			DryRun:      dryRun,
			Verbose:     verbose,
		}

		results, err := newController.BuildFleet(opts)
		if err != nil {
			utils.Log.Fatal(err)
		}

		successCount := 0
		for _, r := range results {
			if r.Success {
				successCount++
			}
		}

		fmt.Printf("\nBuild complete: %d/%d successful\n", successCount, len(results))

		if snapshot && successCount > 0 {
			createSnapshot()
		}
	},
}

var buildVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify build installation on a fleet",
	Run: func(cmd *cobra.Command, args []string) {
		proxy, _ := rootCmd.PersistentFlags().GetString("proxy")
		utils.SetProxy(proxy)

		recipeName, _ := cmd.Flags().GetString("recipe")
		fleetName, _ := cmd.Flags().GetString("name")
		verbose, _ := cmd.Flags().GetBool("verbose")

		if recipeName == "" {
			utils.Log.Fatal("--recipe flag is required")
		}

		if fleetName == "" {
			utils.Log.Fatal("--name flag is required")
		}

		recipe, err := utils.ReadBuildFile(recipeName)
		if err != nil {
			utils.Log.Fatal("Failed to load recipe: ", err)
		}

		if len(recipe.Verify) == 0 {
			utils.Log.Fatal("Recipe has no verification steps")
		}

		provider := controller.GetProvider(globalConfig.Settings.Provider)
		if provider == -1 {
			utils.Log.Fatal(models.ErrInvalidProvider)
		}

		newController := controller.NewController(globalConfig)

		opts := models.BuildOptions{
			Recipe:    recipe,
			FleetName: fleetName,
			Verbose:   verbose,
		}

		results, err := newController.VerifyFleet(opts)
		if err != nil {
			utils.Log.Fatal(err)
		}

		passCount := 0
		for _, passed := range results {
			if passed {
				passCount++
			}
		}

		fmt.Printf("\nVerification complete: %d/%d passed\n", passCount, len(results))
	},
}

var buildCreateCmd = &cobra.Command{
	Use:   "create [recipe-name]",
	Short: "Create a new build recipe",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		recipeName := args[0]
		description, _ := cmd.Flags().GetString("description")
		fromRecipe, _ := cmd.Flags().GetString("from")

		buildsDir, err := utils.GetBuildsDir()
		if err != nil {
			utils.Log.Fatal(err)
		}

		if err := os.MkdirAll(buildsDir, 0755); err != nil {
			utils.Log.Fatal(err)
		}

		var recipe *models.BuildRecipe

		if fromRecipe != "" {
			recipe, err = utils.ReadBuildFile(fromRecipe)
			if err != nil {
				utils.Log.Fatal("Source recipe not found: ", fromRecipe)
			}
			recipe.Name = recipeName
		} else {
			recipe = &models.BuildRecipe{
				Name:        recipeName,
				Description: description,
				Author:      "user",
				Version:     "1.0.0",
				OS: models.OSConfig{
					Supported: []string{"ubuntu", "debian"},
				},
				Vars: map[string]string{
					"USERNAME": "op",
				},
				Steps: []models.BuildStep{
					{
						Name: "Example Step",
						Commands: []string{
							"echo 'Hello from {vars.USERNAME}'",
						},
					},
				},
			}
		}

		if description != "" {
			recipe.Description = description
		}

		recipeFile := filepath.Join(buildsDir, recipeName+".yaml")
		data, err := yaml.Marshal(recipe)
		if err != nil {
			utils.Log.Fatal(err)
		}

		if err := os.WriteFile(recipeFile, data, 0644); err != nil {
			utils.Log.Fatal(err)
		}

		fmt.Printf("Build recipe '%s' created at: %s\n", recipeName, recipeFile)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.AddCommand(buildListCmd)
	buildCmd.AddCommand(buildShowCmd)
	buildCmd.AddCommand(buildRunCmd)
	buildCmd.AddCommand(buildVerifyCmd)
	buildCmd.AddCommand(buildCreateCmd)

	buildRunCmd.Flags().StringP("recipe", "r", "", "Build recipe name")
	buildRunCmd.Flags().StringP("file", "f", "", "Custom recipe file path")
	buildRunCmd.Flags().StringP("name", "n", "", "Fleet name to build")
	buildRunCmd.Flags().StringP("size", "S", "", "Droplet size (overrides config)")
	buildRunCmd.Flags().BoolP("snapshot", "s", false, "Create snapshot after successful build")
	buildRunCmd.Flags().IntP("parallel", "p", 5, "Number of parallel builds")
	buildRunCmd.Flags().BoolP("no-verify", "", false, "Skip verification step")
	buildRunCmd.Flags().BoolP("continue", "", false, "Continue on step failure")
	buildRunCmd.Flags().BoolP("dry-run", "", false, "Show what would be executed")
	buildRunCmd.Flags().BoolP("verbose", "v", false, "Show detailed output")

	buildVerifyCmd.Flags().StringP("recipe", "r", "", "Build recipe name")
	buildVerifyCmd.Flags().StringP("name", "n", "", "Fleet name to verify")
	buildVerifyCmd.Flags().BoolP("verbose", "v", false, "Show detailed output")

	buildCreateCmd.Flags().StringP("description", "", "Custom build recipe", "Recipe description")
	buildCreateCmd.Flags().StringP("from", "", "", "Copy from existing recipe")
}
