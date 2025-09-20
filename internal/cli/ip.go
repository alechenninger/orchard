package cli

import (
	"fmt"

	"github.com/alechenninger/orchard/internal/application"
	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(ipCmd) }

var ipCmd = &cobra.Command{
	Use:   "ip NAME",
	Short: "Show VM IP via mDNS (NAME.local)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		app := application.NewDefault()
		ip, err := app.IP(ctx, args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			fmt.Printf("{\"name\":\"%s\",\"ip\":\"%s\"}\n", args[0], ip)
			return nil
		}
		fmt.Println(ip)
		return nil
	},
}
