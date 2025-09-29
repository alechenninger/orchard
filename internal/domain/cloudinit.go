package domain

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alechenninger/orchard/internal/cloudinit/hdiutil"
	"github.com/spf13/afero"
)

// CIDATABuilder turns a directory containing cloud-init NoCloud files into an ISO image at dstPath.
type CIDATABuilder interface {
	Build(ctx context.Context, fs afero.Fs, srcDir string, dstPath string) error
}

// CloudInit generates a NoCloud seed ISO for a VM using an injected builder.
type CloudInit struct {
	fs      afero.Fs
	builder CIDATABuilder
}

func NewCloudInit() *CloudInit { return &CloudInit{fs: afero.NewOsFs(), builder: hdiutil.Builder{}} }

func NewCloudInitWithFSAndBuilder(fs afero.Fs, builder CIDATABuilder) *CloudInit {
	return &CloudInit{fs: fs, builder: builder}
}

// Generate creates a NoCloud seed ISO at dstPath with the provided ssh key and vm hostname.
// It requires macOS hdiutil to be available.
func (c *CloudInit) Generate(ctx context.Context, vm VM, sshAuthorizedKey string, dstPath string) error {
	af := &afero.Afero{Fs: c.fs}
	workDir, err := af.TempDir("", "orchard-seed-")
	if err != nil {
		return err
	}
	defer af.RemoveAll(workDir)

	userData := buildUserData(vm.Hostname, sshAuthorizedKey)
	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", vm.Name, vm.Hostname)

	if err := af.WriteFile(filepath.Join(workDir, "user-data"), []byte(userData), 0o644); err != nil {
		return err
	}
	if err := af.WriteFile(filepath.Join(workDir, "meta-data"), []byte(metaData), 0o644); err != nil {
		return err
	}

	if err := af.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	// Build CIDATA ISO from workDir into dstPath
	if err := c.builder.Build(ctx, c.fs, workDir, dstPath); err != nil {
		return fmt.Errorf("hdiutil makehybrid failed: %w", err)
	}
	return nil
}

func buildUserData(hostname, sshKey string) string {
	b := &strings.Builder{}
	b.WriteString("#cloud-config\n")
	b.WriteString("preserve_hostname: false\n")
	b.WriteString(fmt.Sprintf("hostname: %s\n", hostname))
	b.WriteString("ssh_pwauth: false\n")
	b.WriteString("users:\n")
	b.WriteString("  - name: fedora\n")
	b.WriteString("    sudo: ALL=(ALL) NOPASSWD:ALL\n")
	b.WriteString("    groups: wheel\n")
	b.WriteString("    shell: /bin/bash\n")
	b.WriteString("    ssh_authorized_keys:\n")
	b.WriteString("      - ")
	b.WriteString(strings.TrimSpace(sshKey))
	b.WriteString("\n")
	b.WriteString("package_update: true\n")
	b.WriteString("packages:\n")
	b.WriteString("  - avahi\n")
	b.WriteString("  - nss-mdns\n")
	b.WriteString("runcmd:\n")
	b.WriteString("  - [systemctl, enable, --now, avahi-daemon]\n")
	return b.String()
}
