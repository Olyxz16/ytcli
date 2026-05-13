package cmdx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize local config files in current directory",
	Run: func(cmd *cobra.Command, args []string) {
		wd, err := os.Getwd()
		if err != nil {
			handleError(err)
		}

		// Create commitable config
		local := &config.LocalConfig{}
		if err := config.SaveLocal(local, wd); err != nil {
			handleError(err)
		}

		// Create private config
		private := &config.LocalPrivateConfig{}
		if err := config.SaveLocalPrivate(private, wd); err != nil {
			handleError(err)
		}

		// Add to .gitignore
		gitignorePath := filepath.Join(wd, ".gitignore")
		entry := ".tkt.local.yml\n"
		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			if err := os.WriteFile(gitignorePath, []byte(entry), 0644); err != nil {
				handleError(err)
			}
		} else {
			content := string(data)
			if !strings.Contains(content, ".tkt.local.yml") {
				f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					handleError(err)
				}
				defer f.Close()
				if _, err := f.WriteString(entry); err != nil {
					handleError(err)
				}
			}
		}

		fmt.Println("Initialized .tktrc.yml and .tkt.local.yml")
		fmt.Println("Added .tkt.local.yml to .gitignore")
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		local, path, err := config.LoadLocal()
		if err != nil {
			handleError(err)
		}
		if path == "" {
			wd, _ := os.Getwd()
			path = wd
		} else {
			path = filepath.Dir(path)
		}

		switch key {
		case "provider.name":
			local.Provider.Name = value
		case "provider.url":
			local.Provider.URL = value
		case "instance":
			// Backwards compatibility: maps to provider.name
			local.Provider.Name = value
		case "instance_url":
			// Backwards compatibility: maps to provider.url
			local.Provider.URL = value
		case "project":
			local.Project = value
		case "default_query":
			local.DefaultQuery = value
		case "wiki_dir":
			local.WikiDir = value
		default:
			fmt.Fprintf(os.Stderr, "Unknown config key: %s\n", key)
			os.Exit(1)
		}

		if err := config.SaveLocal(local, path); err != nil {
			handleError(err)
		}
		fmt.Println("Config updated")
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		local, _, _ := config.LoadLocal()
		private, _, _ := config.LoadLocalPrivate()
		merged, _ := config.Resolve(local, private)

		switch key {
		case "provider.name", "instance":
			fmt.Println(merged.ProviderName)
		case "provider.url", "instance_url":
			fmt.Println(merged.ProviderURL)
		case "project":
			fmt.Println(merged.Project)
		case "default_query":
			fmt.Println(merged.DefaultQuery)
		case "wiki_dir":
			fmt.Println(merged.WikiDir)
		case "current_task":
			fmt.Println(merged.CurrentTask)
		default:
			fmt.Fprintf(os.Stderr, "Unknown config key: %s\n", key)
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	rootCmd.AddCommand(configCmd)
}