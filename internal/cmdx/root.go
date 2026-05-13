package cmdx

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"errors"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/provider"
	"github.com/Olyxz16/tkt/internal/render"
	"github.com/Olyxz16/tkt/internal/service"
)

var (
	instanceFlag string
	outputFlag  string
	quietFlag   bool
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "tkt",
	Short: "A fast CLI for project management",
	Long:  `tkt is a fast, agent-friendly CLI for project management.`,
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
	rootCmd.PersistentFlags().StringVarP(&instanceFlag, "instance", "i", "", "Provider instance name")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "Output format: table, json, wide, markdown")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Minimal output")
}

// getOutputMode returns the effective output mode.
func getOutputMode() render.OutputMode {
	if outputFlag != "" {
		return render.OutputMode(outputFlag)
	}
	return render.OutputTable
}

// buildService creates a service from resolved configuration.
func buildService() (*service.Service, *config.MergedConfig, error) {
	local, _, err := config.LoadLocal()
	if err != nil {
		return nil, nil, fmt.Errorf("load local config: %w", err)
	}
	private, _, err := config.LoadLocalPrivate()
	if err != nil {
		return nil, nil, fmt.Errorf("load private config: %w", err)
	}

	merged, err := config.Resolve(local, private)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve config: %w", err)
	}

	if instanceFlag != "" {
		merged.ProviderName = instanceFlag
	}

	if merged.ProviderURL == "" {
		msg := "no provider URL configured"
		if merged.ProviderName == "" {
			msg = "no provider configured. Run: tkt config set provider.url <url>"
		}
		return nil, nil, fmt.Errorf("%s", msg)
	}

	svc, err := service.NewService(merged)
	if err != nil {
		return nil, nil, err
	}
	return svc, merged, nil
}

// handleError prints an error in the appropriate format and exits with a meaningful code.
// It provides human-readable context and actionable hints based on the error type.
func handleError(err error) {
	render.Error(err, getOutputMode())
	code := 1
	var authErr *provider.AuthError
	var netErr *provider.NetworkError
	var notFoundErr *provider.NotFoundError
	var valErr *provider.ValidationError
	var conflictErr *provider.ConflictError

	switch {
	case errors.As(err, &authErr):
		code = 2
		fmt.Fprintln(os.Stderr, "\nHint: Your authentication token may be expired or invalid.")
		fmt.Fprintln(os.Stderr, "      Re-authenticate with: tkt remote auth")
	case errors.As(err, &netErr):
		code = 2
		fmt.Fprintln(os.Stderr, "\nHint: Could not reach the remote provider. Check your network connection")
		fmt.Fprintln(os.Stderr, "      and verify the provider URL in .tktrc.yml.")
	case errors.As(err, &notFoundErr):
		code = 3
		fmt.Fprintln(os.Stderr, "\nHint: The requested resource was not found.")
		fmt.Fprintln(os.Stderr, "      Check the ID and ensure it exists on the remote.")
	case errors.As(err, &valErr):
		code = 4
		fmt.Fprintln(os.Stderr, "\nHint: The remote provider rejected the operation.")
		fmt.Fprintln(os.Stderr, "      Verify the input values and try again.")
	case errors.As(err, &conflictErr):
		code = 1
		fmt.Fprintln(os.Stderr, "\nHint: A sync conflict was detected. Resolve it with:")
		fmt.Fprintln(os.Stderr, "      tkt remote status")
	}
	os.Exit(code)
}