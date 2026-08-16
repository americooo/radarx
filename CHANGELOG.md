# Changelog

All notable changes to RadarX are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [1.0.0] — 2026-08-16

RadarX grows from a CLI-only recon tool into a full desktop product, with a
plugin system for detection logic on top of the existing monitoring core.

### Added
- **Desktop GUI** (Wails v2 + React/TypeScript/Tailwind), sharing the same
  engine and SQLite store as the CLI:
  - Targets, Scan (live streaming results), Results, Diff, and Settings
    screens
  - First-run onboarding (branded splash + language picker) and full
    Uzbek/English i18n, switchable anytime from Settings
  - Continuous monitoring toggle, Telegram setup (token entry + automatic or
    manual chat-id detection), scope management, and per-module
    enable/disable — all from the GUI, no config files to hand-edit
- **SQLite-backed persistence for everything**: targets, snapshots, scope,
  and settings (Telegram credentials, module toggles) all live in one
  `~/.radarx/radarx.db`; legacy `scope.txt`/`config.json` are migrated in
  automatically on first run
- **Detection module system** (`internal/modules`): a self-registering
  plugin architecture (Module interface + registry + orchestrator, each
  module gets its own persistent state) wired into both the background
  scheduler and manual scans. Eight modules ship in this release:
  - Subdomain takeover (CNAME fingerprinting, 12 common services)
  - Certificate Transparency log monitoring (crt.sh, new-cert signal)
  - DNS record monitoring (A/AAAA/MX/TXT/NS change detection)
  - JS file diff (content-hash tracking across scans)
  - Exposed files detection (`.git`, `.env`, backups, etc.)
  - Cloud storage bucket discovery (S3/GCS, open vs. closed)
  - Rate-limit protection signal (never brute-forces — two probe requests,
    read-only)
  - Optional Nuclei integration (wraps an external `nuclei` binary if
    installed; silently skipped otherwise, keeping the core binary
    self-contained)
- **Release pipeline**: tag-triggered GitHub Actions workflow builds the CLI
  for 4 platforms (CGO-free cross-compile) and the GUI natively on
  Linux/Windows/macOS runners, publishing everything to a GitHub Release
- Custom RadarX app icon (radar-sweep glyph, matches the in-app branding)
  replacing the default Wails placeholder

### Changed
- `scope.txt` and `~/.radarx/config.json` are no longer the source of truth
  (auto-migrated into SQLite); they're left on disk untouched but unused
- Engine's `Scan()` is now a thin wrapper over a streaming `ScanStream()`,
  so both the CLI and GUI render results as they're discovered instead of
  waiting for a full scan to finish

## [0.1.0] — 2026-08-06

Initial release.

### Added
- Recon engine (pure Go stdlib, CGO-free):
  - Subdomain enumeration via DNS brute-force with a bounded worker pool
  - Certificate Transparency subdomain source (crt.sh), passive
  - HTTP/HTTPS probing: status code, page title, `Server` header
  - TLS certificate inspection: common name and expiry
  - Lightweight TCP connect port scanner over common ports
- Snapshot diffing: detects NEW / CHANGED / REMOVED assets between scans
- JSON-backed store with atomic writes and snapshot history
- Background scheduler with per-target intervals and a global safety floor
- Notifications:
  - Console
  - Native desktop toast (notify-send / PowerShell / osascript)
  - Telegram push (token & chat id from environment, never hardcoded)
- Local web dashboard (stdlib `net/http` + embedded assets), auto-refreshing
- Markdown report generation (HackerOne-ready asset inventory)
- CLI: `add`, `scan`, `watch`, `serve`, `report`, `history`, `list`,
  `test-telegram`, `version`
- Packaging: GitHub Actions CI, Makefile with cross-compile + checksums,
  GPL-3.0 license, security policy, contributing guide
