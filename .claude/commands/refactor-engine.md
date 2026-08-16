---
description: Engine paketini channel-based Scanner interfeysiga o'tkazish (Faza 1)
---

`internal/engine` paketini roadmap Faza 1 talabiga muvofiq bir martalik
`Scan(ctx, target, opts) model.Snapshot` dan channel-based streaming
interfeysga o'tkazish. Bu — butun loyihaning eng og'ir va eng muhim refaktori.
Shoshilma, har qadamdan keyin `go build ./... && go test ./...` ishga tushir.

## Oldindan o'qish

1. `CLAUDE.md` ni o'qi — ayniqsa "eng muhim arxitektura qoidasi" va
   "context.Context qoidasi" bo'limlari.
2. `docs/ARCHITECTURE.md` va `docs/AUDIT.md` ni o'qi — hozirgi holat va
   nima o'zgarishi kerakligini tushun.
3. `internal/engine/scanner.go` ni to'liq o'qi — hozirgi `Scan()` funksiyasi
   nima qilishini tushun (subdomain discovery → resolve → HTTP probe + TLS +
   port scan, hammasi bitta snapshot'ga yig'iladi).

## Checklist

- [ ] Yagona `Result` turini aniqla (`internal/engine` ichida, masalan
      `scanner.go` yoki yangi `engine.go` faylida). Bu tur bitta topilgan
      asset yoki progress event'ni ifodalaydi (`model.Asset` ni o'rab olishi
      yoki alohida progress/error variantlarini qo'shishi mumkin).
- [ ] Yagona `Scanner` interfeysi yoki funksiya signature'ini belgila:
      `Run(ctx context.Context, t model.Target, opts ScanOptions) (<-chan Result, error)`.
      `ctx` cancel qilinganda channel yopilishi va goroutine'lar toza
      to'xtashi shart (leak yo'q).
- [ ] Hozirgi `Scan()` ichidagi pipeline'ni (`EnumerateSubdomains` →
      `resolveHosts`/CT → `ProbeHTTP`/`InspectCert`/`ScanPorts`, hammasi
      `sync.WaitGroup` + semaphore bilan) channel'ga yozadigan qilib qayta
      quring — har topilgan asset darhol channel'ga yuborilsin, WaitGroup
      tugagach channel yopilsin (`close(ch)`).
- [ ] Eski `Scan()` ni **darhol o'chirma** — avval yangi `Run()` ni qo'sh,
      ikkalasi ham compile bo'lishini tekshir, keyin CLI'ni yangi funksiyaga
      o'tkaz, shundan keyingina eskisini o'chir (yoki ichki helper sifatida
      qoldirib, `Run()` uni chaqirsin).
- [ ] `cmd/radarx/main.go` dagi `cmdScan` va `internal/scheduler/scheduler.go`
      ni yangi channel-based `Run()` ustiga qayta ul: natijalarni channel'dan
      o'qib, `model.Snapshot` ga yig'ib, keyingi diff/save mantig'ini o'zgarishsiz
      qoldir (CLI chiqish formati bir xil bo'lib qolishi kerak).
- [ ] Har bir scanner funksiyasi (`EnumerateSubdomains`, `ProbeHTTP`,
      `InspectCert`, `ScanPorts`, `EnumerateCertTransparency`) `ctx` ni
      to'g'ri hurmat qilishini tekshir — uzoq davom etadigan operatsiyalarda
      `select { case <-ctx.Done(): return ... }` yoki `DialContext`/
      `NewRequestWithContext` ishlatilganini tasdiqla.
- [ ] "Stop" ssenariysini qo'lda sinab ko'r: `ctx` ni tezda cancel qilib
      (masalan qisqa timeout bilan) goroutine leak yo'qligini tekshir
      (`go test -race` yordam beradi).
- [ ] Engine uchun unit test yoz (masalan `internal/engine/scanner_test.go`):
      kichik/mock target bilan `Run()` chaqirilganda channel to'g'ri natija
      berishini va ctx cancel qilinganda to'xtashini tekshiradigan test.
- [ ] `go build ./...`, `go vet ./...`, `go test -race ./...` — hammasi
      xatosiz o'tsin.
- [ ] CLI'ni qo'lda ishga tushirib tekshir: `radarx scan <domain>` avvalgidek
      ishlashi, chiqish formati o'zgarmaganini ko'r.

## Diqqat

- `internal/engine` hech qachon `cmd/*` ni import qilmaydi — refaktor
  paytida ham bu qoida buzilmasin.
- Bu refaktor CLI'ning tashqi xatti-harakatini (buyruqlar, flag'lar, chiqish
  formati) o'zgartirmasligi kerak — faqat ichki implementatsiya o'zgaradi.
- Katta refaktorni bitta commit'ga yig'ib qo'ymang — mumkin bo'lsa mantiqiy
  bosqichlarga bo'ling (Result turi → Run() qo'shish → CLI ko'chirish →
  scheduler ko'chirish → eski Scan() olib tashlash → testlar).
