package aliasdiscovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dolyyyy/deck/pkg/models"
)

func TestScanner_DiscoverWithProdAliases(t *testing.T) {
	tempDir := t.TempDir()

	// Simulate a parent ~/.bashrc that sources a separate alias file
	bashrcPath := filepath.Join(tempDir, ".bashrc")
	aliasFilePath := filepath.Join(tempDir, ".alias")

	bashrcContent := `
# ~/.bashrc
[ -f ~/.alias ] && . ~/.alias
export PATH=$PATH:/usr/local/bin
alias web="cd /home/admin/web"
alias doly="cd /home/ubuntu/doly"
`
	aliasFileContent := `
# GENERIC
alias c="clear"
alias 1="cd /home/ubuntu/doly"
alias 2="cd /home/admin/web"
alias 4="cd /var/log/"
alias p="chmod -R 777 *"
alias b="cd .."
alias lt='exa --long --header --all -all --classify --group --sort newest'
alias l='exa --long --header --all -all --classify --group'
alias pms='pm2 status'
alias cdn="cd /home/admin/web/cdn.codoly.fr/public_html/p"
alias t="tmux attach-session -t 0"
alias pull="echo 'Fetching & pulling...' && git fetch && git pull && git status"
alias bastion="cat /root/.alias"

# HOSTS
alias lc="ssh root@31.58.68.49"
alias orion="ssh root@31.58.68.13"
alias fanta="ssh root@game.fantasticrp.fr"
alias alexgr="ssh ubuntu@alexgr.codoly.fr"
alias ts="ssh debian@ts.lastcountryrp.fr"
alias next="ssh nextproject@next.doly.ovh"
alias vita="ssh root@vita.codoly.fr"
alias diamond="ssh cfx@137.74.33.137"
alias kylian="ssh ubuntu@hestia.kybr.fr"
alias qlf="ssh cfx@217.182.227.15"
alias inelya="ssh cfx@164.132.21.27"
alias bkps="ssh u481828@u481828.your-storagebox.de -p 23"
alias jump="ssh -J proxyuser@jumphost:22 targetuser@targethost"
alias srvcustom="echo 'Connecting...' && ssh -p 2200 -i ~/.ssh/id_rsa root@custom.host"

# Functions
myfunc() { ssh root@funcremote.host; }
`

	if err := os.WriteFile(bashrcPath, []byte(bashrcContent), 0600); err != nil {
		t.Fatalf("failed to write fake bashrc: %v", err)
	}
	if err := os.WriteFile(aliasFilePath, []byte(aliasFileContent), 0600); err != nil {
		t.Fatalf("failed to write fake alias file: %v", err)
	}

	// We pass only bashrcPath to Scanner, and it should RECURSIVELY discover aliasFilePath!
	scanner := NewScanner(bashrcPath)

	items, err := scanner.Discover(nil, nil, nil)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	servers := make(map[string]*models.Server)
	aliases := make(map[string]*models.DirectoryAlias)
	snippets := make(map[string]*models.Snippet)

	for _, it := range items {
		switch it.Type {
		case TypeServer:
			servers[it.Name] = it.Server
		case TypeAlias:
			aliases[it.Name] = it.Alias
		case TypeSnippet:
			snippets[it.Name] = it.Snippet
		}
	}

	// 1. Verify recursive discovery succeeded
	if len(servers) == 0 {
		t.Fatalf("expected servers to be discovered via recursive sourcing from .alias, but got 0")
	}

	// 2. Verify servers
	expectedServers := []struct {
		name     string
		user     string
		host     string
		port     int
		identity string
	}{
		{"lc", "root", "31.58.68.49", 22, ""},
		{"fanta", "root", "game.fantasticrp.fr", 22, ""},
		{"ts", "debian", "ts.lastcountryrp.fr", 22, ""},
		{"alexgr", "ubuntu", "alexgr.codoly.fr", 22, ""},
		{"bkps", "u481828", "u481828.your-storagebox.de", 23, ""},
		{"srvcustom", "root", "custom.host", 2200, "~/.ssh/id_rsa"},
		{"myfunc", "root", "funcremote.host", 22, ""},
	}

	for _, exp := range expectedServers {
		srv, exists := servers[exp.name]
		if !exists {
			t.Errorf("expected server %q to be discovered", exp.name)
			continue
		}
		if srv.User != exp.user {
			t.Errorf("server %q user mismatch: got %q, want %q", exp.name, srv.User, exp.user)
		}
		if srv.Hostname != exp.host {
			t.Errorf("server %q host mismatch: got %q, want %q", exp.name, srv.Hostname, exp.host)
		}
		if srv.Port != exp.port {
			t.Errorf("server %q port mismatch: got %d, want %d", exp.name, srv.Port, exp.port)
		}
		if exp.identity != "" && srv.IdentityFile != exp.identity {
			t.Errorf("server %q identity mismatch: got %q, want %q", exp.name, srv.IdentityFile, exp.identity)
		}
	}

	// 3. Verify directory aliases
	if _, ok := aliases["web"]; !ok {
		t.Errorf("expected directory alias 'web' to be discovered")
	}
	if _, ok := aliases["cdn"]; !ok {
		t.Errorf("expected directory alias 'cdn' to be discovered")
	}
	if _, ok := aliases["b"]; ok {
		t.Errorf("expected 'cd ..' not to be treated as a valid directory alias")
	}

	// 4. Verify snippets
	if _, ok := snippets["pms"]; !ok {
		t.Errorf("expected snippet 'pms' to be discovered")
	}
	if _, ok := snippets["t"]; !ok {
		t.Errorf("expected snippet 't' to be discovered")
	}
	if _, ok := snippets["c"]; !ok {
		t.Errorf("expected snippet 'c' to be discovered")
	}
	if _, ok := snippets["l"]; ok {
		t.Errorf("expected standard 'l' alias to be ignored")
	}
}

func TestScanner_FishAndGlobs(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Fish config
	fishDir := filepath.Join(tempDir, ".config", "fish")
	if err := os.MkdirAll(fishDir, 0755); err != nil {
		t.Fatalf("failed to create fish config dir: %v", err)
	}
	fishConfig := filepath.Join(fishDir, "config.fish")
	fishContent := `
alias vps "ssh root@fishvps.example.com"
abbr -a devproj "cd /home/user/fishdev"
abbr --add kctl "kubectl get pods"
`
	if err := os.WriteFile(fishConfig, []byte(fishContent), 0600); err != nil {
		t.Fatalf("failed to write fish config: %v", err)
	}

	// 2. Profile.d glob
	profileD := filepath.Join(tempDir, ".profile.d")
	if err := os.MkdirAll(profileD, 0755); err != nil {
		t.Fatalf("failed to create profile.d dir: %v", err)
	}
	profileScript := filepath.Join(profileD, "aliases.sh")
	profileContent := `
alias globserver='ssh -p 2222 admin@globsrv.net'
`
	if err := os.WriteFile(profileScript, []byte(profileContent), 0600); err != nil {
		t.Fatalf("failed to write profile.d script: %v", err)
	}

	mainProfile := filepath.Join(tempDir, ".profile")
	mainContent := `
for f in ` + profileD + `/*.sh; do
    source "$f"
done
`
	if err := os.WriteFile(mainProfile, []byte(mainContent), 0600); err != nil {
		t.Fatalf("failed to write main profile: %v", err)
	}

	scanner := NewScanner(fishConfig, mainProfile)
	items, err := scanner.Discover(nil, nil, nil)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	names := make(map[string]ItemType)
	for _, it := range items {
		names[it.Name] = it.Type
	}

	if names["vps"] != TypeServer {
		t.Errorf("expected fish alias 'vps' to be discovered as SSH Server, got %v", names["vps"])
	}
	if names["devproj"] != TypeAlias {
		t.Errorf("expected fish abbr 'devproj' to be discovered as Directory Alias, got %v", names["devproj"])
	}
	if names["kctl"] != TypeSnippet {
		t.Errorf("expected fish abbr 'kctl' to be discovered as Command Snippet, got %v", names["kctl"])
	}
	if names["globserver"] != TypeServer {
		t.Errorf("expected globbed 'globserver' to be discovered as SSH Server, got %v", names["globserver"])
	}
}

