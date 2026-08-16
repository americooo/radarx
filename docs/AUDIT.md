# RadarX — mavjud kod audit (Faza 0)

Bu hujjat roadmap Faza 0 ning "mavjud kodni audit qiling" talabini bajaradi:
hozirgi holat vs roadmap talablari, aniq gap ro'yxati bilan.

## Hozirgi holat (2026-08-16 holatiga)

- **CLI**: `cmd/radarx/main.go` — to'liq ishlaydigan MVP. Buyruqlar: `add`,
  `scan`, `list`, `watch`, `serve`, `report`, `history`, `test-telegram`.
- **Engine**: `internal/engine/` — `scanner.go` (`Scan(ctx, target, opts)
  model.Snapshot`), `subdomain.go` (DNS brute-force), `certtransparency.go`
  (crt.sh query), `httpprobe.go` (HTTP probe), `portscan.go` (TCP connect
  scan), `tlscert.go` (TLS sertifikat inspeksiya).
- **Tashqi binary bog'liqligi**: **YO'Q.** Barcha scan logikasi sof Go bilan
  yozilgan (standart kutubxona: `net`, `net/http`, `crypto/tls`). subfinder,
  httpx, naabu import qilinmagan, PATH'da chaqirilmagan.
- **Persistence**: `internal/store/store.go` — `Store` interfeysi allaqachon
  mavjud (`SaveTarget`, `ListTargets`, `SaveSnapshot`, `LatestSnapshot`,
  `ListSnapshots`). Yagona implementatsiya: `JSONStore`
  (`~/.radarx/targets/*.json`, `~/.radarx/snapshots/<id>/*.json`).
- **Diff**: `internal/diff/diff.go` — `Compare(old, new Snapshot) DiffResult`
  ishlaydi, test bor (`diff_test.go`).
- **Bildirishnoma**: `internal/notify/` — console, desktop
  (notify-send/osascript/powershell), Telegram.
- **Qo'shimcha**: `internal/report/report.go` (markdown inventory),
  `internal/scheduler/scheduler.go` (davriy scan), `internal/web/server.go`
  (eski JS dashboard, std lib bilan).
- **Repo hujjatlari**: `README.md`, `LICENSE`, `SECURITY.md`,
  `CONTRIBUTING.md`, `CHANGELOG.md`, `.github/workflows/ci.yml` (build/vet/test/
  cross-compile linux+windows+darwin) — bularning barchasi allaqachon mavjud
  (Faza 8'ning bir qismi oldindan bajarilgan).
- **`.claude/`**: yo'q edi, Faza 0 doirasida shu audit bilan birga qo'shildi
  (`.claude/commands/refactor-engine.md`, `add-scanner.md`, `release.md`).

## Roadmap talablariga nisbatan gap'lar

| Sohasi | Roadmap talabi | Hozirgi holat | Gap |
|---|---|---|---|
| Engine interfeysi | Faza 1: yagona `Scanner` interfeysi, `Run(ctx, target) (<-chan Result, error)` — natijalar channel orqali oqib chiqadi | `Scan(ctx, t, opts) model.Snapshot` — bitta to'liq snapshot'ni bir martaga qaytaradi (ichida goroutine+WaitGroup bilan parallel ishlaydi, lekin tashqariga faqat yakuniy natija chiqadi) | Streaming/channel interfeys yo'q. Bu — Faza 1'ning asosiy vazifasi, Faza 0'da qo'l tegmaydi. |
| Progress event'lari | Faza 1: progress/event'lar channel orqali chiqishi, GUI ham CLI ham iste'mol qilishi kerak | Yo'q — `cmdScan` faqat yakuniy son (`discovered %d assets`) va umumiy vaqtni chop etadi | GUI real-time progress ko'rsata olmaydi hozircha. |
| Persistence | Faza 2: `modernc.org/sqlite` (CGo'siz), schema (`targets`, `scans`, `assets`, `changes`) | `JSONStore` (fayl-asosli JSON), `Store` interfeysi allaqachon abstraksiya sifatida tayyor | SQLite implementatsiyasi yo'q, lekin interfeys buni oson qiladi (kod komментida aniq: "swap in a SQLite implementation later"). |
| GUI | Faza 3-4: Wails v2 + React/TS/Tailwind/shadcn desktop app | Yo'q — faqat CLI va eski std-lib JS dashboard (`internal/web`) | `cmd/radarx-gui/`, `frontend/` hali mavjud emas. |
| Continuous monitoring (in-app) | Faza 5: GUI ichida scheduler, tray, notification | `internal/scheduler` CLI (`watch` buyrug'i) orqali allaqachon ishlaydi, GUI integratsiyasi yo'q | Scheduler logikasi tayyor, faqat GUI'ga ulanishi kerak — refaktor emas, integratsiya vazifasi. |
| Scope enforcement | Faza 6: majburiy scope tasdiqlash, `scope.txt` mantig'i, scan bloklash | Yo'q — CLI hech qanday scope tekshiruvisiz istalgan domenni scan qiladi | `internal/scope` paketi hali yaratilmagan. |
| Tashqi binary muammosi | Faza 7: ProjectDiscovery tool'larini Go kutubxonasi sifatida import qilish | **Allaqachon yo'q muammo** — engine boshidanoq sof Go bilan yozilgan, tashqi binary'ga bog'liqlik yo'q | Gap yo'q, Faza 7'ning bu qismi oldindan bajarilgan. |
| Release avtomatizatsiyasi | Faza 7: `git tag` push'da uch platforma binary avtomatik Release'ga chiqishi | `.github/workflows/ci.yml` bor (build/vet/test + cross-compile linux/windows/darwin binary artifact sifatida), lekin GitHub Release'ga yuklovchi `release.yml` yo'q | `release.yml` workflow hali yozilmagan. |
| Open-source hujjatlar | Faza 8: README, LICENSE, SECURITY, CONTRIBUTING, CHANGELOG | Barchasi allaqachon repo root'ida mavjud | Gap yo'q — Faza 8'ning katta qismi oldindan bajarilgan, faqat screenshot/demo GIF qo'shish qolishi mumkin. |
| Claude Code jihozlash | Faza 0: CLAUDE.md, slash command'lar, ARCHITECTURE.md | Shu audit bilan birga qo'shildi (`CLAUDE.md`, `docs/ARCHITECTURE.md`, `.claude/commands/*`) | Bajarildi. |

## Xulosa — Faza 0 "tugadi" mezoni

- [x] Repo'da aniq target struktura hujjatlashtirilgan (`CLAUDE.md`,
      `docs/ARCHITECTURE.md`).
- [x] `.claude/commands/` slash command'lar bilan tayyorlangan.
- [x] Mavjud kod audit qilingan: scan logikasi joyi aniq, tashqi binary
      bog'liqligi yo'qligi tasdiqlangan (yuqoridagi jadval).
- [ ] Faza 1 (streaming engine interfeysi) — **hali boshlanmagan**, keyingi
      qadam.
- [ ] Faza 2 (SQLite) — hali boshlanmagan.
- [ ] Faza 3+ (Wails GUI) — hali boshlanmagan.

Faza 0 doirasida hech qanday Go kod fayli o'zgartirilmadi — faqat hujjat va
slash command fayllari qo'shildi.
