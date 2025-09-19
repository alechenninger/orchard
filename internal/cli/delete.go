package cli

import (
	"fmt"

	"github.com/alechenninger/orchard/internal/application"
	"github.com/spf13/cobra"
)

var flagDeleteForce bool

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVarP(&flagDeleteForce, "force", "f", false, "force stop if running before delete")
}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: "Delete a VM and its resources",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		app := application.NewDefault()
		name := args[0]
		if err := app.Delete(ctx, name, flagDeleteForce); err != nil {
			return err
		}
		if flagJSON {
			fmt.Printf("{\"name\":\"%s\",\"deleted\":true}\n", name)
			return nil
		}
		fmt.Printf("Deleted %s\n", name)
		return nil
	},
}
