package cmdx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/local"
	"github.com/Olyxz16/tkt/internal/store"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a local project with SQLite store",
	Run: func(cmd *cobra.Command, args []string) {
		wd, err := os.Getwd()
		if err != nil {
			handleError(err)
		}

		dbDir := filepath.Join(wd, ".tkt")
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

		// Provider configuration
		if localCfg.Provider.URL == "" {
			var providerName, providerURL string
			fields := []huh.Field{
				huh.NewInput().
					Title("Provider name (e.g., youtrack, github)").
					Placeholder("youtrack").
					Value(&providerName),
				huh.NewInput().
					Title("Provider URL").
					Placeholder("https://company.youtrack.cloud").
					Value(&providerURL),
			}

			form := huh.NewForm(huh.NewGroup(fields...))
			if err := form.Run(); err == nil {
				providerURL = strings.TrimSuffix(providerURL, "/")
				localCfg.Provider = config.ProviderConfig{
					Name: providerName,
					URL:  providerURL,
				}
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
		entry := ".tkt.local.yml\n"
		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			os.WriteFile(gitignorePath, []byte(entry), 0644)
		} else if !contains(string(data), ".tkt.local.yml") {
			f, _ := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
			f.WriteString(entry)
			f.Close()
		}

		fmt.Println("Initialized local project in .tkt/")
		fmt.Println("  - SQLite database: .tkt/store.db")
		fmt.Println("  - Config: .tktrc.yml")
		fmt.Println("  - Private config: .tkt.local.yml")
		if localCfg.Provider.Name != "" {
			fmt.Printf("  - Provider: %s (%s)\n", localCfg.Provider.Name, localCfg.Provider.URL)
		}
		if localCfg.Project != "" {
			fmt.Printf("  - Project: %s\n", localCfg.Project)
		}
	},
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

func init() {
	rootCmd.AddCommand(initCmd)
}