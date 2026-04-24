package cmdx

import (
	"fmt"
	"os"
	"path/filepath"

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

		// Open/create the database
		db, err := store.Open()
		if err != nil {
			handleError(err)
		}
		db.Close()

		// Create/update .ytcli.yml with local schema defaults
		localCfg, path, err := config.LoadLocal()
		if err != nil {
			handleError(err)
		}
		if path == "" {
			path = wd
		} else {
			path = filepath.Dir(path)
		}

		if len(localCfg.Local.States) == 0 {
			localCfg.Local = local.DefaultSchema
		}

		if err := config.SaveLocal(localCfg, path); err != nil {
			handleError(err)
		}

		// Ensure .ytcli.local.yml exists and .gitignore includes it
		private := &config.LocalPrivateConfig{}
		if err := config.SaveLocalPrivate(private, path); err != nil {
			handleError(err)
		}

		gitignorePath := filepath.Join(path, ".gitignore")
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
