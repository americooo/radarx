---
description: internal/engine ga yangi scanner turi qo'shish uchun qadamlar
---

`internal/engine` paketiga yangi scanner (masalan yangi enrichment turi —
DNS record tekshiruvi, WHOIS, favicon hash, JS fayl tahlili va h.k.) qo'shish.
Mavjud fayllar uslubiga qat'iy amal qil: `subdomain.go`, `httpprobe.go`,
`portscan.go`, `tlscert.go`, `certtransparency.go` ga qara.

## Qadamlar

1. **O'qi:** `CLAUDE.md` (ayniqsa scope/xavfsizlik va context.Context
   qoidalari) va `internal/model/types.go` (mavjud `Asset`, `AssetKind`
   turlarini tushunish uchun).
2. **Asset turi kerakmi tekshir:** agar yangi scanner yangi turdagi ma'lumot
   qaytarsa, `internal/model/types.go` dagi `AssetKind` konstantalariga yangi
   qiymat qo'shish kerakmi yoki mavjud `Asset` struct maydonlari yetarlimi —
   hal qil. Minimal o'zgartirish afzal (yangi maydon qo'shsang ham eski
   snapshot'lar JSON deserialize bo'lishda buzilmasligi kerak — hamma yangi
   maydon `omitempty` bo'lsin).
3. **Yangi fayl yarat:** `internal/engine/<qisqa-nom>.go` — bitta fayl, bitta
   mas'uliyat (masalan `internal/engine/favicon.go`). Fayl boshida mavjud
   fayllardagidek qisqa doc-comment yoz: nima qiladi, nega kerak.
4. **Funksiya signature'i:** birinchi parametr har doim
   `ctx context.Context`. Uzoq operatsiyalar (tarmoq so'rovi va h.k.) ctx
   bilan bog'lansin (`DialContext`, `NewRequestWithContext`, yoki `select`
   bilan `ctx.Done()` tekshiruvi). Natija — `(model.Asset, bool)` yoki mavjud
   pattern'larga mos boshqa signature (masalan `InspectCert(ctx, host)
   (model.Asset, bool)` ga o'xshab).
5. **Faqat read-only:** yangi scanner faqat DNS lookup, HTTP GET/HEAD, TLS
   handshake, TCP connect kabi passiv/read-only operatsiyalar bajarsin.
   Invasive yoki exploit xarakteridagi kod yozilmasin (`CLAUDE.md` dagi
   scope/xavfsizlik qoidalariga qara).
6. **`scanner.go` ga ulash:** `Scan()` (yoki refaktordan keyingi `Run()`)
   ichidagi enrichment tsikliga yangi chaqiruvni qo'sh — kerak bo'lsa yangi
   `ScanOptions` flag qo'shib, uni ixtiyoriy qil (default `false`/off), xuddi
   `opts.UseCertLogs` va `opts.ScanPorts` kabi.
7. **Koncurrency:** mavjud `sem := make(chan struct{}, N)` pattern'iga
   mos ravishda ishlating — yangi scanner boshqa host'larni "urib"
   yubormasligi uchun concurrency va timeout cheklovlarini saqlang.
8. **Test yoz:** `internal/engine/<nom>_test.go` — kamida: (a) muvaffaqiyatli
   holat (mock server yoki local listener bilan), (b) ctx cancel qilinganda
   to'g'ri to'xtashi, (c) xato/topilmagan holatda `(zero, false)` qaytarishi.
9. **Tekshir:**
   - `go build ./...`
   - `go vet ./...`
   - `go test -race ./...`
   - `radarx scan <test-domain>` bilan qo'lda sinab, yangi asset kutilganidek
     chiqishini tasdiqla.
10. **Hujjatlashtir:** agar yangi `ScanOptions` flag qo'shilgan bo'lsa,
    `cmd/radarx/main.go` dagi tegishli CLI flag va `usage()` matnini ham
    yangila (agar CLI'dan boshqarilishi kerak bo'lsa).

## Eslatma

`internal/engine` hech qachon `cmd/*` ni import qilmaydi — yangi scanner ham
shu qoidaga bo'ysunadi, faqat `internal/model` ga bog'liq bo'ladi.
