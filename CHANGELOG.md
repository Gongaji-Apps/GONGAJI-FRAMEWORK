# Changelog

Semua perubahan yang menonjol di GONGAJI-FRAMEWORK didokumentasikan di sini.
Format mengikuti [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

Selama `v0.x`, **breaking change boleh terjadi di minor version**. Setelah `v1.0.0`, breaking change hanya di major version.

---

## [v0.12.0] — Request generik & binding endpoint daftar (tag git v0.0.63, 30 Okt 2026)

Tahap 3 konsolidasi: `shared.Query_Request` yang disalin tiap service menjadi `query.Request[P]` (alias di service), dan boilerplate handler daftar (bind payload + pagination + `StructToMap` + `ToPreload`, ±20 baris × ±150 handler) menjadi dua panggilan.

### Added

- `query`: `Request[P any]{Pagination, Query, Preload P}`.
- `binding`: `ListQuery[T, P](ctx) (T, query.Request[P], error)`, `FinishList[T Preloadable[P], P](&request, payload) error`, `FinishWhere[P](&request, payload) error`, interface `Preloadable[P]`. Tested (test pertama untuk `binding`).

---

## [v0.11.0] — validator JWT standar (tag git v0.0.62, 29 Okt 2026)

Tahap 2 konsolidasi: 15 varian `internal/middleware/auth.go` (lookup user + single-device + aktif + role terikat partner/foundation + tester + izin) disatukan menjadi satu konstruktor; service hanya menyisakan adaptor pemetaan `user_mstr → jwt.Account`.

### Added

- `authentication/jwt`: `NewAccountStrategy(AccountStrategyConfig)` → `*Strategy` dengan Validator standar; tipe `Account`, `AccountLookup`, `PermissionLoader`; konstanta pesan 401 `MsgSessionEnded`, `MsgAccountMissing`, `MsgOtherDevice`, `MsgInactive`; `DefaultSingleDeviceRoles = {"USER"}`, `DefaultAccountTTL = 24h`. Akun yang tidak ada (nil atau `AppError` NotFound dari Lookup) → **401**, bukan 404 (dulu bervariasi antar service). Tested.
- `env`: `Int(key, fallback)`.

### Changed

- Pesan 401 kini seragam di semua service yang memakai `NewAccountStrategy` (gaya "Sesi Anda berakhir. Silakan masuk kembali."); service pola baru (ARTICLE/OTHER/QR/ZISWAF) sebelumnya memakai "[Unauthorized] Token tidak valid.".

---

## [v0.10.0] — konsolidasi helper yang dulu disalin tiap service (tag git v0.0.61, 29 Okt 2026)

Kode yang identik di 7–15 service (`config/env.go`, `config/database.go`, `internal/shared/helper.go`, `repository/grouped.go`, `internal/middleware/gzip.go`) kini ada satu di framework. Tidak ada perubahan perilaku untuk pemakai API yang sudah ada.

### Added

- **`env`** (package baru): `env.Require(keys...)` — gagal cepat dengan pesan yang menyebut SEMUA variabel yang kosong; `env.Get(key, fallback)`; `env.Bool(key)` (hanya `"true"`). Tested. Pengganti `config.Require` yang disalin di 15 service.
- `database`: `ConfigFromEnv(EnvOptions)` dan `NewPostgresFromEnv(EnvOptions)` membaca `DB_POSTGRES_*` (konstanta `database.Env*`). `EnvOptions{Production, QuietLog, StatementTimeout}` hanya mengatur perilaku: pool `ProductionMaxOpenConns`/`ProductionMaxIdleConns` (20/10) + log Warn di production, `DefaultMax*` (10/5) + log Info di luar itu; `QuietLog` menahan Warn (AUTHENTICATION: token/OTP di parameter query). Tested. Pengganti `config.NewDatabase` yang disalin di 13 service.
- `database/repository`: `BaseGetArrayGrouped(ctx, qb, p)` — `BaseGetArray` untuk query ber-`GROUP BY`/`DISTINCT`; total dihitung dari subquery `SELECT count(*) FROM (<query>) AS sub`, bukan `COUNT` langsung yang menghasilkan satu angka per grup. Pengganti `grouped_get_array` di IBADAH & STORE (dulu VR).
- `query`: `Wheres(conditions ...string) []Where` untuk kondisi tetap tanpa argumen, dan `UUIDOrNil(value) any` (`""` → `nil`, karena kolom uuid Postgres menolak string kosong). Tested (test pertama untuk `query`).
- `contextx`: `Privileged(ctx)` — ctx tanpa role supaya hook AfterFind tidak menyamarkan kolom sensitif saat service perlu nilai asli; deadline/cancel dan identitas subjek tetap terbawa. `WithOverview`/`GetOverview` kini terdokumentasi. Tested (test pertama untuk `contextx`).
- `notification/fcm`: `Notification.HighPriority` — Android `priority: high` (+ data di `AndroidConfig`) dan APNS `apns-priority: 10`, `apns-push-type: alert`, alert Title/Body, suara default, data sebagai custom payload; berlaku di `Send`, `SendToTopic`, `SendMulticast`. `false` = pesan tidak berubah. Dipakai STORE menggantikan klien Firebase buatan sendiri. Tested.

### Removed

- Direktori kosong `auth/` (sisa sebelum `authentication/`; tidak pernah berisi package sejak v0.4.0).

### Migrasi service (opsional, non-breaking)

| Dulu (salinan di service) | Kini |
|---|---|
| `middleware.Gzip()` (`internal/middleware/gzip.go`) | `compression.Gzip(compression.Config{})` |
| `config.Require(...)` | `env.Require(...)` |
| `config.NewDatabase()` | `database.NewPostgresFromEnv(database.EnvOptions{Production: gin.Mode() == gin.ReleaseMode})` |
| `shared.Privileged_Context(ctx)` | `contextx.Privileged(ctx)` |
| `shared.Wheres(...)` / `shared.Uuid_Or_Nil(v)` | `query.Wheres(...)` / `query.UUIDOrNil(v)` |
| `grouped_get_array[T](ctx, db, qb, p, label)` | `r.BaseGetArrayGrouped(ctx, qb, p)` |

---

## [v0.9.0] — hapus `version_status` (tag git v0.0.60, 28 Okt 2026)

### Removed

- `response`: field `version_status` (selalu `"allow"`, sisa gating versi aplikasi lama) dihapus dari envelope `Response`. **Breaking** bagi klien yang membacanya; keputusan 28 Okt 2026: tidak ada klien yang memakainya. Service yang mengisinya manual (AUTHENTICATION `login_response`) harus menghapus isian itu saat bump.

---

## [v0.8.0] — cache untuk menekan biaya egress (tag git v0.0.59, 27 Okt 2026)

### Added

- **`compression`** (package baru): `compression.Gzip(Config)` — middleware gzip opt-in untuk respons JSON/teks dengan ambang `MinLength` (default 1 KB) dan level `BestSpeed`. Isi respons identik setelah dekompresi. Tested.
- `database`: field opsional `Config.SkipDefaultTransaction` dan `Config.PrepareStmt` (default `false` = perilaku lama). Konstanta `DefaultMaxOpenConns`, `DefaultMaxIdleConns`, `DefaultMaxLifetime`. Test pertama untuk `database` (pool & gorm config, tanpa server).
- `storage/gcs`: field opsional `Config.CacheControl`. Bila diisi, dipasang sebagai metadata `Cache-Control` pada setiap `Upload`. Kosong = perilaku lama (default GCS `public, max-age=3600` untuk objek publik). Test pertama untuk `storage/gcs` (hermetik, tanpa jaringan).

- `response`: galat **bukan** `*errors.AppError` yang sampai ke `response.Error` kini dicatat (level `error`, JSON: pesan galat asli, `method`, `path`, `status_code`, `request_id`) dan dititipkan ke `ctx.Errors`. Sebelumnya klien dan log sama-sama hanya melihat "Terjadi kesalahan di server" tanpa jejak penyebab (kasus nyata: kunci GCS yang dinonaktifkan). Pesan ke klien tidak berubah. `response.SetLogger(l)` untuk memakai logger service sendiri / buffer di test. Test pertama untuk jalur ini.

### Fixed

- `authentication/jwt.Parse`: token gaya AUTHENTICATION lama (klaim `user_uuid` + `expired` unix detik, tanpa `exp`) kini **ditolak bila sudah lewat** `expired`. Sebelumnya pustaka hanya memvalidasi `exp`, sehingga token lama tidak pernah kedaluwarsa di semua service framework. `Claims.ExpiresAt` terisi dari `expired` bila `exp` tidak ada. Token lama tanpa kedua klaim tetap diterima (perilaku lama).

### Changed

- `database.NewPostgres`: `MaxOpenConns`/`MaxIdleConns`/`MaxLifetime` bernilai 0 kini memakai default aman (10 / 5 / 30 menit), bukan **tanpa batas**. Service yang mengisi nilai eksplisit tak terpengaruh. Anjuran docs "50 untuk service besar" dicabut: di Cloud Run total koneksi = `max-instances × MaxOpenConns`.
- `storage/gcs`: `DeleteBatch` dibatasi 16 request serentak (sebelumnya tanpa batas). `DeleteFolder` kini paralel (batas sama, sebelumnya satu per satu), mengabaikan objek yang sudah hilang (404), dan mengembalikan galat Delete asli bila ada yang gagal. Diuji dengan server GCS palsu.
- `response.SuccessWithCache` / `SuccessArrayWithCache`: header `Cache-Control` kini `public, max-age=N`, bukan `public, stale-while-revalidate=N`. SWR tanpa `max-age` membuat respons langsung basi sehingga kesegarannya hanya bergantung pada `Expires`. `Expires` & `ETag` tetap dikirim. Belum ada service consumer yang memanggil helper ini (NGAJI sengaja menulis header manual karena masalah ini), jadi tak ada migrasi.

---

## [v0.7.1] — pesan auth manusiawi (tag git v0.0.57)

### Changed

- `authentication/jwt` & `authentication/middleware`: pesan 401/403 tanpa tag `[Unauthorized]`/`[Forbidden]` ("Sesi Anda berakhir. Silakan masuk kembali.", "Anda tidak memiliki akses untuk tindakan ini.").

---

## [v0.7.0] — pesan galat manusiawi (tag git v0.0.56)

### Changed (perilaku pesan — kontrak JSON tetap)

- **`errors.NormalizeDBError`** tak lagi mengembalikan tag mentah `[Duplicate]` / `[Foreign Key]` / `[Not Null]` / `[Check Constraint]` / "[Internal Server Error] … Data <tabel>". Pesan kini kalimat Indonesia: "<Label> yang sama sudah ada…", "<Label> tidak bisa dihapus karena masih dipakai…", "<Kolom> wajib diisi.", dst. `*AppError` yang terbungkus diteruskan apa adanya.
- **`response.Error`** memakai `errors.As` (AppError terbungkus `%w` tak lagi jadi 500 generik) dan memetakan galat constraint Postgres mentah yang lolos dari service. Pesan 500 generik: `response.PesanGalatUmum`.
- **`BaseRepository`** 404/500: "<Label> tidak ditemukan." — nama tabel mentah tak pernah tampil.
- **Validasi**: pesan per medan memakai label manusiawi; `min/max/len/gte/lte` sadar teks (karakter); tag tanpa pesan → "<Label> tidak valid." (bukan "invalid value"); tambah `boolean`, `required_if`, `datetime`, dll. Pesan ringkas "Periksa kembali isian Anda." (ID) / "Please check your input." (EN, `Accept-Language: en…`) menggantikan "Validation Error!". JSON tipe tak cocok → meta medan terkait; JSON rusak → pesan jelas.

### Added

- `errors.RegisterTableLabels`, `errors.RegisterFieldLabels`, `errors.RegisterConstraintMessages`, `errors.TableLabel`, `errors.FieldLabel`, `errors.HumanizeIdentifier`, `errors.IsDBConstraintError`.

### Migrasi

- Klien yang mencocokkan string `"[Duplicate]"` atau `"Validation Error!"` harus beralih ke kode HTTP (409/400) + `meta`.

---

## [v0.6.0] — mailer/surat: surel HTML bermerek lintas service

### Added

- **`mailer/surat/`** — cetakan surel HTML bermerek yang seragam lintas service (stdlib only):
  - `surat.Surat{Judul, Teks, ParagrafHTML, Kode, CTALabel, CTAURL, Catatan}` — konten satu surel; bagian kosong tidak dirender
  - `surat.Surat.HTML(m Merek)` — dokumen surel utuh: kepala wordmark + garis aksen, kartu isi, kotak kode besar (OTP), tombol CTA + tautan mentah cadangan, catatan redup, kaki "surel otomatis". Tabel + CSS inline (klien surel memangkas `<style>`, mengabaikan flex/grid, memblokir JS); semua isi di-escape
  - `surat.Surat.TeksPolos()` — versi text/plain untuk multipart/alternative (pasangan `mailer.Message.TextBody`)
  - `surat.Merek{Nama, Tagline, WarnaPrimer, WarnaPekat, WarnaAksen, BasisURL, TautanBantuan}` — identitas visual sebagai parameter, bukan konstanta: beberapa produk pada satu pengirim berbagi struktur tanpa berbagi warna/nama; package tidak membaca env (DI, konsisten pola framework)
  - `surat.GoNgaji(basisURL)` — merek bawaan Go Ngaji (indigo `#5145b9`/`#3f3596` + emas `#ffc748`, palet web santri)
  - Pemakai pertama: GONGAJI-API-NGAJI (semua notifikasi + surel sertifikat/pengingat batch) dan API-AUTHENTICATION (OTP reset sandi)

---

## [v0.5.0] — JWT package + AuthorizeRoles

### Added

- **`authentication/jwt/`** — JWT token generation + parsing + ready-to-use `AuthStrategy`. Mengganti pola `Auth_Middleware` + `Extract_Token` yang ada di setiap service:
  - `jwt.New(Config)` — Manager (secret, issuer, default TTL)
  - `jwt.Manager.Generate(Claims)` — sign HS256
  - `jwt.Manager.Parse(token)` — verify signature + expiry + nbf + issuer; return `ErrInvalidToken` (wrappable) on failure
  - `jwt.Strategy{Manager, Validator}` — implements `middleware.AuthStrategy`. `Validator` adalah hook untuk app-specific checks (DB lookup, single-device enforcement, active flag)
  - `jwt.StrategyName` field untuk multi-signer (mis. internal vs partner JWT)
  - Reserved claim names di `Claims.Extra` di-skip (cegah caller override `sub`/`exp`/dll. tidak sengaja)
  - Algorithm whitelist HS256 only (cegah `alg: none` attack)

- **`authentication/middleware.AuthorizeRoles(...string)`** — role-based authorization middleware. Match case-insensitive terhadap `role_code` di context. `AuthorizeRoles()` tanpa argumen = always reject (defensive).

### Dependencies

- New direct dep: `github.com/golang-jwt/jwt/v5 v5.3.1`.

---

## [v0.4.0] — Cleanup deprecated aliases

### Removed (BREAKING)

Deprecated snake_case aliases yang ditambahkan di v0.2.0 sudah dihapus. Service consumer harus migrasi ke nama baru sebelum upgrade.

| Lama | Baru | Deprecated sejak |
|---|---|---|
| `result.Object_Result[T]` | `result.ObjectResult[T]` | v0.2.0 |
| `result.Array_Result[T]` | `result.ArrayResult[T]` | v0.2.0 |
| `repository.Base_Repository[T]` | `repository.BaseRepository[T]` | v0.2.0 |
| `repository.New_Base_Repository` | `repository.NewBaseRepository` | v0.2.0 |
| `repository.Base_Build_Query` | `repository.BaseBuildQuery` | v0.2.0 |
| `repository.Base_Build_Query_From` | `repository.BaseBuildQueryFrom` | v0.2.0 |
| `repository.Base_Get_Array` | `repository.BaseGetArray` | v0.2.0 |
| `repository.Base_Get_Object` | `repository.BaseGetObject` | v0.2.0 |
| `repository.Base_Create` | `repository.BaseCreate` | v0.2.0 |
| `repository.Base_Update` | `repository.BaseUpdate` | v0.2.0 |
| `repository.Base_Delete` | `repository.BaseDelete` | v0.2.0 |
| `formatter.Generate_Slug` | `formatter.GenerateSlug` | v0.2.0 |
| `validator.Init_Validator` | `validator.InitValidator` | v0.2.0 |

---

## [v0.3.0] — Phase C package additions

### Added

- **`httputil/`** — Reusable HTTP client dengan exponential backoff retry, timeout, optional logger. Stdlib-only.
- **`crypto/rsa/`** — RSA key generation, PEM serialization (PKCS#1 + PKCS#8 + PKIX), OAEP encryption (recommended) + PKCS1v15 (legacy interop).
- **`messaging/whatsapp/`** — WhatsApp gateway client dengan multi-session fallback (V1 form-encoded + V2 JSON), built on httputil. Termasuk bug-fix: `url.Values.Add` instead of overwrite untuk array fields.
- **`mailer/`** — SMTP email client dengan HTML+text body, attachments, BCC, Q-encoded subject untuk non-ASCII, implicit TLS + STARTTLS. Stdlib-only.
- **`cloudtask/`** — Google Cloud Tasks v2 wrapper untuk HTTP-target task creation/deletion.
- **`notification/fcm/`** — Firebase Cloud Messaging client untuk push notification (single token + multicast + topic).
- **`scheduler/`** — Cron-style job runner dengan context propagation, panic recovery, skip-if-still-running.
- **`cache/`** + **`cache/redis/`** — Cache interface + in-memory implementation + Redis-backed implementation. Backend interchangeable, both serialize as JSON.

### Added (P2 small additions)

- `errors.NewPaymentRequired`, `errors.NewServiceUnavailable`
- `response.Updated`, `response.NoContent`, `response.SuccessWithMessage`, `response.SuccessWithMeta`, `response.SuccessWithCache`, `response.SuccessArrayWithCache`
- `repository.BaseExists`, `BaseCount`, `BaseCreateBatch`, `BaseUpsert`, `BaseTransaction`
- `random.OTP`, `random.Code`, `random.UUID`
- `formatter.LimitString`, `JoinToSentence`, `CleanDescription`, `EscapeSQL`
- `storage.Storage.Exists`, `GetURL` (interface methods); `UploadFromBase64`, `UploadWithValidation`
- `converter.IntToString`, `Int64ToString`, `TimeToString`, `StringToTime`
- `validator.URL`, `validator.UUID`, `validator.File`
- `contextx.SubjectUUIDKey`, `RoleCodeKey`, `PermissionCodesKey`, `AuthTypeKey` + getters/setters

### Changed

- `binding.JSON`, `binding.Query`, `binding.URI` sekarang otomatis menjalankan `normalizer.NormalizeStruct` setelah bind sukses (sebelumnya hanya `Form`).

---

## [v0.2.0] — Naming convention overhaul (BREAKING)

### Changed (BREAKING)

Pelanggaran Go convention (snake_case di exported names) di-rename. Type & function aliases ditambahkan untuk backward compat (akan dihapus di v0.4.0).

**Type aliases (backward-compatible):**
- `Object_Result[T]` → `ObjectResult[T]`
- `Array_Result[T]` → `ArrayResult[T]`
- `Base_Repository[T]` → `BaseRepository[T]`

**Function/method renames:**
- `New_Base_Repository` → `NewBaseRepository`
- `Base_Build_Query` → `BaseBuildQuery`
- `Base_Build_Query_From` → `BaseBuildQueryFrom`
- `Base_Get_Array` → `BaseGetArray`
- `Base_Get_Object` → `BaseGetObject`
- `Base_Create` → `BaseCreate`
- `Base_Update` → `BaseUpdate`
- `Base_Delete` → `BaseDelete`
- `Generate_Slug` → `GenerateSlug`
- `Init_Validator` → `InitValidator`

**Struct field renames (BREAKING — Go tidak support struct field aliases):**
- `Response.Status_Code` → `Response.StatusCode`
- `Response.Data_Total` → `Response.DataTotal`
- `Query.Update_Data` → `Query.UpdateData`
- `BaseRepository.Table_Name` → `BaseRepository.TableName`
- `ArrayResult.Data_Total` → `ArrayResult.DataTotal`

JSON tag tetap `snake_case` di response (kontrak public ke client tidak berubah).

### Fixed

- `APP_DEBUG` env compare bug — sebelumnya pakai `strings.ToLower(...) == "TRUE"` yang selalu false. Sekarang pakai `strings.EqualFold`.
- `errors.Code` constants di-normalisasi ke `UPPER_SNAKE_CASE` (sebelumnya campur antara PascalCase dan UPPER_SNAKE_CASE).
- `response.httpStatus()` mapping lengkap untuk `PaymentRequired`, `ServiceUnavailable`, `InternalServerError`.
- `storage/gcs.DeleteBatch` resource leak — sebelumnya goroutine lain tetap berjalan setelah error pertama. Sekarang pakai `errgroup` yang properly cancel semua.

---

## [v0.1.0] — Initial release

### Added

Foundation packages:
- `response` — standard response builder
- `errors` — `AppError` + constructors (BadRequest, NotFound, dll.)
- `validator` — gin validator setup + i18n messages (Indonesia + English)
- `binding` — JSON/Query/URI/Form payload binders
- `normalizer` — struct field normalization via `normalize` tag
- `database/repository.Base_Repository[T]` — generic repository pattern
- `query` + `pagination` + `result` — query helpers
- `formatter`, `converter`, `random`, `timeutil` — utilities
- `storage` + `storage/gcs` — file upload abstraction
- `contextx`, `logger`, `tracer` — observability primitives
- `authentication/middleware` — pluggable auth strategy

---

[v0.5.0]: https://github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/releases/tag/v0.5.0
[v0.4.0]: https://github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/releases/tag/v0.4.0
[v0.3.0]: https://github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/releases/tag/v0.3.0
[v0.2.0]: https://github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/releases/tag/v0.2.0
[v0.1.0]: https://github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/releases/tag/v0.1.0
