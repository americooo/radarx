# RadarX arxitekturasi

Bu hujjat qatlamlar orasidagi bog'liqlikni tasvirlaydi. Asosiy qoida:
**`internal/engine` hech narsani (GUI'ni ham, CLI'ni ham) import qilmaydi.**
Bog'liqlik faqat bir tomonlama — yuqoridan pastga.

## Diagramma

```
                        ┌─────────────────────────┐
                        │   cmd/radarx (CLI)       │   mavjud, ishlaydi
                        │   main.go                │   add/scan/list/watch/...
                        └────────────┬─────────────┘
                                     │
                        ┌────────────┴─────────────┐
                        │  cmd/radarx-gui (Wails)   │   REJALASHTIRILGAN
                        │  (hali yo'q — Faza 3)     │   frontend/ (React+TS)
                        └────────────┬─────────────┘
                                     │  ikkalasi ham quyidagilarni chaqiradi:
        ┌────────────────────┬──────┴───────┬────────────────────┐
        ▼                    ▼               ▼                    ▼
┌───────────────┐   ┌────────────────┐ ┌───────────┐    ┌────────────────┐
│ internal/engine│   │ internal/store │ │internal/  │    │ internal/notify │
│ scanner.go     │   │ store.go       │ │diff       │    │ scheduler       │
│ subdomain.go   │   │ (Store iface,  │ │diff.go    │    │ report          │
│ httpprobe.go   │   │  JSONStore)    │ └───────────┘    │ web             │
│ portscan.go    │   └────────────────┘                  └────────────────┘
│ tlscert.go     │
│ certtransparency.go │
└───────┬────────┘
        │ faqat bog'liq
        ▼
┌───────────────┐
│ internal/model │   sof data turlari, xatti-harakatsiz (Target, Snapshot,
│ types.go       │   Asset, DiffResult) — hamma paket shu yerga bog'liq bo'lishi mumkin
└───────────────┘
```

## Qatlamlar

- **`internal/model`** (`internal/model/types.go`) — loyihaning umumiy "til"i:
  `Target`, `Asset`, `Snapshot`, `Change`, `DiffResult`. Hech qanday xatti-harakat
  yo'q, faqat struct'lar — shuning uchun JSON'ga ham, keyinchalik SQLite'ga ham
  o'zgarishsiz map qilinadi. Har bir boshqa paket shu yerga bog'liq bo'lishi mumkin.

- **`internal/engine`** (`internal/engine/scanner.go` va yonidagi fayllar) —
  barcha scan logikasi: subdomain enumeratsiya (`subdomain.go`, DNS brute-force),
  Certificate Transparency query (`certtransparency.go`, crt.sh), HTTP probe
  (`httpprobe.go`), port scan (`portscan.go`), TLS inspeksiya (`tlscert.go`).
  Kirish nuqtasi — `Scan(ctx, target, opts) model.Snapshot`. Faqat `model` ga
  bog'liq, tashqi binary yo'q, GUI/CLI haqida hech narsa bilmaydi.

- **`internal/store`** (`internal/store/store.go`) — persistence. `Store`
  interfeysi (`SaveTarget`, `ListTargets`, `SaveSnapshot`, `LatestSnapshot`,
  `ListSnapshots`) — seam nuqtasi. Hozir yagona implementatsiya `JSONStore`
  (`~/.radarx/targets/*.json`, `~/.radarx/snapshots/<id>/*.json`). Faza 2'da
  SQLite implementatsiyasi shu interfeysni implement qilib qo'shiladi, engine
  yoki GUI kodiga tegilmaydi.

- **`internal/diff`** (`internal/diff/diff.go`) — ikkita `Snapshot` ni
  solishtirib `DiffResult` qaytaradi (yangi/o'zgargan/yo'qolgan asset'lar).
  RadarX'ning asosiy "value"i shu yerda.

- **`internal/notify`, `internal/report`, `internal/scheduler`** — diff
  natijasini iste'mol qiluvchi yordamchi qatlamlar: bildirishnoma yuborish
  (console/desktop/Telegram), markdown report generatsiya, fon rejimidagi
  davriy scan orkestratsiyasi. Ular ham faqat `model`, `engine`, `store`,
  `diff` ga bog'liq — bir-birlariga ustma-ust bog'liqlik yo'q.

- **`cmd/radarx`** (`cmd/radarx/main.go`) — mavjud CLI. Engine/store/diff/notify
  paketlarini import qilib chaqiradi. Buzilmasligi kerak bo'lgan "tirik test".

- **`cmd/radarx-gui`** (hali yo'q, Faza 3'da qo'shiladi) — Wails v2 entry
  point. Xuddi CLI kabi faqat yuqoridagi paketlarni chaqiradi, lekin natijalarni
  Wails `EmitEvent` orqali React frontend'ga uzatadi. `internal/engine` bu
  paket haqida hech narsa bilmaydi va bilmasligi ham kerak.

- **`internal/web`** (`internal/web/server.go`) — eski, faqat standart
  kutubxona bilan yozilgan JS dashboard. Wails GUI kelganda undan foydalanish
  yoki almashtirish hal qilinadi; hozircha CLI'ning `serve`/`watch --web`
  buyruqlari orqali ishlaydi va tegilmaydi.

## Nega bu muhim

Agar `internal/engine` hech qachon `cmd/*` yoki (kelajakdagi) Wails/frontend
kodini import qilmasa, ikkita mustaqil "iste'molchi" (CLI va GUI) bir xil
scan mantig'i ustida ishlay oladi, va engine'ni GUI framework'idan mustaqil
test qilish mumkin bo'ladi. Bu — roadmapdagi Faza 1 ning butun maqsadi.
