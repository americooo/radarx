# Security Policy

## Responsible use

RadarX is a reconnaissance tool. It performs DNS lookups, HTTP requests, TLS
handshakes, and TCP connect probes against the domains you configure. **Only
point RadarX at infrastructure you own or are explicitly authorized to test**
(for example, assets that are in scope for a bug bounty program you are
participating in). Unauthorized scanning may be illegal in your jurisdiction and
is against the terms of most bug bounty programs.

You are responsible for staying within authorized scope.

## What RadarX does *not* do

- It does not exploit, attack, or attempt to compromise any system.
- It performs only read-only, standard network operations (DNS, HTTP GET, TLS
  handshake, TCP connect) — no raw packets, no SYN scanning, no fuzzing.
- It does not exfiltrate data. All results stay on your machine under `~/.radarx/`.
- It contains no telemetry and makes no network calls except to the targets you
  configure and, if you opt in, Certificate Transparency (crt.sh) and the
  Telegram Bot API for alerts.

## Handling of secrets

RadarX never hardcodes credentials. The Telegram bot token and chat id are read
from environment variables (`RADARX_TG_TOKEN`, `RADARX_TG_CHAT_ID`) and are never
written to disk or logged.

## Reporting a vulnerability

If you find a security issue in RadarX itself, please report it privately via
Telegram [@americo_444](https://t.me/americo_444) or open a GitHub security
advisory. Please do not open a public issue for security-sensitive reports.
