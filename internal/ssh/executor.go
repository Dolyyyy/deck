package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Dolyyyy/deck/pkg/models"
)

// Executor implements models.SSHExecutor.
type Executor struct{}

// NewExecutor creates a new SSH executor.
func NewExecutor() *Executor {
	return &Executor{}
}

// BuildSSHArgs constructs the standard argument slice for ssh command execution.
func BuildSSHArgs(server *models.Server) []string {
	var args []string

	if server.Port > 0 && server.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", server.Port))
	}
	if server.IdentityFile != "" {
		args = append(args, "-i", server.IdentityFile)
	}
	if server.ProxyJump != "" {
		args = append(args, "-J", server.ProxyJump)
	}

	for k, v := range server.ExtraOptions {
		args = append(args, "-o", fmt.Sprintf("%s=%s", k, v))
	}

	target := server.Hostname
	if target == "" {
		target = server.Name
	}
	if server.User != "" {
		target = fmt.Sprintf("%s@%s", server.User, target)
	}

	args = append(args, target)
	return args
}

// Connect hands over the current terminal to native OpenSSH client.
func (e *Executor) Connect(server *models.Server) error {
	sshBinary, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh binary not found in PATH: %w", err)
	}

	args := append([]string{"ssh"}, BuildSSHArgs(server)...)

	// Under Unix, syscall.Exec directly replaces process, preserving TTY & signal handling
	err = syscall.Exec(sshBinary, args, os.Environ())
	if err != nil {
		// Fallback for Windows or environments where syscall.Exec fails
		cmd := exec.Command(sshBinary, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return nil
}

// FetchStats connects non-interactively via SSH to fetch lightweight system metrics.
func (e *Executor) FetchStats(ctx context.Context, server *models.Server) (*models.ServerStats, error) {
	sshBinary, err := exec.LookPath("ssh")
	if err != nil {
		return nil, fmt.Errorf("ssh binary not found in PATH: %w", err)
	}

	// Command probes uname, uptime, free memory and root disk usage
	probeCmd := `uname -srm; uptime; free -m 2>/dev/null || vm_stat; df -h / 2>/dev/null`

	baseArgs := BuildSSHArgs(server)
	// Add batch options to prevent hanging on password prompts
	args := append([]string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=3",
		"-o", "StrictHostKeyChecking=accept-new",
	}, baseArgs...)
	args = append(args, probeCmd)

	cmd := exec.CommandContext(ctx, sshBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())
		friendlyErr := humanizeSSHError(errStr, server)
		return &models.ServerStats{
			FetchedAt: time.Now(),
			Error:     friendlyErr,
		}, fmt.Errorf("%s", friendlyErr)
	}

	return parseProbeOutput(stdout.String()), nil
}

func humanizeSSHError(rawErr string, s *models.Server) string {
	lower := strings.ToLower(rawErr)
	target := s.DisplayAddress()

	if strings.Contains(lower, "connection refused") {
		return fmt.Sprintf("Connection refused on %s (port %d is closed or SSH daemon is not running)", target, s.EffectivePort())
	}
	if strings.Contains(lower, "permission denied") {
		return fmt.Sprintf("Authentication failed on %s (SSH key not accepted or user incorrect)", target)
	}
	if strings.Contains(lower, "timed out") || strings.Contains(lower, "timeout") {
		return fmt.Sprintf("Connection timed out to %s (host unreachable or firewall blocking)", target)
	}
	if strings.Contains(lower, "could not resolve") || strings.Contains(lower, "name or service not known") {
		return fmt.Sprintf("Hostname could not be resolved: %s", s.Hostname)
	}
	if rawErr == "" {
		return fmt.Sprintf("SSH connection closed unexpectedly (exit code 255)")
	}
	return rawErr
}

func parseProbeOutput(raw string) *models.ServerStats {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	stats := &models.ServerStats{
		FetchedAt: time.Now(),
	}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if i == 0 {
			stats.OSInfo = line
			continue
		}

		// Uptime line: " 23:12:00 up 45 days,  2 users,  load average: 0.15, 0.22, 0.18"
		if strings.Contains(line, "load average") || strings.Contains(line, "load averages") {
			if parts := strings.Split(line, "load average:"); len(parts) == 2 {
				stats.LoadAvg = strings.TrimSpace(parts[1])
				// Extract uptime
				upPart := parts[0]
				if upIdx := strings.Index(upPart, "up "); upIdx != -1 {
					afterUp := upPart[upIdx+3:]
					if commaIdx := strings.Index(afterUp, ","); commaIdx != -1 {
						stats.Uptime = strings.TrimSpace(afterUp[:commaIdx])
					} else {
						stats.Uptime = strings.TrimSpace(afterUp)
					}
				}
			}
		}

		// Memory line: "Mem:          15926        6842        4210"
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				totalMB, _ := strconv.Atoi(fields[1])
				usedMB, _ := strconv.Atoi(fields[2])
				if totalMB > 0 {
					stats.MemoryTotal = fmt.Sprintf("%.1f GB", float64(totalMB)/1024.0)
					stats.MemoryUsed = fmt.Sprintf("%.1f GB", float64(usedMB)/1024.0)
					stats.MemoryPct = int((float64(usedMB) / float64(totalMB)) * 100)
				}
			}
		}

		// Disk line: "/dev/sda1        50G   22G   28G  45% /"
		if strings.HasSuffix(line, " /") || strings.HasSuffix(line, " /System/Volumes/Data") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				stats.DiskTotal = fields[1]
				stats.DiskUsed = fields[2]
				pctStr := strings.TrimSuffix(fields[4], "%")
				pct, _ := strconv.Atoi(pctStr)
				stats.DiskPct = pct
			}
		}
	}

	return stats
}
