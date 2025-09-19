package cli

import (
	"fmt"

	"github.com/alechenninger/orchard/internal/application"
	"github.com/spf13/cobra"
)

var (
	flagImagePath   string
	flagCPUs        int
	flagMemoryMiB   int
	flagDiskSizeGiB int
	flagSSHKeyPath  string
)

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVar(&flagImagePath, "image", "", "path to base Fedora image (required)")
	upCmd.Flags().IntVar(&flagCPUs, "cpus", 2, "number of vCPUs")
	upCmd.Flags().IntVar(&flagMemoryMiB, "memory", 2048, "memory in MiB")
	upCmd.Flags().IntVar(&flagDiskSizeGiB, "disk-size", 20, "disk size in GiB")
	upCmd.Flags().StringVar(&flagSSHKeyPath, "ssh-key", "", "path to SSH public key (optional)")
	_ = upCmd.MarkFlagRequired("image")
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create a VM record and resources (no start)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		app := application.NewDefault()
		vm, err := app.Up(ctx, application.UpParams{
			ImagePath:   flagImagePath,
			CPUs:        flagCPUs,
			MemoryMiB:   flagMemoryMiB,
			DiskSizeGiB: flagDiskSizeGiB,
			SSHKeyPath:  flagSSHKeyPath,
		})
		if err != nil {
			return err
		}
		if flagJSON {
			fmt.Printf("{\"name\":\"%s\",\"image\":\"%s\"}\n", vm.Name, vm.BaseImageRef)
			return nil
		}
		fmt.Printf("Created VM %s\n", vm.Name)
		return nil
	},
}
