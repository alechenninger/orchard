package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/alechenninger/orchard/internal/domain"
	fsstore "github.com/alechenninger/orchard/internal/vmstore/fs"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List VMs",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		store := fsstore.New(baseDir())
		vms, err := store.List(ctx)
		if err != nil {
			return err
		}
		if flagJSON {
			for _, vm := range vms {
				slog.Info("vm", "name", vm.Name, "status", vm.Status, "cpus", vm.CPUs, "memoryMiB", vm.MemoryMiB)
			}
			return nil
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "NAME\tSTATUS\tCPUS\tMEM(MiB)")
		for _, vm := range vms {
			fmt.Fprintf(tw, "%s\t%s\t%d\t%d\n", vm.Name, ifEmpty(vm.Status, "stopped"), vm.CPUs, vm.MemoryMiB)
		}
		return tw.Flush()
	},
}

func baseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "orchard")
	}
	return filepath.Join(home, ".orchard")
}

func ifEmpty(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

var _ domain.VMStore = (*fsstore.Store)(nil)
