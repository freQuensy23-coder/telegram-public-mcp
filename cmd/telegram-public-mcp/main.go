package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/freQuensy23-coder/telegram-public-mcp/internal/mcp"
	"github.com/freQuensy23-coder/telegram-public-mcp/internal/ratelimit"
	"github.com/freQuensy23-coder/telegram-public-mcp/internal/telegram"
)

func main() {
	addr := env("ADDR", ":8080")
	baseURL := env("TELEGRAM_BASE_URL", "https://t.me")

	client, err := telegram.NewClient(baseURL, &http.Client{Timeout: 20 * time.Second})
	if err != nil {
		log.Fatalf("telegram client: %v", err)
	}
	handler := buildHandler(mcp.NewServer(client))

	log.Printf("telegram-public-mcp listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func buildHandler(mcpHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("telegram-public-mcp\n"))
	})
	mux.Handle("/mcp", ratelimit.New(ratelimit.Config{
		GlobalLimit: envInt("GLOBAL_RATE_LIMIT_PER_MINUTE", 100),
		IPLimit:     envInt("IP_RATE_LIMIT_PER_MINUTE", 35),
		Window:      time.Minute,
	}).Wrap(mcpHandler))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	return mux
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
