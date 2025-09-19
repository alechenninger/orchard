package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "orchard",
		Short: "Opinionated VM orchestration on macOS",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return setupLogging(cmd.Context())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	flagJSON    bool
	flagVerbose bool
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "enable JSON log output")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "enable verbose (debug) logging")
}

func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setupLogging(ctx context.Context) error {
	var handler slog.Handler
	if flagJSON {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: chooseLevel(flagVerbose)})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: chooseLevel(flagVerbose)})
	}
	slog.SetDefault(slog.New(handler))
	slog.Debug("logging initialized")
	return nil
}

func chooseLevel(verbose bool) slog.Leveler {
	if verbose {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
