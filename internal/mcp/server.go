package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/freQuensy23-coder/telegram-public-mcp/internal/telegram"
)

const protocolVersion = "2025-03-26"

type TelegramClient interface {
	ChannelInfo(ctx context.Context, channel string) (telegram.ChannelInfo, error)
	LatestPosts(ctx context.Context, channel string, opts telegram.LatestPostsOptions) ([]telegram.Post, error)
	SearchPosts(ctx context.Context, channel string, opts telegram.SearchOptions) ([]telegram.Post, error)
}

type Server struct {
	client TelegramClient
}

func NewServer(client TelegramClient) *Server {
	return &Server{client: client}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("telegram-public-mcp Streamable HTTP endpoint. Send JSON-RPC POST requests here.\n"))
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponse(w, response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}})
		return
	}
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}
	resp := s.handle(r.Context(), req)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeResponse(w, *resp)
}

func (s *Server) handle(ctx context.Context, req request) *response {
	if req.ID == nil && isNotification(req.Method) {
		return nil
	}
	resp := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    "telegram-public-mcp",
				"version": "0.1.0",
			},
		}
	case "tools/list":
		resp.Result = map[string]any{"tools": tools()}
	case "tools/call":
		result, err := s.callTool(ctx, req.Params)
		if err != nil {
			resp.Error = &rpcError{Code: -32602, Message: err.Error()}
		} else {
			resp.Result = result
		}
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found"}
	}
	return &resp
}

func (s *Server) callTool(ctx context.Context, raw json.RawMessage) (toolResult, error) {
	var params toolCallParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return toolResult{}, err
	}
	switch params.Name {
	case "get_channel_info":
		var args channelArgs
		if err := decodeArgs(params.Arguments, &args); err != nil {
			return toolResult{}, err
		}
		info, err := s.client.ChannelInfo(ctx, args.Channel)
		return jsonToolResult(info, err)
	case "get_latest_posts":
		var args latestPostsArgs
		if err := decodeArgs(params.Arguments, &args); err != nil {
			return toolResult{}, err
		}
		opts, err := args.options()
		if err != nil {
			return toolResult{}, err
		}
		posts, err := s.client.LatestPosts(ctx, args.Channel, opts)
		return jsonToolResult(posts, err)
	case "search_posts":
		var args searchArgs
		if err := decodeArgs(params.Arguments, &args); err != nil {
			return toolResult{}, err
		}
		posts, err := s.client.SearchPosts(ctx, args.Channel, telegram.SearchOptions{Query: args.Query, Limit: args.Limit})
		return jsonToolResult(posts, err)
	default:
		return toolResult{}, errors.New("unknown tool")
	}
}

func decodeArgs(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	return json.Unmarshal(raw, target)
}

func jsonToolResult(value any, err error) (toolResult, error) {
	if err != nil {
		return toolResult{}, err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return toolResult{}, err
	}
	return toolResult{
		Content: []content{{Type: "text", Text: string(data)}},
	}, nil
}

func writeResponse(w http.ResponseWriter, resp response) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func isNotification(method string) bool {
	return method == "notifications/initialized" || method == "notifications/cancelled"
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type channelArgs struct {
	Channel string `json:"channel"`
}

type latestPostsArgs struct {
	Channel      string `json:"channel"`
	Limit        int    `json:"limit,omitempty"`
	BeforePostID int    `json:"before_post_id,omitempty"`
	BeforeTime   string `json:"before_time,omitempty"`
}

func (a latestPostsArgs) options() (telegram.LatestPostsOptions, error) {
	opts := telegram.LatestPostsOptions{Limit: a.Limit, BeforePostID: a.BeforePostID}
	if a.BeforeTime == "" {
		return opts, nil
	}
	t, err := time.Parse(time.RFC3339, a.BeforeTime)
	if err != nil {
		return opts, errors.New("before_time must be RFC3339")
	}
	opts.BeforeTime = t
	return opts, nil
}

type searchArgs struct {
	Channel string `json:"channel"`
	Query   string `json:"query"`
	Limit   int    `json:"limit,omitempty"`
}

type toolResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func tools() []map[string]any {
	return []map[string]any{
		{
			"name":        "get_channel_info",
			"description": "Get public Telegram channel title, description, avatar URL, subscriber counter and canonical t.me/s URL.",
			"inputSchema": objectSchema(map[string]any{
				"channel": stringSchema("Channel username, @username, or t.me link."),
			}, []string{"channel"}),
		},
		{
			"name":        "get_latest_posts",
			"description": "Get latest public Telegram channel posts from t.me/s, including text, image URLs, views and timestamps.",
			"inputSchema": objectSchema(map[string]any{
				"channel":        stringSchema("Channel username, @username, or t.me link."),
				"limit":          numberSchema("Maximum posts to return, default 10, max 100."),
				"before_post_id": numberSchema("Return posts with a Telegram post id lower than this value."),
				"before_time":    stringSchema("Return posts before this RFC3339 timestamp."),
			}, []string{"channel"}),
		},
		{
			"name":        "search_posts",
			"description": "Search public Telegram channel posts using Telegram's t.me/s/{channel}?q= query endpoint.",
			"inputSchema": objectSchema(map[string]any{
				"channel": stringSchema("Channel username, @username, or t.me link."),
				"query":   stringSchema("Search query."),
				"limit":   numberSchema("Maximum posts to return, default 10, max 100."),
			}, []string{"channel", "query"}),
		},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func stringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func numberSchema(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}
