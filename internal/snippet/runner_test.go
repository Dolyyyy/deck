package snippet

import (
	"testing"

	"github.com/Dolyyyy/deck/pkg/models"
)

func TestBuildCommandLocal(t *testing.T) {
	r := NewRunner()
	snip := &models.Snippet{
		Name:    "echo-test",
		Command: "echo hello",
	}

	cmd, err := r.BuildCommand(snip)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}
	if len(cmd.Args) < 3 {
		t.Errorf("Expected at least 3 args for subshell command, got %v", cmd.Args)
	}
}

func TestBuildCommandRemote(t *testing.T) {
	r := NewRunner()
	snip := &models.Snippet{
		Name:       "remote-uptime",
		Command:    "uptime",
		TargetHost: "prod-web",
	}

	cmd, err := r.BuildCommand(snip)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}
	if cmd.Path != "ssh" && cmd.Args[0] != "ssh" {
		t.Errorf("Expected ssh command, got %s", cmd.Args[0])
	}
}

func TestEmptyCommandFails(t *testing.T) {
	r := NewRunner()
	snip := &models.Snippet{
		Name: "empty",
	}

	_, err := r.BuildCommand(snip)
	if err == nil {
		t.Error("Expected error for empty snippet command, got nil")
	}
}
