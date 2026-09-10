package aliasdiscovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dolyyyy/deck/pkg/models"
)

func TestScanner_Discover(t *testing.T) {
	tempDir := t.TempDir()
	bashrcPath := filepath.Join(tempDir, ".bashrc")

	content := `
# System aliases
alias ll='ls -alF'
alias grep='grep --color=auto'

# User custom aliases
alias hestia="ssh root@hestia.codoly.fr"
alias sandbox='ssh -p 2222 dev@192.168.1.100 -i ~/.ssh/id_rsa'
alias myproj="cd /home/user/projects/myproj"
alias d="pnpm dev"
`
	if err := os.WriteFile(bashrcPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create fake .bashrc: %v", err)
	}

	scanner := NewScanner(bashrcPath)

	// Existing server
	existingServers := []*models.Server{
		{Name: "hestia", Hostname: "hestia.codoly.fr", User: "root", Port: 22},
	}

	items, err := scanner.Discover(existingServers, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have filtered 'hestia' and 'll'/'grep', leaving 'sandbox', 'myproj', and 'd'
	foundSandbox := false
	foundMyproj := false
	foundD := false

	for _, it := range items {
		if it.Name == "hestia" {
			t.Errorf("expected hestia to be filtered out because it already exists")
		}
		if it.Name == "ll" || it.Name == "grep" {
			t.Errorf("expected common alias %s to be ignored", it.Name)
		}
		if it.Name == "sandbox" {
			foundSandbox = true
			if it.Type != TypeServer || it.Server.Port != 2222 || it.Server.User != "dev" {
				t.Errorf("sandbox server incorrectly parsed: %+v", it.Server)
			}
		}
		if it.Name == "myproj" {
			foundMyproj = true
			if it.Type != TypeAlias || it.Alias.Path != "/home/user/projects/myproj" {
				t.Errorf("myproj alias incorrectly parsed: %+v", it.Alias)
			}
		}
		if it.Name == "d" {
			foundD = true
			if it.Type != TypeSnippet || it.Snippet.Command != "pnpm dev" {
				t.Errorf("d snippet incorrectly parsed: %+v", it.Snippet)
			}
		}
	}

	if !foundSandbox {
		t.Errorf("expected to find sandbox server alias")
	}
	if !foundMyproj {
		t.Errorf("expected to find myproj directory alias")
	}
	if !foundD {
		t.Errorf("expected to find d command snippet")
	}
}
