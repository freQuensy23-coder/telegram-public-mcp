# telegram-public-mcp

HTTP MCP server for reading public Telegram channels through Telegram's public `t.me/s/{channel}` pages.

It does not use Telegram API credentials. It only performs ordinary HTTP GET requests to public pages, so it can read public channel metadata, recent posts, post images, and Telegram's built-in public search results.

## Tools

- `get_channel_info`: title, description, avatar URL, subscriber counter, canonical `t.me/s` URL.
- `get_latest_posts`: latest posts with text, image URLs, views, timestamps, and offsets by `before_post_id` or `before_time`.
- `search_posts`: public search via `https://t.me/s/{channel}?q={query}`.

## Run

```bash
go run ./cmd/telegram-public-mcp
```

The server listens on `:8080` by default.

Environment variables:

- `ADDR`: HTTP listen address, default `:8080`.
- `TELEGRAM_BASE_URL`: Telegram base URL, default `https://t.me`. Mostly useful for tests.
- `GLOBAL_RATE_LIMIT_PER_MINUTE`: global `/mcp` request limit across all clients, default `100`.
- `IP_RATE_LIMIT_PER_MINUTE`: `/mcp` request limit for one client IP, default `35`.

## MCP Endpoint

The MCP endpoint is:

```text
POST http://localhost:8080/mcp
```

Public Coolify deployment:

```text
https://api.fstr.cc/telegram-public-mcp/mcp
```

Codex setup:

```bash
codex mcp add telegram-public --url https://api.fstr.cc/telegram-public-mcp/mcp
```

It uses JSON-RPC over Streamable HTTP. The implementation supports:

- `initialize`
- `tools/list`
- `tools/call`
- `notifications/initialized`

## Examples

List tools:

```bash
curl -s http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | jq
```

Get channel info:

```bash
curl -s http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "get_channel_info",
      "arguments": { "channel": "telegram" }
    }
  }' | jq
```

Get latest posts:

```bash
curl -s http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "get_latest_posts",
      "arguments": {
        "channel": "telegram",
        "limit": 5,
        "before_time": "2026-05-01T00:00:00Z"
      }
    }
  }' | jq
```

Search posts:

```bash
curl -s http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
      "name": "search_posts",
      "arguments": {
        "channel": "telegram",
        "query": "privacy",
        "limit": 5
      }
    }
  }' | jq
```

## Docker

```bash
docker build -t telegram-public-mcp .
docker run --rm -p 8080:8080 telegram-public-mcp
```

## Tests

```bash
go test ./...
```

The tests use local HTML fixtures and a mock Telegram HTTP server. They do not call real Telegram.

## Limitations

- This depends on Telegram's public web HTML. If Telegram changes `t.me/s` markup, the parser may need updates.
- It only reads public channels.
- Subscriber statistics are exposed as Telegram renders them, usually as display text rather than normalized numbers.
