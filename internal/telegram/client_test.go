package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestChannelInfo(t *testing.T) {
	client := testClient(t, "channel.html")

	info, err := client.ChannelInfo(context.Background(), "@example")
	if err != nil {
		t.Fatalf("ChannelInfo returned error: %v", err)
	}
	if info.Username != "example" {
		t.Fatalf("Username = %q, want example", info.Username)
	}
	if info.Title != "Example Channel" {
		t.Fatalf("Title = %q", info.Title)
	}
	if info.AvatarURL != "https://cdn.example/avatar.jpg" {
		t.Fatalf("AvatarURL = %q", info.AvatarURL)
	}
	if info.Subscribers != "12 345 subscribers" {
		t.Fatalf("Subscribers = %q", info.Subscribers)
	}
}

func TestLatestPostsWithOffsets(t *testing.T) {
	client := testClient(t, "channel.html")
	before := time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)

	posts, err := client.LatestPosts(context.Background(), "example", LatestPostsOptions{
		Limit:        10,
		BeforePostID: 100,
		BeforeTime:   before,
	})
	if err != nil {
		t.Fatalf("LatestPosts returned error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want 1", len(posts))
	}
	if posts[0].ID != 99 {
		t.Fatalf("post id = %d, want 99", posts[0].ID)
	}
	if posts[0].Text != "Second post text" {
		t.Fatalf("post text = %q", posts[0].Text)
	}
}

func TestSearchPostsUsesTelegramQuery(t *testing.T) {
	html, err := os.ReadFile("../../testdata/channel.html")
	if err != nil {
		t.Fatal(err)
	}
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		_, _ = w.Write(html)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}

	posts, err := client.SearchPosts(context.Background(), "example", SearchOptions{Query: "hello", Limit: 1})
	if err != nil {
		t.Fatalf("SearchPosts returned error: %v", err)
	}
	if gotQuery != "hello" {
		t.Fatalf("telegram query = %q, want hello", gotQuery)
	}
	if len(posts) != 1 || posts[0].ID != 100 {
		t.Fatalf("posts = %#v, want first post only", posts)
	}
}

func testClient(t *testing.T, fixture string) *Client {
	t.Helper()
	html, err := os.ReadFile("../../testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(html)
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return client
}
