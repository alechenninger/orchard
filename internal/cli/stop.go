package cli

import (
	"fmt"

	"github.com/alechenninger/orchard/internal/application"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop NAME",
	Short: "Stop a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		app := application.NewDefault()
		if err := app.Stop(ctx, args[0]); err != nil {
			return err
		}
		if flagJSON {
			fmt.Printf("{\"name\":\"%s\",\"stopped\":true}\n", args[0])
			return nil
		}
		fmt.Printf("Stopped %s\n", args[0])
		return nil
	},
}
