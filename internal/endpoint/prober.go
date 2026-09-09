package endpoint

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	"github.com/Dolyyyy/deck/pkg/models"
)

// Prober checks HTTP/HTTPS endpoints for status code, latency, and SSL certificate expiry.
type Prober struct {
	client *http.Client
}

// NewProber creates a new endpoint prober with TLS inspection enabled.
func NewProber(timeout time.Duration) *Prober {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DisableKeepAlives: true,
	}

	return &Prober{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// Probe checks a single endpoint.
func (p *Prober) Probe(ctx context.Context, ep *models.Endpoint) *models.Endpoint {
	res := *ep
	res.LastChecked = time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.URL, nil)
	if err != nil {
		res.Status = models.EndpointStatusDown
		res.Error = err.Error()
		return &res
	}
	req.Header.Set("User-Agent", "deck-healthcheck/0.2.0")

	start := time.Now()
	resp, err := p.client.Do(req)
	res.Latency = time.Since(start)

	if err != nil {
		res.Status = models.EndpointStatusDown
		res.Error = err.Error()
		return &res
	}
	defer resp.Body.Close()

	res.StatusCode = resp.StatusCode
	res.Error = ""

	// Inspect TLS certificate if HTTPS
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		res.SSLExpiry = cert.NotAfter
		res.SSLDaysRemaining = int(time.Until(cert.NotAfter).Hours() / 24)
	}

	// Status resolution
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		if res.SSLDaysRemaining > 0 && res.SSLDaysRemaining <= 7 {
			res.Status = models.EndpointStatusDegraded
		} else {
			res.Status = models.EndpointStatusUp
		}
	} else {
		res.Status = models.EndpointStatusDegraded
	}

	return &res
}

// ProbeAll concurrently checks a slice of endpoints and streams results.
func (p *Prober) ProbeAll(ctx context.Context, endpoints []*models.Endpoint, concurrency int) <-chan *models.Endpoint {
	out := make(chan *models.Endpoint, len(endpoints))
	if len(endpoints) == 0 {
		close(out)
		return out
	}

	if concurrency <= 0 {
		concurrency = 4
	}

	go func() {
		defer close(out)
		sem := make(chan struct{}, concurrency)
		var wg sync.WaitGroup

		for _, ep := range endpoints {
			wg.Add(1)
			sem <- struct{}{}

			go func(e *models.Endpoint) {
				defer wg.Done()
				defer func() { <-sem }()

				select {
				case <-ctx.Done():
					return
				default:
					out <- p.Probe(ctx, e)
				}
			}(ep)
		}

		wg.Wait()
	}()

	return out
}
