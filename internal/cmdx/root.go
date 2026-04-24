package cmdx

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/ytcli/internal/api"
	"github.com/Olyxz16/ytcli/internal/config"
	"github.com/Olyxz16/ytcli/internal/render"
	"github.com/Olyxz16/ytcli/internal/service"
)

var (
	instanceFlag string
	outputFlag   string
	quietFlag    bool
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "ytcli",
	Short: "A fast CLI for YouTrack",
	Long:  `ytcli is a fast, agent-friendly command-line interface for JetBrains YouTrack.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if quietFlag && outputFlag == "" {
			outputFlag = "json"
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&instanceFlag, "instance", "i", "", "YouTrack instance name")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "Output format: table, json, wide, markdown")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Minimal output")
}

// getOutputMode returns the effective output mode.
func getOutputMode() render.OutputMode {
	if outputFlag != "" {
		return render.OutputMode(outputFlag)
	}
	global, _ := config.LoadGlobal()
	if global != nil && global.OutputFormat != "" {
		return render.OutputMode(global.OutputFormat)
	}
	return render.OutputTable
}

// buildService creates a service from resolved configuration.
func buildService() (*service.Service, *config.MergedConfig, error) {
	global, err := config.LoadGlobal()
	if err != nil {
		return nil, nil, fmt.Errorf("load global config: %w", err)
	}
	local, _, err := config.LoadLocal()
	if err != nil {
		return nil, nil, fmt.Errorf("load local config: %w", err)
	}
	private, _, err := config.LoadLocalPrivate()
	if err != nil {
		return nil, nil, fmt.Errorf("load local private config: %w", err)
	}

	merged, err := config.Resolve(global, local, private)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve config: %w", err)
	}

	if instanceFlag != "" {
		merged.Instance = instanceFlag
		inst, ok := global.Instances[merged.Instance]
		if !ok {
			return nil, nil, fmt.Errorf("instance %q not found", merged.Instance)
		}
		merged.InstanceURL = inst.URL
	}

	if merged.InstanceURL == "" {
		return nil, nil, fmt.Errorf("no YouTrack instance configured. Run: ytcli auth login")
	}

	svc, err := service.NewService(merged)
	if err != nil {
		return nil, nil, err
	}
	return svc, merged, nil
}

// handleError prints an error in the appropriate format and exits with a meaningful code.
func handleError(err error) {
	render.Error(err, getOutputMode())
	code := 1
	switch {
	case api.IsAuthError(err):
		code = 2
	case api.IsNotFoundError(err):
		code = 3
	case api.IsValidationError(err):
		code = 4
	}
	os.Exit(code)
}
