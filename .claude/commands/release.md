---
description: Yangi RadarX versiyasini chiqarish checklist'i
---

Yangi RadarX versiyasini tag qilib chiqarish uchun checklist.

> **Eslatma:** Hozircha `.github/workflows/` ichida faqat `ci.yml` bor
> (build/vet/test + cross-compile artifact sifatida). GitHub Release'ga
> avtomatik binary yuklovchi `release.yml` workflow **hali yo'q** — bu
> roadmap Faza 7 ("Packaging va distribution") vazifasi. Agar `release.yml`
> mavjud bo'lmasa, avval Faza 7'ni bajaring (Wails build uch platforma uchun,
> `.github/workflows/release.yml` yozish), keyin shu checklist'ga qayting.
> `docs/AUDIT.md` da joriy holat yozilgan.

## Oldindan tekshir

- [ ] `.github/workflows/release.yml` mavjudligini tekshir. Yo'q bo'lsa —
      to'xta, avval Faza 7'ni bajaring.
- [ ] `main` branch clean va CI (`ci.yml`) yashil ekanini tekshir
      (`gofmt`, `go vet`, `go test -race`, build, cross-compile).
- [ ] Barcha rejalashtirilgan Faza vazifalari shu versiya uchun tugallanganini
      tasdiqla (roadmap fayldagi tegishli Faza checklist'iga qara).

## Versiya va CHANGELOG

- [ ] `CHANGELOG.md` ni yangila — yangi versiya sarlavhasi ostida shu
      release'dagi o'zgarishlarni ro'yxatla (mavjud formatga mos: Added/
      Changed/Fixed bo'limlari).
- [ ] Agar `main.go` dagi `version` o'zgaruvchisi build-time ldflags orqali
      inject qilinsa, release workflow shu versiyani to'g'ri uzatishini
      tekshir (`-ldflags "-X main.version=vX.Y.Z"`).
- [ ] `README.md` dagi versiya/eslatmalarni kerak bo'lsa yangila.

## Test

- [ ] `go build ./...` va `go test -race ./...` mahalliy ravishda xatosiz.
- [ ] CLI'ni qo'lda sinab ko'r: `add`, `scan`, `list`, `watch`, `serve`,
      `report`, `history`, `test-telegram` — asosiy oqim buzilmaganini
      tasdiqla (`CLAUDE.md` dagi "CLI'ni buzmaslik qoidasi").
- [ ] Agar Wails GUI mavjud bo'lsa (Faza 3+), `wails build` uch platforma
      uchun xatosiz ishlashini tekshir.

## Tag va release

- [ ] Commit qil: `git commit -am "chore: prepare vX.Y.Z release"`.
- [ ] Tag qo'y: `git tag vX.Y.Z` (semver'ga rioya qil).
- [ ] Push qil: `git push origin main --tags`.
- [ ] `release.yml` workflow avtomatik ishga tushishini kuzat (GitHub
      Actions tab). Uch platforma binary (`radarx-windows-amd64.exe`,
      `radarx-darwin-arm64`, `radarx-linux-amd64`) Release'ga yuklanganini
      tasdiqla.
- [ ] GitHub Release sahifasida CHANGELOG matnidan qisqa release notes
      qo'shilganini tekshir.

## Release'dan keyin

- [ ] Binary'larni yuklab olib, kamida bitta platformada qo'lda ishga
      tushirib tasdiqla (imzosiz binary uchun SmartScreen/Gatekeeper
      ogohlantirishi normal — README'da eslatilganini tekshir).
- [ ] Agar katta funksionallik (masalan yangi Faza) tugagan bo'lsa,
      `docs/AUDIT.md` dagi gap jadvalini yangila.
