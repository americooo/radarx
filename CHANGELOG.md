# Changelog

All notable changes to RadarX are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/).

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
