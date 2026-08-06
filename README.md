# RadarX — Attack Surface Monitor

**Local-first continuous recon for Windows, Linux, and macOS.** RadarX watches the
domains you're authorized to test, re-scans them on a schedule, and tells you the
moment a *new* asset appears — a new subdomain, a service that came online, a
fresh TLS cert, a newly open port. In bug bounty and red team recon, being first
to see a new asset is the whole edge.

No cloud. No account. No telemetry. One static binary, zero dependencies.

> ⚠️ **Authorization only.** Monitor scopes you are explicitly permitted to test
> (your own infrastructure or in-scope bug bounty programs). RadarX performs
> read-only DNS lookups, HTTP GETs, TLS handshakes, and TCP connect probes — but
> you are responsible for staying in scope. See [SECURITY.md](SECURITY.md).

---

## Why it exists

Recon isn't a one-time job — attack surface drifts every day. Companies deploy new
subdomains, flip a `403` to `200`, spin up a staging box. RadarX turns recon into a
background process: **scan → snapshot → diff against last time → alert on what's new.**

## Features

- **Subdomain discovery** — DNS brute-force + optional Certificate Transparency (crt.sh)
- **HTTP probing** — status code, page title, `Server` header
- **TLS inspection** — certificate common name and expiry
- **Port scanning** — lightweight TCP connect scan on common ports
- **Snapshot diffing** — detects NEW / CHANGED / REMOVED assets between scans
- **Background scheduler** — scans each target on its own interval
- **Alerts** — console, native desktop toast, and Telegram push to your phone
- **Local web dashboard** — live view at `localhost:7777`, no browser toolchain needed
- **Markdown reports** — HackerOne-ready asset inventory

Everything is **pure Go standard library** — cross-compiles with a single command,
no CGO, no node, no external services.

## Architecture

```
Target (root domain)
     │
     ▼
┌──────────── Scan cycle (engine) ────────────┐
│  subdomain enum  (DNS brute + crt.sh)        │
│  HTTP probe      (status / title / server)   │
│  TLS inspect     (cert CN / expiry)          │
│  port scan       (TCP connect)               │
└──────────────────────────────────────────────┘
     │
     ▼   Snapshot  ──►  Store (JSON under ~/.radarx)
     │
     ▼   diff.Compare(previous, current)
     │
     ▼   NEW / CHANGED / REMOVED
     │        │
     │        ├──► console
     │        ├──► desktop toast
     │        └──► Telegram push
     │
     ▼   web dashboard (localhost)
```

| Package | Responsibility |
|---|---|
| `internal/model` | Core types: `Target`, `Asset`, `Snapshot`, `DiffResult` |
| `internal/engine` | Recon: subdomain, HTTP, TLS, ports, CT, scan orchestration |
| `internal/diff` | Snapshot comparison — the competitive core |
| `internal/store` | Persistence (JSON; SQLite-ready interface) |
| `internal/scheduler` | Background scan loop |
| `internal/notify` | Console / desktop / Telegram channels |
| `internal/report` | Markdown report generation |
| `internal/web` | Local dashboard (stdlib http + embedded assets) |
| `cmd/radarx` | CLI entrypoint |

## Install

Requires Go 1.22+.

```bash
git clone https://github.com/americooo/radarx
cd radarx
go build -o radarx ./cmd/radarx
```

Or with the Makefile:

```bash
make build      # builds ./radarx
make release    # cross-compiles all platforms into dist/ with SHA-256 checksums
```

### Cross-compile manually

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o radarx.exe ./cmd/radarx
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o radarx     ./cmd/radarx
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o radarx     ./cmd/radarx
```

Because the engine is CGO-free, these produce fully static binaries with zero
runtime dependencies.

## Usage

```bash
# Register a scope
radarx add example.com --label "Acme BBP" --interval 30

# One-off scan (diffs against the last snapshot)
radarx scan example.com                 # DNS brute-force only
radarx scan example.com --ct            # + Certificate Transparency
radarx scan example.com --ct --ports    # + port scan

# Continuous monitoring in the background (this is the point of RadarX)
radarx watch                            # scan each target on its interval
radarx watch --web 127.0.0.1:7777       # + live dashboard

# Just the dashboard
radarx serve

# Generate a HackerOne-ready markdown inventory
radarx report example.com --out example.md

# Inspect change history
radarx history example.com

# Manage
radarx list
radarx version
```

Data lives under `~/.radarx/` (targets + snapshot history). It is **never** committed
or uploaded.

### Telegram alerts (pushed to your phone)

RadarX can push every change to Telegram so you catch new assets even when away
from the machine. Credentials come from the environment — **never hardcoded**.

```bash
# 1. Create a bot via @BotFather, copy the token
# 2. Message your bot once, then find your chat id
export RADARX_TG_TOKEN="123456:ABC-your-token"
export RADARX_TG_CHAT_ID="987654321"

# 3. Confirm it works
radarx test-telegram

# 4. Alerts now flow automatically while watching
radarx watch
```

### Example output

```
scanning example.com ...
discovered 14 assets in 3.2s

=== CHANGES (2 new) ===
  [+] NEW      subdomain admin.example.com
  [+] NEW      endpoint  https://admin.example.com (200) — Admin Console
  [~] CHANGED  endpoint  https://api.example.com  (status_code,title)
```

## Dashboard

`radarx serve` (or `radarx watch --web ...`) serves a dark, read-only dashboard at
`http://127.0.0.1:7777`. It auto-refreshes every 5s and shows, per target: current
subdomains, HTTP endpoints, open ports, TLS certs, and the most recent changes.

## Roadmap

- [x] Core engine (subdomain / HTTP / TLS / ports) + snapshot diffing
- [x] JSON store, CLI, cross-compile
- [x] Background scheduler (per-target intervals, safety floor)
- [x] Notifications — console, desktop toast, Telegram (secrets from env)
- [x] Certificate Transparency source (crt.sh)
- [x] Local web dashboard (stdlib, embedded assets)
- [x] Markdown reports + change history
- [x] Packaging — CI, Makefile, checksums, LICENSE, SECURITY.md
- [ ] SQLite store for richer history queries
- [ ] Passive DNS / Wayback subdomain sources
- [ ] Optional native desktop wrapper (Wails)

## License

[GPL-3.0](LICENSE) — © 2026 Amirbek Rakhmatullayev
