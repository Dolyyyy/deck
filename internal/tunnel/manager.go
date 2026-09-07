package tunnel

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/Dolyyyy/deck/pkg/models"
)

// Manager manages the lifecycle of background SSH port forwarding tunnels.
type Manager struct {
	processes map[string]*exec.Cmd
	mu        sync.Mutex
}

// NewManager creates a new SSH tunnel manager.
func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*exec.Cmd),
	}
}

// IsPortAvailable checks whether a local TCP port is free.
func (m *Manager) IsPortAvailable(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// Start launches an SSH tunnel process in the background.
func (m *Manager) Start(t *models.Tunnel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t.LocalPort <= 0 || t.LocalPort > 65535 {
		return fmt.Errorf("invalid local port %d", t.LocalPort)
	}
	if t.RemotePort <= 0 || t.RemotePort > 65535 {
		return fmt.Errorf("invalid remote port %d", t.RemotePort)
	}
	if t.ServerName == "" {
		return fmt.Errorf("server name is required")
	}
	if t.RemoteHost == "" {
		t.RemoteHost = "127.0.0.1"
	}

	// Check if already running
	if cmd, ok := m.processes[t.Name]; ok && cmd != nil && cmd.Process != nil {
		if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
			return fmt.Errorf("tunnel %q is already running (PID %d)", t.Name, cmd.Process.Pid)
		}
	}

	// Check if port is available
	if !m.IsPortAvailable(t.LocalPort) {
		return fmt.Errorf("local port %d is already in use by another application", t.LocalPort)
	}

	// Build SSH command: ssh -N -L localPort:remoteHost:remotePort serverName
	var args []string
	switch t.Type {
	case "remote":
		args = []string{"-N", "-R", fmt.Sprintf("%d:%s:%d", t.LocalPort, t.RemoteHost, t.RemotePort), t.ServerName}
	case "dynamic":
		args = []string{"-N", "-D", fmt.Sprintf("%d", t.LocalPort), t.ServerName}
	default: // local
		args = []string{"-N", "-L", fmt.Sprintf("%d:%s:%d", t.LocalPort, t.RemoteHost, t.RemotePort), t.ServerName}
	}

	cmd := exec.Command("ssh", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		t.Active = false
		t.Error = err.Error()
		return fmt.Errorf("failed to start ssh tunnel: %w", err)
	}

	t.PID = cmd.Process.Pid
	t.Active = true
	t.Error = ""
	m.processes[t.Name] = cmd

	// Wait briefly to check if ssh exited immediately (e.g. invalid host or key)
	time.Sleep(150 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Active = false
		t.PID = 0
		delete(m.processes, t.Name)
		return fmt.Errorf("ssh tunnel process exited immediately; verify host %q", t.ServerName)
	}

	return nil
}

// Stop terminates an active SSH tunnel.
func (m *Manager) Stop(t *models.Tunnel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd, ok := m.processes[t.Name]
	if !ok || cmd == nil || cmd.Process == nil {
		// If PID is stored in model, try to kill by PID
		if t.PID > 0 {
			if proc, err := os.FindProcess(t.PID); err == nil {
				_ = proc.Signal(syscall.SIGTERM)
			}
		}
		t.Active = false
		t.PID = 0
		return nil
	}

	// Kill entire process group
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}

	time.Sleep(100 * time.Millisecond)
	_ = cmd.Process.Kill()

	delete(m.processes, t.Name)
	t.Active = false
	t.PID = 0
	t.Error = ""

	return nil
}

// IsRunning verifies if the tunnel process is alive.
func (m *Manager) IsRunning(t *models.Tunnel) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cmd, ok := m.processes[t.Name]; ok && cmd != nil && cmd.Process != nil {
		return cmd.Process.Signal(syscall.Signal(0)) == nil
	}

	if t.PID > 0 {
		proc, err := os.FindProcess(t.PID)
		if err != nil {
			return false
		}
		return proc.Signal(syscall.Signal(0)) == nil
	}

	return false
}

// StopAll stops all active tunnels managed by Deck.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, cmd := range m.processes {
		if cmd != nil && cmd.Process != nil {
			pgid, err := syscall.Getpgid(cmd.Process.Pid)
			if err == nil {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
			} else {
				_ = cmd.Process.Signal(syscall.SIGTERM)
			}
		}
		delete(m.processes, name)
	}
}
