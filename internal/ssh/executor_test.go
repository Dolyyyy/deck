package ssh

import (
	"testing"

	"github.com/Dolyyyy/deck/pkg/models"
)

func TestBuildSSHArgs(t *testing.T) {
	srv := &models.Server{
		Name:         "prod-app",
		Hostname:     "app.prod.internal",
		User:         "deploy",
		Port:         2222,
		IdentityFile: "/home/user/.ssh/id_ed25519",
		ProxyJump:    "bastion.prod.internal",
		ExtraOptions: map[string]string{
			"ServerAliveInterval": "30",
		},
	}

	args := BuildSSHArgs(srv)

	// Check essential arguments
	expectedContains := []string{
		"-p", "2222",
		"-i", "/home/user/.ssh/id_ed25519",
		"-J", "bastion.prod.internal",
		"-o", "ServerAliveInterval=30",
		"deploy@app.prod.internal",
	}

	for i := 0; i < len(expectedContains); i += 2 {
		flag := expectedContains[i]
		if i+1 < len(expectedContains) {
			val := expectedContains[i+1]
			found := false
			for j, a := range args {
				if a == flag && j+1 < len(args) && args[j+1] == val {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected arg %s %s in %v", flag, val, args)
			}
		}
	}

	// Verify target host at the end
	if args[len(args)-1] != "deploy@app.prod.internal" {
		t.Errorf("expected last arg to be deploy@app.prod.internal, got %s", args[len(args)-1])
	}
}

func TestParseProbeOutput(t *testing.T) {
	rawOutput := `Linux 6.8.0-45-generic x86_64
 23:14:02 up 12 days,  3 users,  load average: 0.12, 0.25, 0.18
Mem:          16000        8000        4000
/dev/sda1        100G   40G   60G  40% /
`
	stats := parseProbeOutput(rawOutput)

	if stats.OSInfo != "Linux 6.8.0-45-generic x86_64" {
		t.Errorf("got OSInfo %q, want %q", stats.OSInfo, "Linux 6.8.0-45-generic x86_64")
	}
	if stats.Uptime != "12 days" {
		t.Errorf("got Uptime %q, want %q", stats.Uptime, "12 days")
	}
	if stats.LoadAvg != "0.12, 0.25, 0.18" {
		t.Errorf("got LoadAvg %q, want %q", stats.LoadAvg, "0.12, 0.25, 0.18")
	}
	if stats.MemoryPct != 50 {
		t.Errorf("got MemoryPct %d, want 50", stats.MemoryPct)
	}
	if stats.DiskPct != 40 {
		t.Errorf("got DiskPct %d, want 40", stats.DiskPct)
	}
	if stats.DiskTotal != "100G" || stats.DiskUsed != "40G" {
		t.Errorf("got Disk %s / %s, want 40G / 100G", stats.DiskUsed, stats.DiskTotal)
	}
}
