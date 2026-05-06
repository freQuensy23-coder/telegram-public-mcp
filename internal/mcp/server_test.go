package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/freQuensy23-coder/telegram-public-mcp/internal/telegram"
)

func TestToolsList(t *testing.T) {
	server := httptest.NewServer(NewServer(fakeTelegram{}))
	defer server.Close()

	resp := postRPC(t, server.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	})
	result := resp["result"].(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 3 {
		t.Fatalf("len(tools) = %d, want 3", len(tools))
	}
}

func TestToolsCallLatestPosts(t *testing.T) {
	server := httptest.NewServer(NewServer(fakeTelegram{}))
	defer server.Close()

	resp := postRPC(t, server.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      "call-1",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_latest_posts",
			"arguments": map[string]any{
				"channel": "example",
				"limit":   1,
			},
		},
	})
	if _, ok := resp["error"]; ok {
		t.Fatalf("unexpected error response: %#v", resp)
	}
	result := resp["result"].(map[string]any)
	content := result["content"].([]any)
	first := content[0].(map[string]any)
	if first["type"] != "text" {
		t.Fatalf("content type = %v", first["type"])
	}
	if !bytes.Contains([]byte(first["text"].(string)), []byte("hello")) {
		t.Fatalf("content text = %q", first["text"])
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
	return resp
}

type fakeTelegram struct{}

func (fakeTelegram) ChannelInfo(_ context.Context, _ string) (telegram.ChannelInfo, error) {
	panic("unused")
}

func (fakeTelegram) LatestPosts(_ context.Context, _ string, _ telegram.LatestPostsOptions) ([]telegram.Post, error) {
	return []telegram.Post{{ID: 1, Text: "hello"}}, nil
}

func (fakeTelegram) SearchPosts(_ context.Context, _ string, _ telegram.SearchOptions) ([]telegram.Post, error) {
	return nil, nil
}
