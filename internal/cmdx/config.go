package cmdx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/render"
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
		entry := ".ytcli.local.yml\n"
		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			// Create .gitignore
			if err := os.WriteFile(gitignorePath, []byte(entry), 0644); err != nil {
				handleError(err)
			}
		} else {
			content := string(data)
			if !strings.Contains(content, ".ytcli.local.yml") {
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

		fmt.Println("Initialized .ytcli.yml and .ytcli.local.yml")
		fmt.Println("Added .ytcli.local.yml to .gitignore")
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		global, err := config.LoadGlobal()
		if err != nil {
			handleError(err)
		}

		switch key {
		case "default_instance":
			global.DefaultInstance = value
		case "output_format":
			global.OutputFormat = value
		default:
			if strings.HasPrefix(key, "instances.") {
				parts := strings.SplitN(key[10:], ".", 2)
				if len(parts) == 2 && parts[1] == "url" {
					if global.Instances == nil {
						global.Instances = make(map[string]config.InstanceConfig)
					}
					global.Instances[parts[0]] = config.InstanceConfig{URL: value}
				} else {
					fmt.Fprintf(os.Stderr, "Unknown config key: %s\n", key)
					os.Exit(1)
				}
			} else {
				// Try local config
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
case "instance":
				if _, ok := global.Instances[value]; !ok && len(global.Instances) > 0 {
					fmt.Fprintf(os.Stderr, "Warning: instance %q not found in global config. Available instances:\n", value)
					for name, inst := range global.Instances {
						fmt.Fprintf(os.Stderr, "  - %s: %s\n", name, inst.URL)
					}
				}
				local.Instance = value
				case "project":
					local.Project = value
				case "default_query":
					local.DefaultQuery = value
				default:
					fmt.Fprintf(os.Stderr, "Unknown config key: %s\n", key)
					os.Exit(1)
				}
				if err := config.SaveLocal(local, path); err != nil {
					handleError(err)
				}
				fmt.Println("Local config updated")
				return
			}
		}

		if err := config.SaveGlobal(global); err != nil {
			handleError(err)
		}
		fmt.Println("Global config updated")
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		global, _ := config.LoadGlobal()
		local, _, _ := config.LoadLocal()
		private, _, _ := config.LoadLocalPrivate()
		merged, _ := config.Resolve(global, local, private)

		switch key {
		case "instance":
			fmt.Println(merged.Instance)
		case "instance_url":
			fmt.Println(merged.InstanceURL)
		case "project":
			fmt.Println(merged.Project)
		case "default_query":
			fmt.Println(merged.DefaultQuery)
		case "current_task":
			fmt.Println(merged.CurrentTask)
		case "output_format":
			fmt.Println(merged.OutputFormat)
		default:
			fmt.Fprintf(os.Stderr, "Unknown config key: %s\n", key)
			os.Exit(1)
		}
	},
}

var configInstancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "List configured instances",
	Run: func(cmd *cobra.Command, args []string) {
		global, err := config.LoadGlobal()
		if err != nil {
			handleError(err)
		}
		local, _, _ := config.LoadLocal()

		activeInstance := global.DefaultInstance
		if local != nil && local.Instance != "" {
			activeInstance = local.Instance
		}

		if getOutputMode() == render.OutputJSON {
			type instanceInfo struct {
				Name     string `json:"name"`
				URL      string `json:"url"`
				Active   bool   `json:"active"`
				Default  bool   `json:"default"`
			}
			var instances []instanceInfo
			for name, inst := range global.Instances {
				instances = append(instances, instanceInfo{
					Name:    name,
					URL:     inst.URL,
					Active:  name == activeInstance,
					Default: name == global.DefaultInstance,
				})
			}
			render.JSON(instances)
			return
		}

		for name, inst := range global.Instances {
			marker := " "
			if name == activeInstance {
				marker = "*"
			}
			fmt.Printf("%s %s: %s\n", marker, name, inst.URL)
		}
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configInstancesCmd)
	rootCmd.AddCommand(configCmd)
}
