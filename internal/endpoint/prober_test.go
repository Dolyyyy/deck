package endpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dolyyyy/deck/pkg/models"
)

func TestEndpointProber(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	p := NewProber(2 * time.Second)
	ep := &models.Endpoint{
		Name: "test-server",
		URL:  ts.URL,
	}

	res := p.Probe(context.Background(), ep)
	if res.Status != models.EndpointStatusUp {
		t.Errorf("Expected status UP, got %s (error: %s)", res.Status, res.Error)
	}
	if res.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
	if res.Latency <= 0 {
		t.Error("Expected positive latency measurement")
	}
}

func TestEndpointProberDown(t *testing.T) {
	p := NewProber(1 * time.Second)
	ep := &models.Endpoint{
		Name: "invalid-host",
		URL:  "http://127.0.0.1:59999/non-existent",
	}

	res := p.Probe(context.Background(), ep)
	if res.Status != models.EndpointStatusDown {
		t.Errorf("Expected status DOWN, got %s", res.Status)
	}
	if res.Error == "" {
		t.Error("Expected error message to be populated")
	}
}
