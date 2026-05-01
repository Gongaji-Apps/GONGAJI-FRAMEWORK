# CLAUDE.md

Panduan untuk AI assistant (Claude Code) saat bekerja di repo `GONGAJI-FRAMEWORK`. Baca file ini sebelum melakukan perubahan apapun.

---

## Apa repo ini

Internal Go framework untuk REST API service di GoNgaji. Mengonsolidasikan pola yang berulang di 19+ service: response, error handling, validation, repository, HTTP client, integrasi pihak ketiga (WhatsApp, FCM, Cloud Tasks, SMTP), caching, scheduler, JWT auth.

**Bukan** aplikasi standalone — repo ini diimport oleh service-service lain.

Module path: `github.com/Gongaji-Apps/GONGAJI-FRAMEWORK`
Go version: `1.25`

---

## Cara orient diri

1. Baca [`README.md`](README.md) — overview + package matrix
2. Baca [`docs/packages.md`](docs/packages.md) — referensi setiap package
3. Lihat [`CHANGELOG.md`](CHANGELOG.md) — riwayat perubahan + breaking changes per version

---

## Commands

| Tujuan | Command |
|---|---|
| Build semua package | `go build ./...` |
| Static analysis | `go vet ./...` |
| Run semua test | `go test ./...` |
| Run test satu package + verbose | `go test ./<package>/... -v` |
| Sync deps | `go mod tidy` |

**Wajib lulus sebelum commit:** `go build ./...`, `go vet ./...`, `go test ./...`. Tidak ada CI yet — quality gate manual.

---

## Konvensi kode

### Naming (Go convention, **wajib**)

- Exported types/functions: `PascalCase`
- Local variables / unexported: `camelCase`
- **TIDAK ADA** underscore di nama Go (`Base_Repository` ❌, `BaseRepository` ✅)
- Constants enum-style: `UPPER_SNAKE_CASE` untuk **value** (mis. `Code = "BAD_REQUEST"`), tapi nama identifier tetap PascalCase (`BadRequest`)
- JSON tag tetap `snake_case` di response struct (kontrak public ke client mobile/legacy — jangan diubah)

Pelanggaran convention sebelumnya sudah dibersihkan di v0.2.0–v0.4.0. **Jangan perkenalkan ulang.**

### Struktur package

Setiap package punya:
- Satu file utama `<package>.go`
- Test di `<package>_test.go` (atau `_test.go` per file kalau besar)
- Doc comment di package declaration (`// Package foo provides ...`)
- Minimal satu contoh penggunaan di doc comment

### Error handling

- Return `*errors.AppError` dari service/repository, **bukan** raw `error` atau `fmt.Errorf`
- Constructor: `errors.NewBadRequest`, `errors.NewNotFound`, dll.
- Pesan error adalah **untuk user**, bukan log. Tulis "Email diblokir." bukan "failed to validate email field"
- Untuk DB error: pakai `errors.NormalizeDBError(err, tableName)` — auto-map duplicate/FK/NOT NULL
- Tidak boleh mutate global error state (lihat anti-pola di v0.2.0 release notes — `domain.ErrInternalServerError.Message = ...` adalah race condition)

### Generics

- `BaseRepository[T any]`, `result.ObjectResult[T]`, `result.ArrayResult[T]`, `binding.JSON[T]` — pola sudah established
- Pakai generic untuk repository dan binding; jangan untuk hal yang lebih sederhana dengan interface

### Context

- Semua public function yang melakukan I/O (DB, HTTP, file) **wajib** menerima `ctx context.Context` sebagai parameter pertama
- Jangan pernah `context.Background()` di service/repository — propagate dari caller
- Background goroutine: propagate ctx eksplisit dari caller, bukan bikin ctx baru

### Logger

- Optional di sebagian besar package (pattern: terima `Logger` interface di `Config`, nil-safe)
- Jangan import `logger` package langsung di package lain — pakai interface kecil di package lokal supaya consumer bisa swap

---

## Arsitektur

### Layer dan dependency direction

```
Service consumer code (api-service)
    │
    ▼
Web layer:    response, errors, binding, validator, normalizer
    │
    ▼
Domain layer: result, query, pagination, contextx
    │
    ▼
Infra layer:  database/repository, httputil, mailer, cache, scheduler, ...
    │
    ▼
External:     gorm, gin, redis-go, firebase-admin, ...
```

**Aturan dependency:**
- Lower-layer **boleh** depend ke higher-layer **interface saja** (mis. `httputil.Logger` interface, bukan `logger.Logger` concrete)
- Upper-layer **bebas** depend ke lower-layer
- Tidak boleh circular import — kalau muncul, redesign

### Tested vs untested

**Sudah tested**: `converter`, `crypto/rsa`, `httputil`, `messaging/whatsapp`, `mailer`, `cloudtask`, `notification/fcm`, `scheduler`, `cache`, `cache/redis`, `authentication/jwt`, `authentication/middleware`.

**Belum tested**: `database/repository`, `binding`, `validator`, `response`, `errors`, `query`, `result`, `pagination`, `formatter` (kecuali converter), `random`, `timeutil`, `tracer`, `storage/gcs`. Tambahkan test saat menyentuh kode di sini.

### Pattern dependency injection di tested packages

Setiap package yang melakukan I/O eksternal punya **dependency-injection seam** untuk hermetic testing:
- `mailer`: `Mailer.sendFn` (default = real SMTP)
- `notification/fcm`: `Client.msg` interface (default = real Firebase)
- `cloudtask`: pure `buildTask` extracted from `Create`
- `cache/redis`: `NewFromClient` for borrowed client (used by miniredis tests)
- `httputil`: `Config.HTTPClient` override
- `authentication/jwt`: `Strategy.Validator` callback

Pertahankan pola ini di package baru — testability over convenience.

---

## Aturan untuk perubahan

### Setiap penambahan/perubahan function → update docs (HARD RULE)

Per direktif user (2026-04-30), setiap kali function ditambah, di-rename, atau signature diubah:

1. Update [`docs/packages.md`](docs/packages.md) — section package terkait
2. Update [`docs/recipes.md`](docs/recipes.md) kalau pola pemakaian berubah signifikan
3. Update [`README.md`](README.md) kalau menambah package baru
4. Update [`CHANGELOG.md`](CHANGELOG.md) — section unreleased atau next version
5. Kalau breaking change: tambahkan tabel "Removed" / "Changed" di CHANGELOG

Jangan commit perubahan code tanpa doc update yang sesuai. Tidak ada exception.

### Breaking change protocol

1. Default: tambah type/function aliases dengan `// Deprecated: use X instead` selama satu minor version
2. Major bump (`v0.x.0` → `v0.x+1.0`): boleh hapus aliases yang sudah deprecated 1 version
3. CHANGELOG **wajib** punya tabel "Removed" / "Changed" dengan kolom "Use instead" + "Available since"
4. PR description **wajib** punya pre-merge checklist memastikan semua 19 service consumer sudah migrate

Lihat PR #10 (cleanup deprecated aliases) untuk template lengkap.

### PR workflow

Setiap perubahan = satu PR. Ukuran PR sebaiknya kecil — satu package, satu fitur. Format:

- **Title**: imperative, singkat ("Add X package", "Fix Y bug", "Remove deprecated Z")
- **Body sections**:
  1. `## Summary` — apa berubah, kenapa
  2. `## API` (untuk package baru) — code example
  3. `## Behavior` (kalau ada nuansa) — quirks, defaults, edge cases
  4. `## Tests` — list cakupan test
  5. `## Dependencies` — kalau menambah dep baru
  6. `## Test plan` — checklist `go build`/`vet`/`test` + manual smoke kalau perlu

Lihat PR #2 (httputil), PR #4 (whatsapp), PR #12 (jwt) untuk template.

### Commits

- One concept per commit
- Subject line imperative ("Add", "Fix", "Remove" — bukan "Added", "Fixed")
- Body explains **why**, bukan **what** (diff sudah explain what)
- Footer: `Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>` kalau dibantu Claude

---

## Hal-hal yang **tidak boleh** dilakukan

| ❌ Jangan | ✅ Lakukan ini |
|---|---|
| Pakai underscore di nama exported Go | PascalCase |
| Mutate global error variable (`domain.ErrX.Message = ...`) | Return new `*AppError` instance |
| Hardcode credential di code | Baca dari env, document di README |
| Pakai `os.Getenv` di package framework (kecuali debug flag) | Terima via `Config` struct |
| Bikin field `Status_Code` / `Data_Total` baru | Pakai `StatusCode` / `DataTotal` (sudah established) |
| Skip update docs saat ubah API | Update docs dalam commit yang sama |
| Push langsung ke main | Bikin branch + PR (review by user) |
| Override stdlib package name tanpa alias | Kalau package collision (mis. `crypto/rsa`), document alias requirement di package doc |
| Pakai `interface{}` di code baru | `any` (Go 1.18+) |
| Logging via `fmt.Println` / `log.Println` | Pakai `Logger` interface yang di-inject |

---

## Hal-hal yang sering bikin developer / AI baru bingung

1. **Package name collision**: `crypto/rsa` di framework collide dengan stdlib `crypto/rsa`. Consumer harus alias on import. Sudah document di package doc.

2. **Field rename breaking change**: Go tidak support struct field aliases. Field renames di v0.2.0 (`Status_Code` → `StatusCode`) adalah breaking change yang tidak bisa di-shim. Semua service consumer harus update bersamaan.

3. **JSON tag vs Go field name**: response struct field-nya `StatusCode` (Go convention) tapi JSON tag-nya tetap `status_code` (kontrak public ke client). **Jangan ubah JSON tag** — akan memecah client mobile.

4. **Strategy fallback semantic** di `middleware.Auth`: kalau dua strategy `CanHandle == true` dan strategy pertama gagal `Authenticate`, framework **tidak** fallback ke strategy berikutnya. Design supaya hanya satu strategy yang `CanHandle` per request (pakai header berbeda).

5. **APP_DEBUG bug history**: pernah ada bug `strings.ToLower(env) == "TRUE"` (selalu false). Sekarang pakai `strings.EqualFold`. Jangan reintroduce.

6. **`AuthorizeRoles()` tanpa argumen = always reject**. Defensive untuk mencegah accidentally-open route. Untuk endpoint yang harus open ke semua authenticated user, jangan pasang `AuthorizeRoles` sama sekali.

7. **JWT reserved claims**: di `jwt.Claims.Extra`, key seperti `sub`, `exp`, `iat`, `nbf`, `iss`, `aud`, `jti` di-strip otomatis saat Generate. Jangan kaget kalau caller pass `Extra: {"sub": "x"}` dan tidak ke-emit.

8. **Cache backend swap**: `cache.Memory` dan `cache/redis.Cache` punya semantic identik (JSON encoding, `ErrNotFound` on miss, idempotent Delete). Swap impl = 1-line change.

---

## Versioning

Saat ini di `v0.5.0`. Selama `v0.x`, breaking change boleh di minor version (mis. v0.4.0 hapus deprecated alias dari v0.2.0). Setelah `v1.0.0`, breaking hanya di major.

Tag release **setelah** PR di-merge ke main:

```bash
git checkout main && git pull
git tag -a v0.5.0 -m "Add authentication/jwt + AuthorizeRoles"
git push origin v0.5.0
```

---

## Ekspektasi user untuk Claude Code

Berdasarkan interaksi sebelumnya, user mengharapkan:

1. **Tanya sebelum scope-besar**: kalau task melibatkan multiple package atau breaking change, sajikan plan + minta confirm sebelum mulai.
2. **Decisions opinionated**: kalau user bilang "yang terbaik" atau "yang direkomendasikan", ambil keputusan + jelaskan trade-off singkat.
3. **PR per package/feature**: jangan gabung beberapa fitur di satu PR.
4. **Docs sync wajib**: setiap perubahan code = update docs di commit yang sama.
5. **Konsisten Bahasa Indonesia** untuk pesan user-facing (di `errors.New*`) dan dokumentasi. Variable name + technical term tetap English.
6. **Jangan over-engineer**: kalau feature bisa diimplement minimal + extensible nanti, pilih minimal. YAGNI.
7. **Tampilkan PR URL** dengan tag `<pr-created>...</pr-created>` setelah create.

---

## Saat ada hal yang tidak yakin

1. Cek [`docs/packages.md`](docs/packages.md) section package terkait
2. Lihat similar package yang sudah established (mis. menambah integrasi baru → lihat `mailer` atau `notification/fcm` sebagai reference)
3. Tanya user — jangan nebak. User lebih suka clarification 30 detik daripada PR yang harus di-revert.

---

## Maintenance

Update file ini saat:
- Konvensi baru disepakati
- Hard rule berubah
- Pattern arsitektur baru muncul
- Lesson learned dari bug / regression yang penting di-record
