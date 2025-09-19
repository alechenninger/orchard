package cli

import (
	"fmt"
	"log/slog"

	"github.com/alechenninger/orchard/internal/application"
	"github.com/spf13/cobra"
)

var (
	flagShimVM string
)

func init() {
	rootCmd.AddCommand(shimCmd)
	shimCmd.Hidden = true
	shimCmd.Flags().StringVar(&flagShimVM, "vm", "", "VM name to run")
	_ = shimCmd.MarkFlagRequired("vm")
}

var shimCmd = &cobra.Command{
	Use:   "_shim",
	Short: "internal: per-VM shim entrypoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		slog.Info("shim starting", "vm", flagShimVM)
		_ = application.NewDefault()
		// Placeholder: later call provider to start and block
		fmt.Println("shim would run VM", flagShimVM)
		_ = ctx
		return nil
	},
}
