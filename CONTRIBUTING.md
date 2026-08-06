# Contributing to RadarX

Thanks for your interest. RadarX aims to stay **small, local-first, and
dependency-free**. Contributions should respect that.

## Ground rules

1. **Standard library first.** The engine has zero external dependencies and
   cross-compiles with `CGO_ENABLED=0`. Adding a dependency needs a strong
   justification — prefer stdlib.
2. **Read-only recon only.** RadarX never exploits or attacks. Pull requests that
   add offensive/exploitation capability will be declined. Recon, enrichment,
   reporting, and UX are welcome.
3. **Everything must build and pass.** `go build ./...`, `go vet ./...`, and
   `go test ./...` must all pass, and `gofmt` must be clean.

## Development

```bash
git clone https://github.com/americooo/radarx
cd radarx

go build ./...
go vet ./...
go test ./...
gofmt -l .        # should print nothing

go run ./cmd/radarx scan example.com
```

## Project layout

| Path | What lives there |
|---|---|
| `internal/model` | Core data types |
| `internal/engine` | Recon: subdomain, HTTP, TLS, ports, CT, scan orchestration |
| `internal/diff` | Snapshot comparison |
| `internal/store` | Persistence (JSON) |
| `internal/scheduler` | Background scan loop |
| `internal/notify` | Console / desktop / Telegram channels |
| `internal/report` | Markdown report generation |
| `internal/web` | Local dashboard (stdlib http + embedded assets) |
| `cmd/radarx` | CLI entrypoint |

## Commit style

Short, imperative subject lines: `add port scanner`, `fix diff ordering`.
Keep commits focused.
