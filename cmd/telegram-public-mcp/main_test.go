package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildHandlerRateLimitsMCPButNotHealth(t *testing.T) {
	t.Setenv("GLOBAL_RATE_LIMIT_PER_MINUTE", "1")
	t.Setenv("IP_RATE_LIMIT_PER_MINUTE", "1")

	handler := buildHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	if resp := serve(handler, http.MethodPost, "/mcp"); resp.Code != http.StatusOK {
		t.Fatalf("first mcp status = %d, want 200", resp.Code)
	}
	if resp := serve(handler, http.MethodPost, "/mcp"); resp.Code != http.StatusTooManyRequests {
		t.Fatalf("second mcp status = %d, want 429", resp.Code)
	}
	if resp := serve(handler, http.MethodGet, "/healthz"); resp.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want 200", resp.Code)
	}
}

func serve(handler http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = "203.0.113.1:12345"
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}
