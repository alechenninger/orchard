package cli

import (
	"fmt"

	"github.com/alechenninger/orchard/internal/application"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start NAME",
	Short: "Start a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		app := application.NewDefault()
		vm, err := app.Start(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			fmt.Printf("{\"name\":\"%s\",\"pid\":%d}\n", vm.Name, vm.PID)
			return nil
		}
		fmt.Printf("Started %s (pid %d)\n", vm.Name, vm.PID)
		return nil
	},
}
