package ping

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Dolyyyy/deck/pkg/models"
)

// DefaultTimeout for individual TCP ping probes.
const DefaultTimeout = 2500 * time.Millisecond

// TCPPinger implements models.Pinger using concurrent TCP SYN connections.
type TCPPinger struct {
	timeout time.Duration
}

// NewTCPPinger returns a new pinger with configured timeout.
func NewTCPPinger(timeout time.Duration) *TCPPinger {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &TCPPinger{timeout: timeout}
}

// Probe checks a single server's reachability and measures latency.
func (p *TCPPinger) Probe(ctx context.Context, server *models.Server) (*models.PingResult, error) {
	if server == nil {
		return nil, fmt.Errorf("nil server")
	}

	host := server.Hostname
	if host == "" {
		host = server.Name
	}
	target := net.JoinHostPort(host, fmt.Sprintf("%d", server.EffectivePort()))

	d := net.Dialer{Timeout: p.timeout}

	start := time.Now()
	conn, err := d.DialContext(ctx, "tcp", target)
	latency := time.Since(start)
	if latency <= 0 {
		latency = 1 * time.Microsecond
	}

	res := &models.PingResult{
		ServerName: server.Name,
		Latency:    latency,
		Timestamp:  time.Now(),
	}

	if err != nil {
		res.Status = models.StatusOffline
		res.Err = err
		return res, nil
	}

	_ = conn.Close()
	res.Status = models.StatusOnline
	return res, nil
}

// ProbeAll launches a concurrent worker pool to ping all servers asynchronously.
func (p *TCPPinger) ProbeAll(ctx context.Context, servers []*models.Server, concurrency int) <-chan *models.PingResult {
	if concurrency <= 0 {
		concurrency = 10
	}
	if concurrency > len(servers) && len(servers) > 0 {
		concurrency = len(servers)
	}

	results := make(chan *models.PingResult, len(servers))
	serverChan := make(chan *models.Server, len(servers))

	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range serverChan {
				select {
				case <-ctx.Done():
					return
				default:
					res, _ := p.Probe(ctx, s)
					results <- res
				}
			}
		}()
	}

	go func() {
		for _, s := range servers {
			serverChan <- s
		}
		close(serverChan)
		wg.Wait()
		close(results)
	}()

	return results
}
