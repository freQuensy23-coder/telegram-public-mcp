package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	errNotFound       = errors.New("telegram channel not found")
	postIDPattern     = regexp.MustCompile(`/([0-9]+)$`)
	backgroundPattern = regexp.MustCompile(`url\(['"]?([^'")]+)['"]?\)`)
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	userAgent  string
}

func NewClient(base string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(base) == "" {
		base = "https://t.me"
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    u,
		httpClient: httpClient,
		userAgent:  "telegram-public-mcp/0.1 (+https://github.com/freQuensy23-coder/telegram-public-mcp)",
	}, nil
}

func (c *Client) ChannelInfo(ctx context.Context, channel string) (ChannelInfo, error) {
	doc, canonical, err := c.fetch(ctx, channel, "")
	if err != nil {
		return ChannelInfo{}, err
	}
	header := doc.Find(".tgme_channel_info").First()
	info := ChannelInfo{
		Username: sanitizeChannel(channel),
		Title:    cleanText(header.Find(".tgme_channel_info_header_title").First().Text()),
		URL:      canonical,
	}
	if info.Title == "" {
		info.Title = cleanText(doc.Find("meta[property='og:title']").AttrOr("content", ""))
	}
	info.Description = cleanText(header.Find(".tgme_channel_info_description").First().Text())
	info.Subscribers = cleanText(header.Find(".tgme_channel_info_counters .counter_type").FilterFunction(func(_ int, s *goquery.Selection) bool {
		return strings.Contains(strings.ToLower(s.Text()), "subscribers")
	}).First().Parent().Text())
	if info.Subscribers == "" {
		header.Find(".tgme_channel_info_counter").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			txt := cleanText(s.Text())
			if strings.Contains(strings.ToLower(txt), "subscribers") {
				info.Subscribers = txt
				return false
			}
			return true
		})
	}
	if src, ok := header.Find(".tgme_page_photo_image img").First().Attr("src"); ok {
		info.AvatarURL = absolutize(c.baseURL, src)
	} else if style, ok := header.Find(".tgme_page_photo_image").First().Attr("style"); ok {
		info.AvatarURL = absolutize(c.baseURL, backgroundURL(style))
	}
	if info.Title == "" {
		return ChannelInfo{}, errNotFound
	}
	return info, nil
}

func (c *Client) LatestPosts(ctx context.Context, channel string, opts LatestPostsOptions) ([]Post, error) {
	doc, _, err := c.fetch(ctx, channel, "")
	if err != nil {
		return nil, err
	}
	posts := parsePosts(doc, c.baseURL)
	posts = filterPosts(posts, opts)
	return limitPosts(posts, normalizeLimit(opts.Limit)), nil
}

func (c *Client) SearchPosts(ctx context.Context, channel string, opts SearchOptions) ([]Post, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return nil, errors.New("query is required")
	}
	doc, _, err := c.fetch(ctx, channel, opts.Query)
	if err != nil {
		return nil, err
	}
	return limitPosts(parsePosts(doc, c.baseURL), normalizeLimit(opts.Limit)), nil
}

func (c *Client) fetch(ctx context.Context, channel, query string) (*goquery.Document, string, error) {
	username := sanitizeChannel(channel)
	if username == "" {
		return nil, "", errors.New("channel is required")
	}
	u := *c.baseURL
	u.Path = path.Join(c.baseURL.Path, "s", username)
	if query != "" {
		q := u.Query()
		q.Set("q", query)
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		io.Copy(io.Discard, resp.Body)
		return nil, "", errNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, resp.Body)
		return nil, "", fmt.Errorf("telegram returned HTTP %d", resp.StatusCode)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, "", err
	}
	canonical := c.baseURL.ResolveReference(&url.URL{Path: path.Join("s", username)}).String()
	return doc, canonical, nil
}

func parsePosts(doc *goquery.Document, base *url.URL) []Post {
	posts := make([]Post, 0)
	doc.Find(".tgme_widget_message").Each(func(_ int, s *goquery.Selection) {
		dataPost := s.AttrOr("data-post", "")
		id := parsePostID(dataPost)
		if id == 0 {
			return
		}
		post := Post{
			ID:     id,
			URL:    base.ResolveReference(&url.URL{Path: dataPost}).String(),
			Text:   cleanText(s.Find(".tgme_widget_message_text").First().Text()),
			Views:  cleanText(s.Find(".tgme_widget_message_views").First().Text()),
			Images: postImages(s, base),
		}
		if dt := s.Find("time").First().AttrOr("datetime", ""); dt != "" {
			post.Timestamp = dt
			if parsed, err := time.Parse(time.RFC3339, dt); err == nil {
				post.Datetime = parsed
			}
		}
		posts = append(posts, post)
	})
	return posts
}

func postImages(s *goquery.Selection, base *url.URL) []string {
	seen := map[string]bool{}
	var images []string
	s.Find(".tgme_widget_message_photo_wrap, .tgme_widget_message_video_thumb, img").Each(func(_ int, img *goquery.Selection) {
		candidates := []string{img.AttrOr("src", ""), backgroundURL(img.AttrOr("style", ""))}
		for _, candidate := range candidates {
			if candidate == "" {
				continue
			}
			absolute := absolutize(base, candidate)
			if !seen[absolute] {
				seen[absolute] = true
				images = append(images, absolute)
			}
		}
	})
	return images
}

func filterPosts(posts []Post, opts LatestPostsOptions) []Post {
	filtered := make([]Post, 0, len(posts))
	for _, post := range posts {
		if opts.BeforePostID > 0 && post.ID >= opts.BeforePostID {
			continue
		}
		if !opts.BeforeTime.IsZero() && !post.Datetime.IsZero() && !post.Datetime.Before(opts.BeforeTime) {
			continue
		}
		filtered = append(filtered, post)
	}
	return filtered
}

func limitPosts(posts []Post, limit int) []Post {
	if len(posts) <= limit {
		return posts
	}
	return posts[:limit]
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func parsePostID(dataPost string) int {
	match := postIDPattern.FindStringSubmatch(dataPost)
	if len(match) != 2 {
		return 0
	}
	id, _ := strconv.Atoi(match[1])
	return id
}

func sanitizeChannel(channel string) string {
	channel = strings.TrimSpace(channel)
	channel = strings.TrimPrefix(channel, "@")
	if u, err := url.Parse(channel); err == nil && u.Host != "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		for i, part := range parts {
			if part == "s" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return strings.Trim(channel, "/")
}

func cleanText(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func backgroundURL(style string) string {
	match := backgroundPattern.FindStringSubmatch(style)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

func absolutize(base *url.URL, raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return base.ResolveReference(u).String()
}
