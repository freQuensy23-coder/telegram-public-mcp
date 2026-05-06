package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/freQuensy23-coder/telegram-public-mcp/internal/mcp"
	"github.com/freQuensy23-coder/telegram-public-mcp/internal/telegram"
)

func main() {
	addr := env("ADDR", ":8080")
	baseURL := env("TELEGRAM_BASE_URL", "https://t.me")

	client, err := telegram.NewClient(baseURL, &http.Client{Timeout: 20 * time.Second})
	if err != nil {
		log.Fatalf("telegram client: %v", err)
	}
	server := mcp.NewServer(client)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("telegram-public-mcp\n"))
	})
	mux.Handle("/mcp", server)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	log.Printf("telegram-public-mcp listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
