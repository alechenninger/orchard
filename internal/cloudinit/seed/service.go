package seed

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alechenninger/orchard/internal/domain"
)

type Service struct{}

func New() *Service { return &Service{} }

// Generate creates a NoCloud seed ISO at dstPath with the provided ssh key and vm hostname.
// It requires macOS hdiutil to be available.
func (s *Service) Generate(ctx context.Context, vm domain.VM, sshAuthorizedKey string, dstPath string) error {
	workDir, err := os.MkdirTemp("", "orchard-seed-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	userData := buildUserData(vm.Hostname, sshAuthorizedKey)
	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", vm.Name, vm.Hostname)

	if err := os.WriteFile(filepath.Join(workDir, "user-data"), []byte(userData), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workDir, "meta-data"), []byte(metaData), 0o644); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	// hdiutil makehybrid -iso -joliet -default-volume-name CIDATA <workDir> -o <dstPath>
	cmd := exec.CommandContext(ctx, "hdiutil", "makehybrid", "-iso", "-joliet", "-default-volume-name", "CIDATA", workDir, "-o", dstPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
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
	b.WriteString("  - default\n")
	if strings.TrimSpace(sshKey) != "" {
		b.WriteString("    ssh_authorized_keys:\n")
		b.WriteString("      - ")
		b.WriteString(strings.TrimSpace(sshKey))
		b.WriteString("\n")
	}
	b.WriteString("package_update: true\n")
	b.WriteString("packages:\n")
	b.WriteString("  - avahi\n")
	b.WriteString("  - nss-mdns\n")
	b.WriteString("runcmd:\n")
	b.WriteString("  - [systemctl, enable, --now, avahi-daemon]\n")
	return b.String()
}
