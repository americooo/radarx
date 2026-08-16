# CLAUDE.md — RadarX

Bu repo — **RadarX**: authorized attack surface monitor. Hozircha zero-dependency
Go CLI (`cmd/radarx`), maqsad — uni **Wails v2 desktop GUI** product'ga aylantirish
(engine'ni buzmasdan, ustiga qatlam qo'shib). To'liq reja: `RADARX_ROADMAP.md`
(repo root'ida emas, alohida saqlanadi — Faza 0–10). Claude Code shu faylni har
sessiyada o'qiydi.

## Loyiha nima

RadarX target domenlarni scan qiladi (subdomain enum, HTTP probe, TLS inspect,
port scan), natijani snapshot sifatida saqlaydi, oldingi snapshot bilan diff
qiladi va yangi topilgan asset'lar haqida xabar beradi (console/desktop/Telegram).
Barcha scan logikasi **sof Go** bilan yozilgan — subfinder/httpx/naabu kabi
tashqi binary'larga bog'liqlik yo'q (bu roadmap'dagi Faza 1 muammosini oldindan
hal qiladi).

## Eng muhim arxitektura qoidasi

**`internal/engine` hech qachon GUI yoki CLI paketlarini import qilmaydi.**
Bog'liqlik faqat bir tomonlama:

```
cmd/radarx (CLI)  ─┐
                    ├──> internal/engine, internal/store, internal/diff, ...
cmd/radarx-gui (GUI, kelajakda) ─┘
```

`internal/engine` faqat `internal/model` ga bog'liq bo'lishi mumkin. Agar engine
ichida `net/http`, `context`, standart kutubxonadan tashqari biror joyda
`cmd/...` yoki (kelajakdagi) Wails/frontend import qilinsa — bu arxitektura buzilishi,
darhol to'xtatilsin. Xuddi shunday store, diff, notify, report, scheduler
paketlari ham GUI'ni bilmasligi kerak — ular ham faqat model'ga va bir-biriga
bog'liq.

## Papka strukturasi

Hozirgi haqiqiy holat:

```
radarx/
├── cmd/
│   └── radarx/              # mavjud CLI (add, scan, list, watch, serve, report, history, test-telegram)
│       └── main.go
├── internal/
│   ├── engine/               # barcha scan logikasi (sof Go, tashqi binary yo'q)
│   │   ├── scanner.go        # Scan(ctx, target, opts) model.Snapshot — asosiy kirish nuqtasi
│   │   ├── subdomain.go      # DNS brute-force
│   │   ├── certtransparency.go # crt.sh (CT log) query
│   │   ├── httpprobe.go      # HTTP probe (status, title, server header)
│   │   ├── portscan.go       # yengil TCP connect port scan
│   │   └── tlscert.go        # TLS sertifikat inspeksiyasi
│   ├── store/                 # persistence (Store interfeysi + JSONStore)
│   ├── diff/                  # ikki snapshot orasidagi farqni hisoblash
│   ├── notify/                 # console/desktop/Telegram bildirishnoma kanallari
│   ├── report/                 # markdown asset inventory
│   ├── scheduler/              # fon rejimidagi davriy scan
│   ├── web/                    # eski JS dashboard (Wails GUI kelganda qayta ko'rib chiqiladi)
│   └── model/                  # sof data turlari (Target, Snapshot, Asset, DiffResult)
├── docs/                        # ARCHITECTURE.md, AUDIT.md va h.k.
└── .claude/commands/            # slash command'lar
```

**Rejalashtirilgan, hali mavjud emas:** `cmd/radarx-gui/` (Wails entry point),
`internal/scope/` (scope enforcement), `frontend/` (React app), SQLite store
implementatsiyasi. Bularni qo'shishdan oldin shu faylni yangilang.

## Naming konvensiyalari

- Fayl nomlari: kichik harf, defissiz, bitta mazmun bitta faylda
  (masalan `httpprobe.go`, `certtransparency.go`, `tlscert.go`).
- Paket nomlari: qisqa, bitta so'z, aniq mas'uliyat (`engine`, `store`, `diff`,
  `notify`, `report`, `scheduler`, `model`, `web`).
- Eksport qilinadigan funksiyalar aniq fe'l bilan boshlanadi: `Scan`, `Compare`,
  `EnumerateSubdomains`, `ProbeHTTP`, `InspectCert`, `ScanPorts`.
- Har bir faylning boshida paket/fayl vazifasini tushuntiruvchi qisqa doc-comment
  bo'lsin (mavjud kod uslubiga qarang — inglizcha, aniq, "nega" ga urg'u beradi).
- Izohlar aralash bo'lishi mumkin: kod va doc-comment'lar inglizcha, ishchi
  muhokama/rejalashtirish izohlari o'zbekcha bo'lishi mumkin.

## Scope va xavfsizlik qoidalari

RadarX — offensive security tool. Faqat **authorized** target'lar uchun mo'ljallangan
(o'z infratuzilma yoki ruxsat berilgan bug bounty scope).

1. Claude Code hech qachon haqiqiy target'ga qarshi scan buyrug'ini o'zi ishga
   tushirmaydi — faqat kod yozadi/tuzatadi, ishga tushirishni foydalanuvchi qiladi.
2. Yangi scanner yoki funksiya qo'shganda ham u faqat read-only operatsiyalar
   bo'lsin (DNS lookup, HTTP GET, TLS handshake, TCP connect) — invasive/exploit
   kod yozilmaydi.
3. Rate limiting va "good citizen" xatti-harakati default bo'lib qolishi kerak
   (masalan `scanner.go` dagi `sem := make(chan struct{}, 20)` kabi concurrency
   cheklovlari) — agressiv flag/parametr qo'shilganda ham default past bo'lsin.
4. Faza 6 (`internal/scope`) kelguncha ham, har qanday yangi kodni yozishda scope
   tashqarisiga chiqish imkoniyatini oldindan o'ylab qo'ying (masalan CT log'dan
   kelgan subdomain'lar root domenga tegishli ekanini tekshirish).

## context.Context qoidasi

**Har bir scanner funksiyasi `context.Context` ni birinchi parametr sifatida
qabul qilishi SHART** (`ctx context.Context, ...`). Bu — kelajakdagi GUI'dagi
"Stop" tugmasi ishlashi uchun yagona mexanizm. Yangi tarmoq so'rovi/DNS
so'rovi/tashqi chaqiruv yozilganda ctx cancellation'ga hurmat qilinishi shart
(masalan `net.Dialer{}.DialContext`, `http.NewRequestWithContext`).

**Bilib qo'ying (hozircha o'zgartirilmaydi, Faza 1 uchun qoldirilgan):**
`engine.Scan()` hozir bitta to'liq `model.Snapshot` ni bir martaga qaytaradi.
Roadmap Faza 1 buni channel-based streaming interfeysga o'tkazishni talab qiladi
(masalan `Run(ctx, target) (<-chan Result, error)`), shunda GUI ham, CLI ham
natijalarni real-time iste'mol qila oladi. Bu — kelajakdagi ish, Faza 0'da
qo'l tegmaydi. Tafsilotlar uchun `docs/AUDIT.md` ga qarang.

## CLI'ni buzmaslik qoidasi

`cmd/radarx/main.go` — mavjud, ishlaydigan CLI (add, scan, list, watch, serve,
report, history, test-telegram). Bu CLI **engine to'g'ri ajratilganining tirik
testi**: agar engine yaxshi ajratilgan bo'lsa, CLI hech qachon buzilmasligi kerak.

- Engine yoki store'ga o'zgartirish kiritilganda avval `go build ./...` va
  `go test ./...` ishlatib CLI hali kompilyatsiya qilinishini va testlar
  o'tishini tekshiring.
- Mavjud CLI buyruqlari, flag'lari va chiqish formatini o'zgartirmang (agar
  aniq vazifa buni talab qilmasa).
- Yangi funksionallik qo'shilganda avval engine/store darajasida qo'shing,
  keyin CLI (yoki kelajakdagi GUI) uni chaqirsin — hech qachon aksincha emas.
