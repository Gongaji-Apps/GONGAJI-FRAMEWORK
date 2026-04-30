# GONGAJI-FRAMEWORK

Internal Go framework untuk membangun REST API service di GoNgaji. Mengonsolidasikan pola yang berulang di 19+ service: response shape, error handling, validation, repository pattern, HTTP client, integrasi pihak ketiga (WhatsApp, FCM, Cloud Tasks, SMTP), caching, dan scheduler.

> **Tujuan:** menulis service baru ≤ 1 hari, bukan 1 minggu.

---

## Daftar Isi

- [Instalasi](#instalasi)
- [Quick Start (5 menit)](#quick-start-5-menit)
- [Daftar Package](#daftar-package)
- [Dokumentasi Lengkap](#dokumentasi-lengkap)
- [Versioning & Stabilitas](#versioning--stabilitas)

---

## Instalasi

```bash
go get github.com/Gongaji-Apps/GONGAJI-FRAMEWORK@latest
```

Atau pin ke versi spesifik:

```bash
go get github.com/Gongaji-Apps/GONGAJI-FRAMEWORK@v0.4.0
```

Persyaratan:

- Go ≥ **1.25**
- (Opsional, sesuai package yang dipakai) PostgreSQL via GORM, Redis, Google Cloud credentials, Firebase service account, SMTP server.

---

## Quick Start (5 menit)

Endpoint REST yang menerima JSON body, validasi, dan kembalikan response standar.

```go
package main

import (
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/binding"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/response"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/validator"
    "github.com/gin-gonic/gin"
)

type CreateUserRequest struct {
    Name  string `json:"name"  binding:"required" normalize:""`
    Email string `json:"email" binding:"required,email"`
}

func main() {
    validator.InitValidator()

    r := gin.Default()
    r.POST("/users", func(c *gin.Context) {
        req, err := binding.JSON[CreateUserRequest](c)
        if err != nil {
            response.Error(c, err)
            return
        }

        if req.Email == "blocked@example.com" {
            response.Error(c, errors.NewBadRequest("Email diblokir."))
            return
        }

        response.Created(c, gin.H{
            "name":  req.Name,
            "email": req.Email,
        })
    })

    r.Run(":8080")
}
```

Yang sudah Anda dapatkan tanpa nulis sendiri:

- **Validation** — `binding:"required,email"` divalidasi otomatis dengan pesan error rapih
- **Normalization** — `normalize:""` tag akan trim whitespace di string field
- **Standard response** — semua endpoint kirim shape `{status, status_code, message, data, ...}` yang sama
- **Error handling** — `errors.NewBadRequest`, `errors.NewNotFound`, dll. otomatis di-map ke HTTP status yang tepat lewat `response.Error`

Example response sukses:

```json
{
  "status_code": 201,
  "status": true,
  "message": "Data berhasil dibuat.",
  "data": { "name": "Budi", "email": "budi@x.com" }
}
```

Example response error:

```json
{
  "status_code": 400,
  "status": false,
  "message": "Validation Error!",
  "meta": { "Email": "Email harus berupa alamat email yang valid" }
}
```

---

## Daftar Package

| Package | Fungsi | Dependency Eksternal |
|---|---|---|
| **Web API layer** | | |
| [`response`](docs/packages.md#response) | Response builder standar (Success, Created, Error, dll.) | gin |
| [`errors`](docs/packages.md#errors) | `AppError` + constructor untuk semua HTTP error | — |
| [`binding`](docs/packages.md#binding) | Bind + validate + normalize request payload | gin |
| [`validator`](docs/packages.md#validator) | Custom validator + i18n error messages | go-playground/validator |
| [`result`](docs/packages.md#result) | `ObjectResult[T]` & `ArrayResult[T]` untuk response generic | — |
| [`pagination`](docs/packages.md#pagination) | Metadata pagination | — |
| [`query`](docs/packages.md#query) | `Query` struct untuk repository | — |
| [`normalizer`](docs/packages.md#normalizer) | Trim whitespace via struct tag | — |
| **Database** | | |
| [`database`](docs/packages.md#database) | Postgres connection helper | gorm + postgres driver |
| [`database/repository`](docs/packages.md#databaserepository) | Generic `BaseRepository[T]` (CRUD + query + upsert + transaction) | gorm |
| **Auth** | | |
| [`authentication/middleware`](docs/packages.md#authenticationmiddleware) | Strategy-pattern auth middleware | gin |
| [`authentication/utils`](docs/packages.md#authenticationutils) | Bearer extractor, API key generator | — |
| [`contextx`](docs/packages.md#contextx) | Typed context keys (request ID, subject UUID, role, dll.) | — |
| **Integrasi pihak ketiga** | | |
| [`httputil`](docs/packages.md#httputil) | Reusable HTTP client + retry + timeout | — (stdlib only) |
| [`messaging/whatsapp`](docs/packages.md#messagingwhatsapp) | WhatsApp gateway client (V1+V2 fallback) | httputil |
| [`mailer`](docs/packages.md#mailer) | SMTP email + HTML body + attachment | — (stdlib only) |
| [`notification/fcm`](docs/packages.md#notificationfcm) | Firebase Cloud Messaging push | firebase-admin |
| [`cloudtask`](docs/packages.md#cloudtask) | Google Cloud Tasks v2 wrapper | cloud.google.com/go/cloudtasks |
| [`storage`](docs/packages.md#storage) + [`storage/gcs`](docs/packages.md#storagegcs) | File upload/download abstraction + GCS impl | cloud.google.com/go/storage |
| **Infrastructure** | | |
| [`cache`](docs/packages.md#cache) + [`cache/redis`](docs/packages.md#cacheredis) | Cache interface + in-memory + Redis impl | redis/go-redis |
| [`scheduler`](docs/packages.md#scheduler) | Cron-style background job runner | robfig/cron |
| [`logger`](docs/packages.md#logger) | Structured JSON logger dengan request ID | logrus |
| [`tracer`](docs/packages.md#tracer) | OpenTelemetry tracer setup | otel |
| **Utilities** | | |
| [`converter`](docs/packages.md#converter) | String/byte/base64/time/int conversion | — |
| [`formatter`](docs/packages.md#formatter) | Slug, currency, phone, string utilities | golang.org/x/text |
| [`random`](docs/packages.md#random) | Random string, OTP, UUID | google/uuid |
| [`timeutil`](docs/packages.md#timeutil) | Time helpers (SameDay, Ramadhan check) | — |
| [`crypto/rsa`](docs/packages.md#cryptorsa) | RSA key gen + OAEP/PKCS1v15 encrypt/decrypt | — (stdlib only) |

---

## Dokumentasi Lengkap

| Dokumen | Untuk |
|---|---|
| **[docs/getting-started.md](docs/getting-started.md)** | Tutorial end-to-end membangun service dari nol |
| **[docs/packages.md](docs/packages.md)** | Referensi lengkap setiap package: types, functions, examples, gotchas |
| **[docs/recipes.md](docs/recipes.md)** | Resep-resep umum: cron job, kirim WhatsApp, cache hit, dll. |
| **[CHANGELOG.md](CHANGELOG.md)** | Riwayat versi dan breaking changes |

---

## Versioning & Stabilitas

Framework mengikuti **semantic versioning** dengan tag `vMAJOR.MINOR.PATCH`.

| Version | Highlights |
|---|---|
| `v0.4.0` | Hapus deprecated snake_case aliases dari v0.2.0 |
| `v0.3.0` | Tambah `httputil`, `crypto/rsa`, `messaging/whatsapp`, `mailer`, `cloudtask`, `notification/fcm`, `scheduler`, `cache` |
| `v0.2.0` | Naming convention overhaul (PascalCase + camelCase). Field renames bersifat breaking |
| `v0.1.0` | Foundation: response, errors, validator, repository, binding, normalizer |

Karena masih `v0.x`, **breaking change boleh terjadi di minor version**. Setelah `v1.0.0`, breaking change hanya di major version.

**Strategi pin version untuk service consumer:**

```go
// go.mod
require github.com/Gongaji-Apps/GONGAJI-FRAMEWORK v0.4.0
```

Jangan pakai `@latest` di production — selalu pin ke version eksplisit untuk reproducibility.

---

## Lisensi

MIT — lihat [LICENSE](LICENSE).
