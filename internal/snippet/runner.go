package snippet

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/atotto/clipboard"
	"github.com/Dolyyyy/deck/pkg/models"
)

// Runner manages the execution and clipboard copying of command snippets.
type Runner struct{}

// NewRunner creates a new command snippet runner.
func NewRunner() *Runner {
	return &Runner{}
}

// CopyToClipboard copies the snippet command to the OS clipboard.
func (r *Runner) CopyToClipboard(s *models.Snippet) error {
	if s.Command == "" {
		return fmt.Errorf("snippet command is empty")
	}
	return clipboard.WriteAll(s.Command)
}

// BuildCommand constructs the exec.Cmd for either local or remote SSH execution.
func (r *Runner) BuildCommand(s *models.Snippet) (*exec.Cmd, error) {
	if s.Command == "" {
		return nil, fmt.Errorf("snippet command is empty")
	}

	if s.TargetHost != "" {
		// Remote SSH execution
		cmd := exec.Command("ssh", "-t", s.TargetHost, s.Command)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd, nil
	}

	// Local execution via user's shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-c", s.Command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}
