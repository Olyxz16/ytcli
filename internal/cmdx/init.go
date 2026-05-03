package cmdx

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/local"
	"github.com/Olyxz16/ytcli/internal/store"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a local project with SQLite store",
	Run: func(cmd *cobra.Command, args []string) {
		wd, err := os.Getwd()
		if err != nil {
			handleError(err)
		}

		dbDir := filepath.Join(wd, ".ytcli")
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			handleError(err)
		}

		db, err := store.Open()
		if err != nil {
			handleError(err)
		}
		db.Close()

		localCfg, path, err := config.LoadLocal()
		if err != nil {
			handleError(err)
		}
		configDir := wd
		if path != "" {
			configDir = filepath.Dir(path)
		}

		if len(localCfg.Local.States) == 0 {
			localCfg.Local = local.DefaultSchema
		}

		global, _ := config.LoadGlobal()
		if global.Instances == nil {
			global.Instances = make(map[string]config.InstanceConfig)
		}

		if localCfg.Instance == "" && len(global.Instances) > 0 {
			instanceNames := sortedKeys(global.Instances)

			var selectedInstance string
			options := make([]huh.Option[string], len(instanceNames))
			for i, name := range instanceNames {
				options[i] = huh.NewOption(fmt.Sprintf("%s (%s)", name, global.Instances[name].URL), name)
			}

			var skipInstance bool
			fields := []huh.Field{
				huh.NewSelect[string]().
					Title("Select YouTrack instance for this project").
					Options(options...).
					Value(&selectedInstance).
					WithHeight(min(len(options)+1, 10)),
			}

			if len(instanceNames) > 0 {
				fields = append(fields, huh.NewConfirm().
					Title("Skip instance binding? (for local-only project)").
					Affirmative("No").
					Negative("Yes").
					Value(&skipInstance))
			}

			form := huh.NewForm(huh.NewGroup(fields...))
			if err := form.Run(); err == nil && !skipInstance && selectedInstance != "" {
				localCfg.Instance = selectedInstance
				localCfg.InstanceURL = strings.TrimSuffix(global.Instances[selectedInstance].URL, "/")
			}
		}

		if localCfg.Project == "" {
			var projectShort string
			prompt := huh.NewInput().
				Title("Project short name (e.g., PROJ)").
				Placeholder("PROJ").
				Value(&projectShort)
			form := huh.NewForm(huh.NewGroup(prompt))
			if err := form.Run(); err == nil {
				projectShort = strings.TrimSpace(projectShort)
				if projectShort != "" {
					localCfg.Project = projectShort
				}
			}
		}

		if err := config.SaveLocal(localCfg, configDir); err != nil {
			handleError(err)
		}

		private := &config.LocalPrivateConfig{}
		if err := config.SaveLocalPrivate(private, configDir); err != nil {
			handleError(err)
		}

		gitignorePath := filepath.Join(configDir, ".gitignore")
		entry := ".ytcli.local.yml\n"
		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			os.WriteFile(gitignorePath, []byte(entry), 0644)
		} else if !contains(string(data), ".ytcli.local.yml") {
			f, _ := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
			f.WriteString(entry)
			f.Close()
		}

		fmt.Println("Initialized local project in .ytcli/")
		fmt.Println("  - SQLite database: .ytcli/store.db")
		fmt.Println("  - Config: .ytcli.yml")
		fmt.Println("  - Private config: .ytcli.local.yml")
		if localCfg.Instance != "" {
			fmt.Printf("  - Instance: %s\n", localCfg.Instance)
		}
		if localCfg.Project != "" {
			fmt.Printf("  - Project: %s\n", localCfg.Project)
		}
	},
}

func sortedKeys(m map[string]config.InstanceConfig) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	rootCmd.AddCommand(initCmd)
}