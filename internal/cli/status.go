package cli

import (
	"fmt"

	"github.com/alechenninger/orchard/internal/application"
	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(statusCmd) }

var statusCmd = &cobra.Command{
	Use:   "status NAME",
	Short: "Show VM status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		app := application.NewDefault()
		running, pid, err := app.Status(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			fmt.Printf("{\"name\":\"%s\",\"running\":%v,\"pid\":%d}\n", args[0], running, pid)
			return nil
		}
		if running {
			fmt.Printf("%s: running (pid %d)\n", args[0], pid)
		} else {
			fmt.Printf("%s: stopped\n", args[0])
		}
		return nil
	},
}
