package tunnel

import (
	"testing"

	"github.com/Dolyyyy/deck/pkg/models"
)

func TestPortAvailability(t *testing.T) {
	mgr := NewManager()

	// High random port should normally be free
	if !mgr.IsPortAvailable(49152) {
		t.Log("Port 49152 is occupied, skipping")
	}

	// Invalid ports
	tun := &models.Tunnel{
		Name:       "bad-tunnel",
		ServerName: "localhost",
		LocalPort:  -1,
		RemotePort: 80,
	}
	if err := mgr.Start(tun); err == nil {
		t.Error("Expected error for negative local port, got nil")
	}
}

func TestTunnelStopSafety(t *testing.T) {
	mgr := NewManager()
	tun := &models.Tunnel{
		Name:   "non-existent",
		Active: false,
		PID:    0,
	}

	// Stopping a non-running tunnel should be a no-op without panic
	if err := mgr.Stop(tun); err != nil {
		t.Errorf("Unexpected error stopping inactive tunnel: %v", err)
	}

	if mgr.IsRunning(tun) {
		t.Error("Expected IsRunning to return false")
	}
}
