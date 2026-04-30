# Changelog

Semua perubahan yang menonjol di GONGAJI-FRAMEWORK didokumentasikan di sini.
Format mengikuti [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

Selama `v0.x`, **breaking change boleh terjadi di minor version**. Setelah `v1.0.0`, breaking change hanya di major version.

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
