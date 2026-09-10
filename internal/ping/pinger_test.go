package ping

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Dolyyyy/deck/pkg/models"
)

func TestTCPPinger_ProbeOnline(t *testing.T) {
	// Start a local dummy TCP listener
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port

	pinger := NewTCPPinger(1 * time.Second)
	srv := &models.Server{
		Name:     "local-dummy",
		Hostname: "127.0.0.1",
		Port:     port,
	}

	ctx := context.Background()
	res, err := pinger.Probe(ctx, srv)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}

	if res.Status != models.StatusOnline {
		t.Fatalf("expected status online, got %s (err: %v)", res.Status, res.Err)
	}
	if res.Latency < 0 {
		t.Fatal("expected non-negative latency measurement")
	}
}

func TestTCPPinger_ProbeOffline(t *testing.T) {
	// Non-listening port
	pinger := NewTCPPinger(200 * time.Millisecond)
	srv := &models.Server{
		Name:     "offline-srv",
		Hostname: "127.0.0.1",
		Port:     65530, // Unlikely to be open
	}

	ctx := context.Background()
	res, err := pinger.Probe(ctx, srv)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}

	if res.Status != models.StatusOffline {
		t.Fatalf("expected status offline, got %s", res.Status)
	}
}

func TestTCPPinger_ProbeAll(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port

	servers := []*models.Server{
		{Name: "srv1", Hostname: "127.0.0.1", Port: port},
		{Name: "srv2", Hostname: "127.0.0.1", Port: 65530},
	}

	pinger := NewTCPPinger(300 * time.Millisecond)
	ctx := context.Background()

	resultsChan := pinger.ProbeAll(ctx, servers, 2)

	var received int
	for res := range resultsChan {
		received++
		if res.ServerName == "srv1" && res.Status != models.StatusOnline {
			t.Errorf("srv1 should be online, got %s", res.Status)
		}
	}

	if received != 2 {
		t.Fatalf("expected 2 results, got %d", received)
	}
}
