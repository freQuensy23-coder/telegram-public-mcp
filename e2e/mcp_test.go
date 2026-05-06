package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/freQuensy23-coder/telegram-public-mcp/internal/mcp"
	"github.com/freQuensy23-coder/telegram-public-mcp/internal/telegram"
)

func TestMCPGetChannelInfoEndToEnd(t *testing.T) {
	html, err := os.ReadFile("../testdata/channel.html")
	if err != nil {
		t.Fatal(err)
	}
	telegramMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/s/example" {
			t.Fatalf("telegram path = %q, want /s/example", r.URL.Path)
		}
		_, _ = w.Write(html)
	}))
	defer telegramMock.Close()

	telegramClient, err := telegram.NewClient(telegramMock.URL, &http.Client{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	mcpServer := httptest.NewServer(mcp.NewServer(telegramClient))
	defer mcpServer.Close()

	resp := postRPC(t, mcpServer.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_channel_info",
			"arguments": map[string]any{
				"channel": "example",
			},
		},
	})

	result := resp["result"].(map[string]any)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if !bytes.Contains([]byte(text), []byte(`"title": "Example Channel"`)) {
		t.Fatalf("response content = %s", text)
	}
	if !bytes.Contains([]byte(text), []byte(`"subscribers": "12 345 subscribers"`)) {
		t.Fatalf("response content = %s", text)
	}
}

func postRPC(t *testing.T, endpoint string, body map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	httpResp, err := http.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer httpResp.Body.Close()
	var resp map[string]any
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["error"]; ok {
		t.Fatalf("unexpected MCP error: %#v", resp)
	}
	return resp
}
