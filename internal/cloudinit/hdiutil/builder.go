package hdiutil

import (
	"context"
	"os/exec"

	"github.com/spf13/afero"
)

// Builder builds a cloud-init CIDATA ISO using macOS hdiutil.
// It expects srcDir to contain NoCloud files (user-data, meta-data) and writes to dstPath.
type Builder struct{}

func (Builder) Build(ctx context.Context, _ afero.Fs, srcDir string, dstPath string) error {
	cmd := exec.CommandContext(ctx, "hdiutil", "makehybrid", "-iso", "-joliet", "-default-volume-name", "CIDATA", srcDir, "-o", dstPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
