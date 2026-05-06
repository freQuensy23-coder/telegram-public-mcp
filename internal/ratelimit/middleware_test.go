package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddlewareLimitsRequestsGlobally(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 5, 6, 18, 0, 0, 0, time.UTC))
	middleware := New(Config{
		GlobalLimit: 2,
		IPLimit:     100,
		Window:      time.Minute,
		Clock:       clock.Now,
	})
	handler := middleware.Wrap(okHandler())

	for i := 0; i < 2; i++ {
		resp := request(handler, "203.0.113.1")
		if resp.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i+1, resp.Code)
		}
	}

	resp := request(handler, "203.0.113.2")
	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", resp.Code)
	}
	if got := resp.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("Retry-After = %q, want 60", got)
	}
}

func TestMiddlewareLimitsRequestsPerIP(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 5, 6, 18, 0, 0, 0, time.UTC))
	middleware := New(Config{
		GlobalLimit: 100,
		IPLimit:     1,
		Window:      time.Minute,
		Clock:       clock.Now,
	})
	handler := middleware.Wrap(okHandler())

	if resp := request(handler, "203.0.113.1"); resp.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", resp.Code)
	}
	if resp := request(handler, "203.0.113.1"); resp.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", resp.Code)
	}
	if resp := request(handler, "203.0.113.2"); resp.Code != http.StatusOK {
		t.Fatalf("different IP status = %d, want 200", resp.Code)
	}
}

func TestMiddlewareResetsAfterWindow(t *testing.T) {
	clock := newFakeClock(time.Date(2026, 5, 6, 18, 0, 0, 0, time.UTC))
	middleware := New(Config{
		GlobalLimit: 1,
		IPLimit:     1,
		Window:      time.Minute,
		Clock:       clock.Now,
	})
	handler := middleware.Wrap(okHandler())

	if resp := request(handler, "203.0.113.1"); resp.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", resp.Code)
	}
	clock.Advance(time.Minute)
	if resp := request(handler, "203.0.113.1"); resp.Code != http.StatusOK {
		t.Fatalf("after reset status = %d, want 200", resp.Code)
	}
}

func TestMiddlewareUsesForwardedIP(t *testing.T) {
	middleware := New(Config{
		GlobalLimit: 100,
		IPLimit:     1,
		Window:      time.Minute,
		Clock:       time.Now,
	})
	handler := middleware.Wrap(okHandler())

	if resp := forwardedRequest(handler, "198.51.100.7, 10.0.0.1"); resp.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", resp.Code)
	}
	if resp := forwardedRequest(handler, "198.51.100.7, 10.0.0.2"); resp.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", resp.Code)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func request(handler http.Handler, remoteIP string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = remoteIP + ":12345"
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}

func forwardedRequest(handler http.Handler, xForwardedFor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", xForwardedFor)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}

type fakeClock struct {
	now time.Time
}

func newFakeClock(now time.Time) *fakeClock {
	return &fakeClock{now: now}
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}
