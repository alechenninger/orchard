package cli

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	vfprov "github.com/alechenninger/orchard/internal/provider/vz"
	runfs "github.com/alechenninger/orchard/internal/runstate/fs"
	"github.com/alechenninger/orchard/internal/shim/proc"
	fsstore "github.com/alechenninger/orchard/internal/vmstore/fs"
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
		cctx, cancel := context.WithCancel(ctx)
		defer cancel()
		store := fsstore.NewDefault()
		run := runfs.NewDefault()
		provider := vfprov.New()
		if err := proc.RunChild(cctx, store, run, provider, flagShimVM); err != nil {
			return err
		}
		// Should not reach here until signaled; just in case
		time.Sleep(10 * time.Millisecond)
		fmt.Println("shim exiting for", flagShimVM)
		return nil
	},
}
