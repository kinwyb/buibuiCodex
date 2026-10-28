# Repository Guidelines

Go service that runs Codex-backed agents over WebSocket, WeCom, and an MCP server. Module `github.com/kinwyb/buibuiCodex`, Go 1.27.

## Out of Scope

Ignore `codex/jsonRpc/schema/` (`config.go`, `models.go`, `models1.go`). It is quicktype-generated, does not compile (duplicate declarations), and nothing imports it. Do not read, edit, review, build, test, or vet it, and never treat its errors as regressions. Protocol types in use are hand-written in `codex/jsonRpc` (`types.go`, `thread.go`, `turn.go`, `event.go`). The path is also listed in `.cgcignore`. Drop it from repo-wide commands:

```bash
go list ./... | grep -v '/jsonRpc/schema'
```

## Project Structure & Module Organization

- `main.go` wires config, message bus, channels, storage, agents, MCP, and the WS server.
- `core/` holds the agent `Manager`, config, bus, db, cron, and types; `core/ws/` is the frontend channel, split into `protocol`, `conn`, `command` (one file per upstream command), `bridge`, `server`, `preflow`, `replay`, `filestore`.
- `codex/` is the Codex app-server client (`cdm`, `jsonRpc`, `proc/docker`); `channel/` holds IM adapters registered from `init()`; `mcp/` is the MCP server plus business tools (`serv/order`, `serv/finance`); `decision/` is a decision-model client.
- Deeper docs: `core/ws/README.md`, `decision/README.md`, `channel/wxcom/README.md`.

## Build, Test, and Development Commands

```bash
go build $(go list ./... | grep -v '/jsonRpc/schema')          # compile
go vet $(go list ./... | grep -v '/jsonRpc/schema')            # vet (clean today)
go test ./core/ws/... ./channel/... ./mcp/... ./decision/...   # offline tests
go run .                                                       # start the service
```

Never run a bare `go test ./...`: the `core` and `codex/...` tests start Docker and the Codex app-server with hardcoded workspace paths and hang offline. Scope tests to the packages you changed. `main.go` reads its config path from a literal in `main()`; point it at your local `buibui.json`.

## Coding Style & Naming Conventions

- Standard `gofmt` (tabs); keep `gofmt -l` empty for files you touch. No linter config in-repo.
- New files use snake case (`delta_coalescer.go`); legacy camelCase exists (`eventHandler.go`).
- Constructors are `New` or `NewXxx` taking an options struct (`ws.Options`) or functional options (`decision.WithAPIKey`). Wrap errors with `%w`, log through `slog`.
- Doc comments and tool descriptions are Chinese; match the file you edit.

## Testing Guidelines

- Stdlib `testing` plus `httptest` only; testify is not a dependency. Name tests `TestXxx<Behavior>` (`TestChatSendRejectsUnknownAgent`) and share stubs in `helper_test.go`.
- Use `db.NewSQLiteStorage` and fake agent lists so tests stay offline; avoid hardcoded absolute paths.
- Red before your change: `TestHandshakeAndPing`, `TestRejectsUnauthorized`, and `TestTokenAsUserID` in `core/ws/server` expect an unknown token to fall through as `userID`, while identity resolution returns a WeCom bot id. Leave them unless identity mapping is your task.

## Commit & Pull Request Guidelines

- Single Chinese subject prefixed by the affected package: `ws 支持多用户鉴权与会话隔离`, `decision 新增百炼决策模型 API 客户端`. Fixes start with `修复…`; no `feat:` or `fix:` keywords.
- PRs state the behavior change, list the packages built and tested, link issues, and add screenshots for `core/ws` protocol changes.

## Security & Configuration Tips

`buibui.json` is gitignored and holds provider API keys, database credentials, and bot secrets. Never commit it or paste its contents into commits, PRs, or logs. Omit `database` to fall back to SQLite at `<workspace>/buibui.db`; WS uploads go to `tmp/ws/`.
