# Packages Reference

Dokumentasi referensi untuk **setiap package** di framework. Tiap section berisi:
- **Tujuan**: apa fungsi package ini
- **Types & Functions**: API publik
- **Examples**: kode siap-pakai
- **Gotchas**: kebiasaan yang sering bikin developer baru bingung

Pakai Ctrl+F untuk cari nama function/type yang Anda butuhkan.

---

## Daftar Isi

**Web API layer:**
- [response](#response) · [errors](#errors) · [binding](#binding) · [validator](#validator) · [result](#result) · [pagination](#pagination) · [query](#query) · [normalizer](#normalizer)

**Database:**
- [database](#database) · [database/repository](#databaserepository)

**Auth:**
- [authentication/middleware](#authenticationmiddleware) · [authentication/utils](#authenticationutils) · [contextx](#contextx)

**Integrasi pihak ketiga:**
- [httputil](#httputil) · [messaging/whatsapp](#messagingwhatsapp) · [mailer](#mailer) · [notification/fcm](#notificationfcm) · [cloudtask](#cloudtask) · [storage](#storage) · [storage/gcs](#storagegcs)

**Infrastructure:**
- [cache](#cache) · [cache/redis](#cacheredis) · [scheduler](#scheduler) · [logger](#logger) · [tracer](#tracer)

**Utilities:**
- [converter](#converter) · [formatter](#formatter) · [random](#random) · [timeutil](#timeutil) · [crypto/rsa](#cryptorsa)

---

## response

**Tujuan**: response builder standar. Semua endpoint kirim shape yang sama: `{status_code, status, message, data, data_total, pagination, meta}`.

### Types

```go
type Response struct {
    StatusCode int              `json:"status_code"`
    Status     bool             `json:"status"`
    Message    string           `json:"message"`
    Data       any              `json:"data,omitempty"`
    DataTotal  *int64           `json:"data_total,omitempty"`
    Pagination *pagination.Meta `json:"pagination,omitempty"`
    Meta       any              `json:"meta,omitempty"`
}
```

JSON tag tetap `snake_case` untuk kompatibilitas dengan client mobile / legacy.

### Functions

| Function | HTTP | Default message |
|---|---|---|
| `Success(c, data)` | 200 | "Permintaan berhasil diproses." |
| `Created(c, data)` | 201 | "Data berhasil dibuat." |
| `Updated(c, data)` | 200 | "Data berhasil diperbarui." |
| `Deleted(c)` | 200 | "Data berhasil dihapus." |
| `NoContent(c)` | 204 | (no body) |
| `SuccessWithMessage(c, msg, data)` | 200 | custom |
| `SuccessWithMeta(c, data, meta)` | 200 | default + custom meta |
| `SuccessWithCache(c, data, etag, maxAge)` | 200 | + Cache-Control + ETag headers |
| `SuccessObject[T](c, result)` | 200 | unwrap `result.ObjectResult[T]` |
| `SuccessArray[T](c, result)` | 200 | unwrap `result.ArrayResult[T]` |
| `SuccessArrayWithCache[T](c, result, etag, maxAge)` | 200 | array + cache headers |
| `Error(c, err)` | dari error | dari error |
| `Send(c, code, status, msg, data, total, page, meta)` | custom | low-level |

### Examples

```go
// Sukses sederhana
response.Success(c, gin.H{"id": 1})

// Created with data
response.Created(c, user)

// Array dengan pagination
res, _ := svc.List(ctx, p)
response.SuccessArray(c, *res)

// Cache 60 detik
response.SuccessWithCache(c, data, "v1-abc", 60*time.Second)

// Error
response.Error(c, errors.NewNotFound("User tidak ditemukan."))
```

### Gotchas

- `response.Error(c, err)` otomatis baca `*errors.AppError` dan map ke HTTP status. **Selalu return `*errors.AppError`** dari service/repository, jangan raw `error`.
- `SuccessWithCache` set header `Cache-Control: public, stale-while-revalidate=N`. Untuk private (per-user) cache, pakai `SuccessWithMeta` lalu set header sendiri.

---

## errors

**Tujuan**: domain-aware error type yang otomatis di-map ke HTTP status oleh `response.Error`.

### Types

```go
type Code string

const (
    BadRequest          Code = "BAD_REQUEST"            // 400
    Unauthorized        Code = "UNAUTHORIZED"           // 401
    PaymentRequired     Code = "PAYMENT_REQUIRED"       // 402
    Forbidden           Code = "FORBIDDEN"              // 403
    NotFound            Code = "NOT_FOUND"              // 404
    Conflict            Code = "CONFLICT"               // 409
    InternalServerError Code = "INTERNAL_SERVER_ERROR"  // 500
    ServiceUnavailable  Code = "SERVICE_UNAVAILABLE"    // 503
)

type AppError struct {
    Code    Code
    Message string
    Meta    any  // optional, e.g. validation error details
}
```

### Functions

| Constructor | HTTP |
|---|---|
| `NewBadRequest(message)` | 400 |
| `NewBadRequestValidation(message, meta)` | 400 + meta payload |
| `NewUnauthorized(message)` | 401 |
| `NewPaymentRequired(message)` | 402 |
| `NewForbidden(message)` | 403 |
| `NewNotFound(message)` | 404 |
| `NewConflict(message)` | 409 |
| `NewInternalServerError(message)` | 500 |
| `NewServiceUnavailable(message)` | 503 |
| `NewValidationError(message)` | 400 alias untuk `NewBadRequest` |
| `Wrap(code, message, meta)` | low-level |

### DB error normalization

```go
err := errors.NormalizeDBError(gormErr, "users")
```

Otomatis map:
- Duplicate key violation → `NewConflict("[Duplicate]")`
- Foreign key violation → `NewConflict("[Foreign Key]")`
- NOT NULL violation → `NewBadRequest("[Not Null]")`
- Check constraint → `NewBadRequest("[Check Constraint]")`
- Lainnya → `NewInternalServerError(...)`

`BaseRepository[T]` sudah otomatis pakai `NormalizeDBError` di Create/Update/Delete — Anda tidak perlu panggil manual.

### Examples

```go
// Service layer
if exists, _ := repo.EmailExists(ctx, req.Email); exists {
    return errors.NewConflict("Email sudah terdaftar.")
}

// Validation error dengan detail
return errors.NewBadRequestValidation("Validation Error!", map[string]string{
    "Email": "Format email tidak valid",
})

// Wrap external error
if err := externalAPI.Call(); err != nil {
    return errors.NewServiceUnavailable("Layanan eksternal tidak tersedia.")
}
```

### Gotchas

- **Pesan untuk user**, bukan log. `Message` di `AppError` akan muncul di response JSON ke client. Jangan tulis `"failed to query users table: ..."` — tulis `"Tidak dapat memuat daftar user."`
- Untuk debugging, log raw error sebelum return `AppError`.

---

## binding

**Tujuan**: bind + validate + normalize request payload dalam satu pemanggilan generic.

### Functions

| Function | Source |
|---|---|
| `JSON[T any](c) (T, error)` | request body (Content-Type: application/json) |
| `Query[T any](c) (T, error)` | URL query string `?foo=bar` |
| `URI[T any](c) (T, error)` | path parameter `/users/:uuid` |
| `Form[T any](c) (T, error)` | form-encoded / multipart |

Semua function:
1. Bind via Gin's `ShouldBind*`
2. Translate validation error ke `errors.NewBadRequestValidation(meta)` dengan i18n message (Accept-Language: `id` atau `en`)
3. Auto-call `normalizer.NormalizeStruct(&payload)`

### Examples

```go
// JSON body
type CreateUserRequest struct {
    Name  string `json:"name"  binding:"required" normalize:""`
    Email string `json:"email" binding:"required,email"`
}
req, err := binding.JSON[CreateUserRequest](c)
if err != nil {
    response.Error(c, err)
    return
}

// Query parameter (pagination + filters)
type ListQuery struct {
    Limit      int    `form:"limit"      binding:"omitempty,min=1,max=100"`
    Pagination int    `form:"pagination" binding:"omitempty,min=1"`
    Search     string `form:"search"     normalize:""`
}
q, err := binding.Query[ListQuery](c)

// URI parameter
type UserURI struct {
    UUID string `uri:"uuid" binding:"required,uuid"`
}
u, err := binding.URI[UserURI](c)

// Multipart form
type UploadRequest struct {
    Title string                `form:"title" binding:"required"`
    File  *multipart.FileHeader `form:"file"  binding:"required"`
}
req, err := binding.Form[UploadRequest](c)
```

### Validation error response

```json
{
  "status_code": 400,
  "status": false,
  "message": "Validation Error!",
  "meta": {
    "Email": "Email harus berupa alamat email yang valid",
    "Name":  "Name wajib diisi"
  }
}
```

### Gotchas

- Set header `Accept-Language: id` di client untuk dapat error message bahasa Indonesia. Default English.
- Custom validator (`phone_id`, `no_space`, dll.) hanya aktif setelah `validator.InitValidator()` dipanggil di startup.

---

## validator

**Tujuan**: extend [go-playground/validator](https://github.com/go-playground/validator) dengan custom rules + i18n error messages.

### Setup

```go
// di main.go, sebelum router.Run
validator.InitValidator()
```

### Custom rules built-in

| Tag | Validasi |
|---|---|
| `phone_id` | Format nomor telepon Indonesia (08xxx, +628xxx) |
| `no_space` | String tidak boleh mengandung spasi |
| `image_mime` | Content-type harus image/* |
| `not_empty` | String non-empty setelah TrimSpace |
| `id_string` | String berisi hanya huruf, angka, tanda ` -._`  |

Pakai sebagai biasa di binding tag:

```go
type Req struct {
    Phone string `json:"phone" binding:"required,phone_id"`
}
```

### Helper functions (langsung dipanggil tanpa struct tag)

```go
err := validator.URL("https://example.com")
err := validator.UUID("550e8400-e29b-41d4-a716-446655440000")
err := validator.File(contentType, sizeBytes, maxMB, allowedTypes)
```

Return `errors.NewBadRequest("...")` kalau invalid.

### Mendaftarkan custom validator dari kode service

Kalau service Anda butuh rule yang spesifik domain (misal `npwp_format`), daftarkan via `validator.Register`:

```go
import (
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/validator"
    govalidator "github.com/go-playground/validator/v10"
)

func init() {
    validator.Register(func(v *govalidator.Validate) {
        v.RegisterValidation("npwp_format", validateNPWP)
    })
}

func validateNPWP(fl govalidator.FieldLevel) bool {
    // ... implementasi
    return true
}
```

`validator.InitValidator()` akan menjalankan semua callback yang di-Register.

### Gotchas

- **Lupa panggil `validator.InitValidator()`** = custom rule tidak aktif, validator default tetap jalan tapi tag custom diabaikan.
- Pesan error i18n butuh header `Accept-Language: id`. Tanpa header, default English.

---

## result

**Tujuan**: generic types untuk membungkus return data dari repository/service.

### Types

```go
type ObjectResult[T any] struct {
    Data *T `json:"data"`
}

type ArrayResult[T any] struct {
    Data       []T             `json:"data"`
    DataTotal  int64           `json:"data_total"`
    Pagination pagination.Meta `json:"pagination"`
}
```

### Examples

```go
// Single item — kemungkinan nil kalau tidak ditemukan
res, err := repo.FindByUUID(ctx, uuid, false)
if res.Data == nil {
    // tidak ditemukan, tapi notFoundError=false
}

// Array dengan pagination
res, err := repo.Find(ctx, q, p)
fmt.Println("Total:", res.DataTotal)
fmt.Println("Page:", res.Pagination.Current, "of", res.Pagination.Total)
```

`response.SuccessArray(c, *res)` otomatis serialize jadi:

```json
{
  "status_code": 200,
  "status": true,
  "data": [...],
  "data_total": 42,
  "pagination": {"current": 1, "next": 2, "total": 5, "last": false}
}
```

---

## pagination

**Tujuan**: metadata pagination yang konsisten di semua response.

### Types

```go
type Meta struct {
    Current int  `json:"current"`            // halaman saat ini
    Next    *int `json:"next,omitempty"`     // nil di halaman terakhir
    Total   int  `json:"total"`              // total halaman
    Last    bool `json:"last"`               // true di halaman terakhir
}
```

### Functions

```go
// pagination.New(current, total int) Meta
m := pagination.New(2, 5)
// → {Current:2, Next:&3, Total:5, Last:false}
```

`BaseRepository.BaseGetArray` otomatis bikin `Meta` dari `query.Pagination`. Anda jarang perlu panggil manual.

---

## query

**Tujuan**: struct yang memodelkan query parameters untuk repository.

### Types

```go
type Query struct {
    Select     string         // optional: SELECT clause
    UpdateData any            // payload untuk Update operation
    Havings    []string       // HAVING clauses
    Where      any            // single where (struct atau map)
    Wheres     []Where        // multiple where dengan args
    Order      string         // ORDER BY clause
}

type Where struct {
    Query string  // "name LIKE ?", "id IN (?)", dll.
    Args  []any   // argumen yang menggantikan ?
}

type Pagination struct {
    Limit      int `json:"limit"      form:"limit"`
    Pagination int `json:"pagination" form:"pagination"`  // 1-indexed page number
}
```

### Examples

```go
// Single where via struct
q := query.Query{
    Where: User{Active: true},
    Order: "created_at DESC",
}

// Multiple wheres dengan args
q := query.Query{
    Wheres: []query.Where{
        {Query: "name LIKE ?", Args: []any{"%budi%"}},
        {Query: "created_at > ?", Args: []any{lastWeek}},
        {Query: "status IN (?)", Args: []any{[]string{"active", "pending"}}},
    },
    Order: "name ASC",
}

// Update data
q := query.Query{
    Wheres:     []query.Where{{Query: "uuid = ?", Args: []any{uuid}}},
    UpdateData: map[string]any{"active": false, "updated_at": time.Now()},
}
err := repo.BaseUpdate(ctx, nil, q)

// Pagination — biasanya bind dari query string
type ListReq struct {
    query.Pagination
    Search string `form:"search"`
}
```

### Gotchas

- `Where` (singular) takes a struct/map; `Wheres` (plural) takes raw SQL with `?` placeholders. Pilih sesuai kebutuhan.
- `BaseRepository.applyPagination` normalize `Limit` ke 1–1000 (default 100) dan `Pagination` ke ≥1.

---

## normalizer

**Tujuan**: trim whitespace di string field setelah binding, tanpa boilerplate.

### Tag syntax

```go
type Request struct {
    Name    string  `normalize:""`         // trim
    Phone   *string `normalize:"nil"`      // trim, set nil kalau hasilnya empty
    Address string                          // tidak di-normalize
}
```

### Functions

```go
normalizer.NormalizeStruct(&payload)
```

Recursive: jalan ke nested struct + slice + array.

`binding.JSON`, `Query`, `URI`, `Form` otomatis call ini setelah bind sukses. Anda tidak perlu panggil manual kecuali kalau bypass binding.

### Gotchas

- Tag `normalize:""` (string kosong) cukup untuk trim. Tag `normalize:"nil"` khusus pointer string yang ingin nil-ed kalau empty after trim.
- Tidak ada validasi di normalizer — itu tugas validator.

---

## database

**Tujuan**: helper untuk koneksi Postgres via GORM dengan default yang sane.

### Types

```go
type Config struct {
    Host         string
    User         string
    Password     string
    DBName       string
    Port         string
    SSLMode      string  // "disable", "require", "verify-full"
    TimeZone     string  // "Asia/Jakarta", "UTC"
    MaxOpenConns int
    MaxIdleConns int
    MaxLifetime  time.Duration
    LogLevel     gormlogger.LogLevel  // gormlogger.Silent / Error / Warn / Info
}
```

### Functions

```go
db, err := database.NewPostgres(database.Config{
    Host:         "localhost",
    User:         "postgres",
    Password:     "secret",
    DBName:       "gongaji",
    Port:         "5432",
    SSLMode:      "disable",
    TimeZone:     "Asia/Jakarta",
    MaxOpenConns: 25,
    MaxIdleConns: 5,
    MaxLifetime:  5 * time.Minute,
    LogLevel:     gormlogger.Warn,
})
```

### Gotchas

- `MaxOpenConns` terlalu tinggi → connection storm di DB. Aman: 25 untuk service kecil, 50 untuk service besar.
- `MaxLifetime` >0 supaya connection di-recycle (cegah stale connection setelah DB restart).

---

## database/repository

**Tujuan**: generic CRUD repository untuk GORM model. Ganti boilerplate `func GetXByY()` per entity dengan `BaseRepository[T]` yang punya semua method dasar.

### Types

```go
type BaseRepository[T any] struct {
    DB        *gorm.DB
    TableName string
    Debug     bool  // dari env APP_DEBUG=true
}

func NewBaseRepository[T any](db *gorm.DB, table string) BaseRepository[T]
```

### Methods

#### Read

| Method | Returns |
|---|---|
| `BaseBuildQuery(ctx, q, custom)` | `*gorm.DB` (chainable) |
| `BaseBuildQueryFrom(ctx, base, q, custom)` | sama tapi pakai base `*gorm.DB` (untuk preload callback) |
| `BaseGetArray(ctx, qb, p)` | `*result.ArrayResult[T]` |
| `BaseGetObject(ctx, qb, mode, notFound)` | `*result.ObjectResult[T]` (mode: `"FIRST"` atau `"LAST"`) |
| `BaseExists(ctx, qb)` | `(bool, error)` |
| `BaseCount(ctx, qb)` | `(int64, error)` |

#### Write

| Method | Fungsi |
|---|---|
| `BaseCreate(ctx, tx, data)` | INSERT single row |
| `BaseCreateBatch(ctx, tx, []data)` | INSERT many |
| `BaseUpsert(ctx, tx, data, conflictCols, updateCols)` | INSERT ... ON CONFLICT UPDATE |
| `BaseUpdate(ctx, tx, q)` | UPDATE WHERE clause (pakai `q.UpdateData`) |
| `BaseDelete(ctx, tx, q)` | DELETE WHERE clause |
| `BaseTransaction(ctx, fn)` | wrap operation dalam DB transaction |

`tx *gorm.DB` boleh `nil` → otomatis pakai `r.DB`. Pass `tx` non-nil saat di dalam transaction.

### Examples

#### Wrap BaseRepository dengan custom helper

```go
type UserRepository struct {
    repository.BaseRepository[User]
}

func NewUserRepository(db *gorm.DB) *UserRepository {
    return &UserRepository{
        BaseRepository: repository.NewBaseRepository[User](db, "users"),
    }
}

func (r *UserRepository) FindActive(ctx context.Context, p query.Pagination) (*result.ArrayResult[User], error) {
    qb := r.BaseBuildQuery(ctx, query.Query{
        Wheres: []query.Where{{Query: "active = ?", Args: []any{true}}},
        Order:  "created_at DESC",
    }, nil)
    return r.BaseGetArray(ctx, qb, p)
}
```

#### Custom callback untuk JOIN / Preload

```go
qb := r.BaseBuildQuery(ctx, q, func(db *gorm.DB) *gorm.DB {
    return db.
        Preload("Profile").
        Joins("LEFT JOIN orders ON orders.user_id = users.uuid").
        Group("users.uuid")
})
```

#### Transaction

```go
err := r.BaseTransaction(ctx, func(tx *gorm.DB) error {
    if err := r.BaseCreate(ctx, tx, user); err != nil {
        return err
    }
    if err := profileRepo.BaseCreate(ctx, tx, profile); err != nil {
        return err
    }
    return nil
})
```

#### Upsert (INSERT ... ON CONFLICT UPDATE)

```go
err := r.BaseUpsert(ctx, nil,
    user,
    []string{"email"},                  // conflict columns (UNIQUE)
    []string{"name", "phone"},          // columns yang di-update kalau conflict
)
```

Kalau `updateCols` kosong, semua kolom di-update (`UpdateAll: true`).

### Gotchas

- **Selalu pass `*BaseRepository[T]` ke handler/service via pointer**. Method receiver pakai pointer.
- `BaseGetObject(..., notFoundError=true)` → return `*errors.AppError(NotFound)`. Pakai `false` kalau ingin handle di service layer.
- `BaseBuildQuery` mengaplikasikan `Wheres`, `Havings`, `Order` lalu jalan `custom(qb)`. Custom boleh override apapun.
- `Debug: true` dari env `APP_DEBUG=true` aktifkan GORM SQL logging. Jangan aktifkan di production.

---

## authentication/middleware

**Tujuan**: middleware Gin untuk authentication dengan **strategy pattern** — satu middleware bisa handle multiple auth method (JWT, API key, dll.).

### Types

```go
type AuthClaims struct {
    SubjectUUID     string
    Role            string
    PermissionCodes map[string]bool
    Extra           map[string]any
}

type AuthStrategy interface {
    Name() string
    CanHandle(ctx *gin.Context) bool
    ExtractToken(ctx *gin.Context) (string, error)
    Authenticate(ctx context.Context, rawToken string) (*AuthClaims, error)
}
```

### Functions

| Function | Fungsi |
|---|---|
| `Auth(strategies ...AuthStrategy) gin.HandlerFunc` | Coba setiap strategy yang `CanHandle()`, autentikasi, set context |
| `RequirePermission(code string) gin.HandlerFunc` | Reject 403 kalau tidak punya permission code |

### Implement strategy

```go
type JWTStrategy struct {
    Secret string
}

func (s JWTStrategy) Name() string { return "JWT" }

func (s JWTStrategy) CanHandle(c *gin.Context) bool {
    return strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ")
}

func (s JWTStrategy) ExtractToken(c *gin.Context) (string, error) {
    return authutils.ExtractBearer(c)
}

func (s JWTStrategy) Authenticate(ctx context.Context, raw string) (*middleware.AuthClaims, error) {
    // ... parse + verify JWT
    return &middleware.AuthClaims{
        SubjectUUID: sub,
        Role:        role,
        PermissionCodes: map[string]bool{
            "USER_CREATE": true,
            "USER_DELETE": true,
        },
    }, nil
}
```

### Wire up

```go
authMW := middleware.Auth(
    JWTStrategy{Secret: jwtSecret},
    APIKeyStrategy{Repo: apiKeyRepo},  // fallback kalau tidak ada Authorization header
)

protected := r.Group("/api/v1", authMW)
protected.GET("/users",         userHdl.List)
protected.POST("/users",        middleware.RequirePermission("USER_CREATE"), userHdl.Create)
```

### Reading claims

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/contextx"

func (h *Handler) Me(c *gin.Context) {
    uuid := contextx.GetSubjectUUID(c.Request.Context())
    role := contextx.GetRoleCode(c.Request.Context())
}
```

### Gotchas

- Strategy diuji **berurutan** di `Auth(...)`. Letakkan strategy yang paling spesifik di depan.
- Kalau **tidak ada strategy** yang `CanHandle`, middleware return 401 `"Metode autentikasi tidak dikenali."`
- `RequirePermission` baca dari `c.Get("permission_codes")` (di-set oleh `Auth`). Pasang `Auth` dulu, baru `RequirePermission`.

---

## authentication/utils

**Tujuan**: helper kecil yang sering dipakai di strategy implementation.

### Functions

```go
// ExtractBearer mengembalikan token dari header "Authorization: Bearer <token>".
// Return error kalau header missing atau format invalid.
token, err := authutils.ExtractBearer(c)

// GenerateApiKey membuat API key dengan prefix custom (mis. "ngji_").
// Format: prefix + 256-bit random base64-url-encoded.
key, err := authutils.GenerateApiKey("ngji_")

// HashApiKey mengembalikan SHA-256 hex hash. Simpan ini di DB, bukan key plain.
hashed := authutils.HashApiKey(key)
```

### Pattern: API key auth

```go
// Saat user request key baru:
key, _ := authutils.GenerateApiKey("ngji_")
hashed := authutils.HashApiKey(key)
db.Create(&APIKey{Hash: hashed, OwnerUUID: ...})
// → return `key` ke user (sekali aja, jangan disimpan plain)

// Saat verify:
incoming := c.GetHeader("X-Api-Key")
hash := authutils.HashApiKey(incoming)
var stored APIKey
err := db.Where("hash = ?", hash).First(&stored).Error
```

---

## contextx

**Tujuan**: typed context keys + getter untuk metadata yang flow di seluruh request lifecycle.

### Keys

```go
const (
    RequestIDKey       key = "request_id"
    CorrelationIDKey   key = "correlation_id"
    SubjectUUIDKey     key = "subject_uuid"
    RoleCodeKey        key = "role_code"
    PermissionCodesKey key = "permission_codes"
    AuthTypeKey        key = "auth_type"
)
```

### Setters & Getters

| Setter | Getter |
|---|---|
| `WithRequestID(ctx, id)` | `GetRequestID(ctx) string` |
| `WithCorrelationID(ctx, id)` | `GetCorrelationID(ctx) string` |
| `WithSubjectUUID(ctx, uuid)` | `GetSubjectUUID(ctx) string` |
| `WithRoleCode(ctx, role)` | `GetRoleCode(ctx) string` |
| `WithPermissionCodes(ctx, []string)` | `GetPermissionCodes(ctx) []string` |
| `WithAuthType(ctx, type)` | `GetAuthType(ctx) string` |

### Examples

```go
// Middleware set context
ctx := contextx.WithRequestID(c.Request.Context(), uuid.New().String())
c.Request = c.Request.WithContext(ctx)

// Service layer baca
func (s *Service) Audit(ctx context.Context, action string) {
    log.Info("audit",
        "request_id", contextx.GetRequestID(ctx),
        "actor",      contextx.GetSubjectUUID(ctx),
        "action",     action,
    )
}
```

### Gotchas

- `auth.Auth` middleware set keys via `c.Set(...)` (gin context), **bukan** `context.WithValue(...)`. `contextx.GetSubjectUUID(c.Request.Context())` baca via gin context wrapper, jadi tetap berfungsi.
- Kalau pakai goroutine background, propagate ctx eksplisit — `contextx` value tidak otomatis ikut.

---

## httputil

**Tujuan**: reusable HTTP client dengan retry, timeout, optional logging, dan body encoding/decoding otomatis.

### Types

```go
type Config struct {
    BaseURL    string
    Timeout    time.Duration         // default 60s
    Headers    map[string]string     // default headers
    Retry      *RetryConfig          // nil = defaults (3 attempts)
    Logger     Logger                // optional
    HTTPClient *http.Client          // override (custom transport/TLS)
}

type RetryConfig struct {
    MaxAttempts  int           // default 3
    InitialDelay time.Duration // default 200ms
    MaxDelay     time.Duration // default 5s
    Multiplier   float64       // default 2.5
    JitterFactor float64       // default 0.3 (±30%)
}

type HTTPError struct {
    Method, URL, Status string
    StatusCode          int
    Body                []byte
}
```

### Methods

```go
c := httputil.New(httputil.Config{BaseURL: "https://api.example.com"})

// JSON helpers
err := c.Get(ctx, "/path", &resp)
err := c.Post(ctx, "/path", req, &resp)
err := c.Put(ctx, "/path", req, &resp)
err := c.Patch(ctx, "/path", req, &resp)
err := c.Delete(ctx, "/path")

// Form-encoded
err := c.PostForm(ctx, "/path", url.Values{"name": {"alice"}}, &resp)

// Generic (any HTTP method, any body shape)
err := c.Do(ctx, "POST", "/path", body, &result)

// Cloning helpers (immutable — original unchanged)
authed := c.WithAuth("Bearer", token)
scoped := c.WithHeader("X-Tenant", tenantID)
```

### Body encoding rules

| `body` type | Sent as |
|---|---|
| `nil` | no body |
| `[]byte` | raw |
| `string` | raw |
| `io.Reader` | drained into memory then sent (so retry works) |
| anything else | `json.Marshal(body)` + Content-Type: application/json |

### Response decoding rules

| `result` type | Behavior |
|---|---|
| `nil` | response body discarded |
| `*[]byte` | raw body |
| anything else | `json.Unmarshal(body, result)` |

### Retry behavior

Default retry **on**:
- Network errors (connection refused, timeout, DNS)
- HTTP 408 Request Timeout
- HTTP 429 Too Many Requests (honors `Retry-After` header)
- All HTTP 5xx (honors `Retry-After` for 503)

Default retry **off**:
- 4xx (kecuali 408, 429)
- Context cancel/deadline exceeded

### HTTPError introspection

```go
err := c.Post(ctx, "/users", req, &resp)
if httpErr, ok := httputil.AsHTTPError(err); ok {
    if httpErr.StatusCode == 422 {
        // server-side validation failure
        log.Warn("422", "body", string(httpErr.Body))
    }
}
```

### Gotchas

- `BaseURL` trailing slash dihilangkan otomatis. Pass `path` mulai dengan `/`.
- Kalau `path` adalah full URL (`https://...`), ignore BaseURL.
- `WithAuth("Bearer", token)` build header `Authorization: Bearer <token>`. Untuk format lain (mis. `"ApiKey foo"`), pakai `WithHeader("Authorization", "ApiKey foo")`.

---

## messaging/whatsapp

**Tujuan**: client untuk WhatsApp gateway internal GoNgaji (V1 form-encoded + V2 JSON), dengan multi-session fallback.

### Types

```go
type Config struct {
    BaseURLV1  string         // V1 endpoint (form-encoded)
    SessionsV1 []SessionV1    // V1 session names
    BaseURLV2  string         // V2 endpoint (JSON)
    APIKeyV2   string         // V2 API key
    SessionsV2 []SessionV2    // V2 dev/app code pairs
    Timeout    time.Duration
    Logger     httputil.Logger
}

type SessionV1 string
type SessionV2 struct{ DevCode, AppCode string }
```

### Methods

```go
c := whatsapp.New(whatsapp.Config{
    BaseURLV1:  "http://gateway.local:8082",
    SessionsV1: []whatsapp.SessionV1{"GONGAJI", "GONGAJI_BACKUP"},
    BaseURLV2:  "http://wa-v2.example.com/api/",
    APIKeyV2:   "secret",
    SessionsV2: []whatsapp.SessionV2{{DevCode: "DEV1", AppCode: "APP1"}},
})

// Kirim pesan — coba semua V1 session, lalu semua V2 session
err := c.SendMessage(ctx, "+628123", "Halo dunia")

// OTP — wrap SendMessage dengan template standar
err := c.SendOTP(ctx, "+628123", "Budi", "123456")

// V1-only operations
groupID, inviteCode, err := c.CreateGroup(ctx,
    "Group Subject", "Group Description",
    []string{"+628111"}, []string{"+628222", "+628333"},
)

err := c.AddGroupMember(ctx, groupID, []string{"+628444"}, "member")
```

### Fallback semantic

`SendMessage` mencoba **semua session V1 berurutan**. Setelah semua V1 gagal, mencoba **semua session V2**. Return error kalau semua gagal.

`CreateGroup` dan `AddGroupMember` **hanya V1** (V2 tidak support group ops).

### Examples

```go
// SendOTP integrasi dengan random package
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/random"

otp, _ := random.OTP(6)  // "738291"
if err := wa.SendOTP(ctx, user.Phone, user.Name, otp); err != nil {
    return errors.NewServiceUnavailable("Tidak dapat mengirim OTP.")
}
// simpan otp ke cache 5 menit untuk verifikasi
cache.Set(ctx, "otp:"+user.UUID, otp, 5*time.Minute)
```

### Gotchas

- **Bug yang di-fix di framework**: V1 `group_admin[]` / `group_member[]` array fields. Original kode service lama overwrite slice setiap iterasi (cuma kirim member terakhir). Framework pakai `url.Values.Add` yang properly append.
- Fallback try-all-then-fail bisa lambat kalau gateway down (3 V1 + 3 V2 = 6 attempts × 60s timeout = 6 menit). Kalau OTP, set context deadline pendek (10s) untuk fail fast.

---

## mailer

**Tujuan**: SMTP email client dengan HTML body, attachments, BCC, dan TLS/STARTTLS.

### Types

```go
type Config struct {
    Host     string
    Port     int
    Username string
    Password string
    From     string         // default From address
    UseTLS   bool           // true = port 465 implicit TLS, false = STARTTLS
    Timeout  time.Duration  // default 30s
}

type Message struct {
    From        string
    To, CC, BCC []string
    ReplyTo     string
    Subject     string
    TextBody    string         // plain-text fallback
    HTMLBody    string         // HTML body
    Attachments []Attachment
    Headers     map[string]string
}

type Attachment struct {
    Filename    string
    ContentType string
    Data        []byte
}
```

### Methods

```go
m := mailer.New(mailer.Config{
    Host:     "smtp.gmail.com",
    Port:     587,
    Username: smtpUser,
    Password: smtpPass,
    From:     "GoNgaji <noreply@gongaji.id>",
    UseTLS:   false,  // STARTTLS
})

// Quick HTML
err := m.SendHTML(ctx, "user@x.com", "Welcome", "<h1>Halo!</h1>")

// Full message
err := m.Send(ctx, mailer.Message{
    To:      []string{"user@x.com"},
    CC:      []string{"manager@x.com"},
    BCC:     []string{"audit@x.com"},
    Subject: "Invoice #1234",
    HTMLBody: "<p>See attached invoice.</p>",
    TextBody: "See attached invoice.",
    Attachments: []mailer.Attachment{
        {Filename: "invoice.pdf", ContentType: "application/pdf", Data: pdfBytes},
    },
})
```

### Body composition

| Provided | Result |
|---|---|
| HTML only | `text/html` single-part |
| Text only | `text/plain` single-part |
| Both | `multipart/alternative` (text first, HTML second) |
| + attachments | `multipart/mixed` (body part + each attachment) |

### Gotchas

- **Subject otomatis Q-encoded** untuk non-ASCII (`=?UTF-8?q?...?=`). Aman untuk emoji 🎉, kanji, dll.
- **BCC tidak muncul di header** — hanya di SMTP envelope. To+CC+BCC semua ke `RCPT TO`.
- `UseTLS: true` dipakai port 465 (implicit). `UseTLS: false` di port 587 akan upgrade ke STARTTLS kalau server support.
- Anonymous SMTP: kosongkan `Username`. Auth otomatis di-skip.

---

## notification/fcm

**Tujuan**: Firebase Cloud Messaging push notification untuk single device, multi-device, dan topic.

### Types

```go
type Config struct {
    ProjectID      string
    CredentialJSON []byte // raw service account JSON (alternatif: CredentialFile)
    CredentialFile string
}

type Notification struct {
    Title  string             // headline
    Body   string             // detail
    Module string             // misal "TRANSACTION", "IBADAH"
    Type   string             // misal "ORDER_RECEIVED"
    Route  string             // deep link route untuk client app
    Data   map[string]string  // extra payload (merged after Module/Type/Route)
}

type MulticastResult struct {
    SuccessCount int
    FailureCount int
    Errors       []TokenError
}

type TokenError struct {
    Token string
    Err   error
}
```

### Methods

```go
c, err := fcm.New(ctx, fcm.Config{
    ProjectID:      "my-project",
    CredentialJSON: []byte(os.Getenv("FCM_SERVICE_ACCOUNT")),
})

notif := fcm.Notification{
    Title:  "Pesanan Diterima",
    Body:   "Pesanan #1234 telah diterima.",
    Module: "TRANSACTION",
    Type:   "ORDER_RECEIVED",
    Route:  "/orders/1234",
}

// Single device
msgID, err := c.Send(ctx, deviceToken, notif)

// Multiple devices (chunks otomatis ke 500 per batch)
res, err := c.SendMulticast(ctx, allTokens, notif)
log.Info("FCM", "success", res.SuccessCount, "fail", res.FailureCount)
for _, e := range res.Errors {
    log.Warn("FCM token failed", "token", e.Token, "err", e.Err)
}

// Topic broadcast
msgID, err := c.SendToTopic(ctx, "ramadhan-reminders", notif)
```

### Data payload structure

`Notification.Module`/`Type`/`Route` di-merge ke `data` map dengan key persis itu (`module`, `type`, `route`). `Data` map di-merge terakhir, jadi key yang sama akan override.

Client mobile bisa baca:
```kotlin
val module = remoteMessage.data["module"]   // "TRANSACTION"
val route  = remoteMessage.data["route"]    // "/orders/1234"
```

### Gotchas

- `MaxTokensPerBatch = 500` (FCM limit). `SendMulticast` chunks otomatis. 
- Token invalid? Hapus dari DB. Cek `TokenError.Err` — kalau messaging error code `registration-token-not-registered`, token sudah dead.
- ApplicationDefaultCredentials: kalau `CredentialJSON` & `CredentialFile` keduanya kosong, akan baca dari env `GOOGLE_APPLICATION_CREDENTIALS`.

---

## cloudtask

**Tujuan**: Google Cloud Tasks v2 wrapper untuk delayed/queued HTTP task.

### Types

```go
type Config struct {
    ProjectID  string  // GCP project
    LocationID string  // e.g. "asia-southeast2"
}

type TaskRequest struct {
    QueueID  string             // queue name (short, bukan full path)
    URL      string             // target HTTP endpoint (worker)
    Method   string             // default POST
    Payload  []byte             // request body
    Headers  map[string]string
    Schedule *time.Time         // run-at time; nil = immediately
    Name     string             // optional, untuk dedup; short atau full path
}
```

### Methods

```go
c, err := cloudtask.New(ctx, cloudtask.Config{
    ProjectID:  "gongaji-prod",
    LocationID: "asia-southeast2",
})
defer c.Close()

runAt := time.Now().Add(15 * time.Minute)
task, err := c.Create(ctx, cloudtask.TaskRequest{
    QueueID:  "settle-orders",
    URL:      "https://worker.example.com/jobs/settle",
    Payload:  []byte(`{"order_id":"abc"}`),
    Headers:  map[string]string{"Content-Type": "application/json"},
    Schedule: &runAt,
    Name:     "settle-abc",  // dedup: kalau dipanggil 2x, error
})

// Cancel
err = c.Delete(ctx, task.Name)

// Bantu build queue path manual
queuePath := c.QueuePath("settle-orders")
// → projects/gongaji-prod/locations/asia-southeast2/queues/settle-orders
```

### Use cases

- **Delayed action**: kirim reminder email 24 jam setelah signup
- **Async heavy work**: API endpoint kembalikan 202, worker dipanggil via Cloud Task
- **Rate-limited fanout**: queue limit batasi RPS ke downstream

### Gotchas

- Service worker yang menerima task harus respond 2xx dalam timeout queue, atau Cloud Tasks akan retry (lihat queue config).
- `Schedule` ke depan max 30 hari (limit GCP).
- `Name` dedup harus unique per-queue. Kalau coba bikin task dengan name yang sudah ada, error `ALREADY_EXISTS`.

---

## storage

**Tujuan**: interface abstraksi untuk file storage. Implementasi GCS di subpackage `storage/gcs`.

### Interface

```go
type Storage interface {
    Upload(ctx, reader, path, contentType) (url string, err error)
    Delete(ctx, path) error
    DeleteBatch(ctx, paths) error
    DeleteFolder(ctx, prefix) error
    Copy(ctx, src, dst) error
    Move(ctx, src, dst) error
    SignedURL(path, expire) (string, error)
    Exists(ctx, path) (bool, error)
    GetURL(path) string
}
```

### Helper functions

```go
// Upload dari multipart.FileHeader (gin form upload)
url, err := storage.UploadFromMultipart(ctx, s, fileHeader, "users/avatars/uuid.png")

// Upload dari base64 string (data URI atau plain base64)
url, err := storage.UploadFromBase64(ctx, s, b64Data, "users/avatars/uuid.png")

// Upload dengan validation (mime + size)
url, err := storage.UploadWithValidation(ctx, s, fileHeader,
    "users/avatars/uuid.png",
    "image/",       // hanya image/* yang diterima
    5_000,          // max 5000 KB
)
```

### Gotchas

- `SignedURL(path, expire)` untuk akses sementara — pakai untuk private file.
- `GetURL(path)` return public URL (kalau bucket-nya public). Untuk private bucket, pakai `SignedURL`.

---

## storage/gcs

**Tujuan**: Google Cloud Storage implementasi `storage.Storage`.

### Setup

```go
import (
    "cloud.google.com/go/storage"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/storage/gcs"
    "google.golang.org/api/option"
)

client, _ := storage.NewClient(ctx, option.WithCredentialsFile("sa.json"))
bucket := gcs.New(client, gcs.Config{
    Bucket:  "my-bucket",
    BaseURL: "https://storage.googleapis.com",
})
defer client.Close()
```

`bucket` implements `storage.Storage` — pakai langsung di service.

### Examples

```go
// Upload
url, err := bucket.Upload(ctx, fileReader, "files/abc.pdf", "application/pdf")

// Signed URL 1 jam
url, err := bucket.SignedURL("files/abc.pdf", time.Hour)

// Bulk delete (concurrent dengan errgroup)
err := bucket.DeleteBatch(ctx, []string{"files/a.pdf", "files/b.pdf"})

// Cek exists
ok, err := bucket.Exists(ctx, "files/abc.pdf")
```

### Gotchas

- `BaseURL` biasanya `https://storage.googleapis.com`. Untuk custom domain, sesuaikan.
- `SignedURL` butuh service account dengan IAM permission `iam.serviceAccountTokenCreator` di project.
- `DeleteBatch` pakai `errgroup` — semua goroutine di-cancel kalau ada yang gagal.

---

## cache

**Tujuan**: cache abstraction. In-memory implementation di package ini, Redis di subpackage `cache/redis`.

### Interface

```go
type Cache interface {
    Get(ctx, key, dest) error
    Set(ctx, key, value, ttl) error
    Delete(ctx, key) error
    Exists(ctx, key) (bool, error)
    Flush(ctx) error
}

var ErrNotFound = errors.New("cache: key not found")
```

### In-memory: `cache.Memory`

```go
m := cache.NewMemory()

ctx := context.Background()

// Set + Get
m.Set(ctx, "user:1", user, 5*time.Minute)

var u User
if err := m.Get(ctx, "user:1", &u); err != nil {
    if errors.Is(err, cache.ErrNotFound) {
        // miss
    }
}

// Manual sweep (untuk long-running process)
removed := m.Sweep()  // hapus expired entries
```

`Memory` cocok untuk:
- Test
- Short-lived process
- Non-shared cache di service single-instance

**Bukan untuk** multi-instance / production high-traffic — pakai Redis.

### Behavior

- TTL `0` atau negative = no expiration.
- Get pada key absent atau expired = `ErrNotFound`.
- Delete pada key absent = no-op (no error).

### Gotchas

- Memory backend serialize value sebagai JSON. Type yang tidak JSON-marshalable (channel, function) akan error di Set.
- Multi-instance tidak share Memory cache. Kalau service di-deploy >1 replika, pakai Redis.

---

## cache/redis

**Tujuan**: Redis implementation `cache.Cache`, drop-in replacement untuk Memory.

### Setup

```go
import (
    cacheredis "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/cache/redis"
)

c := cacheredis.New(cacheredis.Config{
    Addr:      "localhost:6379",
    Password:  os.Getenv("REDIS_PASS"),
    DB:        0,
    KeyPrefix: "api-store:",         // semua key ditambah prefix ini
    Timeout:   5 * time.Second,
})
defer c.Close()
```

### Borrowed client (shared across services in same process)

```go
client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})

userCache  := cacheredis.NewFromClient(client, "user:")
orderCache := cacheredis.NewFromClient(client, "order:")
// userCache.Close() = no-op (client owned externally)
```

### Behavior

Sama dengan `Memory`, plus:
- Backed by Redis: persistent across process restart, shared antar replika
- `Flush(ctx)` = `FLUSHDB` (hapus seluruh DB). Pakai dengan hati-hati di production!

### Gotchas

- `KeyPrefix` di-prepend ke semua key. Multi-tenant: setiap tenant pakai prefix berbeda.
- `Flush` adalah `FLUSHDB` — hapus semua key di DB Redis ini, **tidak** dibatasi oleh `KeyPrefix`. Untuk prefix-scoped flush, pakai `KEYS prefix*` + `DEL` manual atau split per DB.
- Get pada key absent = `cache.ErrNotFound`. Sama dengan Memory backend, semua impl share interface error.

---

## scheduler

**Tujuan**: cron-style background job runner dengan context propagation, panic recovery, dan skip-if-still-running.

### Types

```go
type Config struct {
    Location     *time.Location  // default UTC
    Logger       Logger
    WithSeconds  bool   // true = enable 6-field cron
    AllowOverlap bool   // false = skip if previous still running
}

type Job struct {
    Name     string
    Schedule string                                 // cron expression
    Handler  func(ctx context.Context) error
}

type Logger interface {
    Info(msg string, fields ...any)
    Error(msg string, fields ...any)
}
```

### Lifecycle

```go
s := scheduler.New(scheduler.Config{
    Location:    time.FixedZone("WIB", 7*3600),
    WithSeconds: true,
})

// Register sebelum Start
s.Register(scheduler.Job{
    Name:     "settle-orders",
    Schedule: "0 */15 * * * *",  // setiap 15 menit (6-field with seconds)
    Handler:  func(ctx context.Context) error {
        return runSettlement(ctx)
    },
})

// Start non-blocking
ctx, cancel := context.WithCancel(context.Background())
s.Start(ctx)

// Graceful shutdown
shutdown := s.Stop()         // stop scheduling new runs
<-shutdown.Done()             // wait until in-flight jobs finish
cancel()
```

### Cron expression format

| Mode | Format | Contoh |
|---|---|---|
| `WithSeconds: false` (default) | `min hour dom month dow` | `*/15 * * * *` (every 15 min) |
| `WithSeconds: true` | `sec min hour dom month dow` | `0 */15 * * * *` (every 15 min, on second 0) |

Special: `@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly`.

### Behavior

- **Panic recovery**: handler yang panic di-log via `Logger.Error("scheduler: job panic", ...)` dan scheduler tetap berjalan.
- **Skip if still running** (default): kalau handler sebelumnya masih jalan saat tick berikutnya datang, tick di-skip. Override dengan `AllowOverlap: true`.
- **Context propagation**: handler menerima context yang dibatalkan saat `Stop()` atau parent ctx cancel.

### Gotchas

- `Register` setelah `Start` = error.
- `Start` sekali aja — tidak bisa restart setelah `Stop`.
- Sub-minute schedule butuh `WithSeconds: true`. 5-field cron (`*/5 * * * *`) tidak akan accept 6 field.

---

## logger

**Tujuan**: structured JSON logger berbasis logrus dengan request ID + correlation ID auto-injection.

### Setup

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/logger"

lg := logger.New()
// Output: {"level":"info","msg":"...","time":"..."}
```

### Per-request log

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/contextx"

func (s *Service) DoThing(ctx context.Context) {
    s.lg.WithCtx(ctx).WithField("user_id", uuid).Info("doing thing")
    // → {"level":"info","msg":"doing thing","request_id":"...","correlation_id":"...","user_id":"..."}
}
```

`WithCtx(ctx)` otomatis ekstrak `request_id` + `correlation_id` dari context.

### Gotchas

- Logger embed `*logrus.Logger` — semua method logrus tersedia langsung (`Info`, `Error`, `WithField`, dll.).
- Default level `Info`. Set ke `Debug` di development: `lg.SetLevel(logrus.DebugLevel)`.

---

## tracer

**Tujuan**: setup OpenTelemetry tracer (untuk distributed tracing).

### Setup

Tergantung exporter (Jaeger, OTLP, Stackdriver). Lihat source `tracer/tracer.go` untuk konfigurasi spesifik. Pattern umum:

```go
shutdown, err := tracer.Init(ctx, tracer.Config{...})
if err != nil { ... }
defer shutdown(ctx)
```

Selanjutnya pakai `otel.Tracer("...")` standar dari `go.opentelemetry.io/otel`.

---

## converter

**Tujuan**: type conversion helper yang return `(*T, error)` style.

### Functions

```go
// String → *primitive
i, err := converter.StringToInt("42")          // *int
f, err := converter.StringToFloat32("3.14")
f, err := converter.StringToFloat64("3.14")
b, err := converter.StringToBool("true")
t, err := converter.StringToDate("2026-04-30") // format "2006-01-02"

// Generic time
t, err := converter.StringToTime("2026-04-30 14:00:00", "2006-01-02 15:04:05")
str := converter.TimeToString(time.Now(), time.RFC3339)

// Int → string
s := converter.IntToString(42)
s := converter.Int64ToString(int64(42))

// Bytes
n, err := converter.BytesToInt([]byte{...})

// Base64
data, contentType, err := converter.DecodeBase64("data:image/png;base64,iVBOR...")
data, err := converter.Base64ToBytes(b64Str)

// Struct ↔ map
m := converter.StructToMap(struct)  // map[string]any via JSON marshal
```

### Float ↔ Rupiah

```go
// converter.Float64ToRupiah(value float64) string
s := converter.Float64ToRupiah(1234567.89)  // "Rp 1.234.567,89"
```

Untuk integer Rupiah, lihat [formatter](#formatter).

---

## formatter

**Tujuan**: string + currency + phone + slug formatting.

### Functions

```go
// Slug
slug := formatter.GenerateSlug("Halo Dunia! 🎉")  // "halo-dunia"

// Indonesian title case
formatter.TitleCase("halo dunia")  // "Halo Dunia"

// SQL LIKE wrap
formatter.WrapLike("budi")  // "%budi%"

// String utilities
formatter.LimitString("Halo dunia panjang sekali", 10)  // "Halo dunia..."
formatter.JoinToSentence([]string{"A", "B", "C"})        // "A, B, dan C"
formatter.CleanDescription("Halo @#$%^ dunia")           // "Halo dunia"
formatter.EscapeSQL("d'angelo")                          // "d''angelo"

// Currency (integer)
formatter.Rupiah(1234567)  // "1.234.567"

// Phone
formatter.NormalizePhone("0812-3456-7890")  // "081234567890"
formatter.ToInternational("081234567890")    // "+6281234567890"

// Filename
formatter.SafeFilename("foto saya.png")  // "foto_saya.png"
```

### Gotchas

- `GenerateSlug` keep huruf+angka, ganti spasi/dash/underscore jadi single dash, drop sisanya. Untuk emoji/karakter unicode, mereka di-drop.
- `TitleCase` pakai locale Indonesia — beda dengan English title case (yang treat preposisi berbeda).

---

## random

**Tujuan**: cryptographically-secure random string generation.

### Functions

```go
// Custom charset
s, err := random.String(8, random.AlphaNumeric)  // "K3dF8a9z"

// Common patterns
otp, err := random.OTP(6)              // "738291" (numeric only)
code, err := random.Code(8)            // "K3DF8A9Z" (alphanumeric uppercase)
id := random.UUID()                    // "550e8400-e29b-41d4-a716-446655440000"
```

### Charsets built-in

```go
random.Alpha                   // a-zA-Z
random.AlphaUpper              // A-Z
random.AlphaLower              // a-z
random.Numeric                 // 0-9
random.AlphaNumeric            // a-zA-Z0-9
random.AlphaNumericUpper       // A-Z0-9
random.AlphaNumericLower       // a-z0-9
random.Character               // !@#$%^&*()-_=+[]{}|;:,.<>/?
random.All                     // alphanumeric + character
```

### Gotchas

- Pakai `crypto/rand` di belakang — aman untuk OTP, API key, password reset token, dll.
- `random.UUID()` pakai `google/uuid` v4 (random-based).

---

## timeutil

**Tujuan**: time helper khusus domain GoNgaji.

### Functions

```go
// Cek apakah dua waktu di hari yang sama
ok := timeutil.SameDay(t1, t2)

// Cek apakah waktu jatuh di Ramadhan
ok := timeutil.IsRamadhan(time.Now())
```

### Gotchas

- `IsRamadhan` saat ini hardcoded ke tanggal Ramadhan tahun spesifik. Lihat [analisis original](../README.md) — direkomendasikan untuk dibuat configurable.

---

## crypto/rsa

**Tujuan**: RSA key generation, PEM serialization, dan encrypt/decrypt.

### ⚠️ Package name collision

Package ini bernama `rsa` (sama dengan stdlib `crypto/rsa`). **Wajib alias saat import**:

```go
import gongajirsa "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/crypto/rsa"
```

### Key management

```go
// Generate (default 2048-bit)
priv, pub, err := gongajirsa.GenerateKeyPair()

// Custom size (>= 2048)
priv, pub, err := gongajirsa.GenerateKeyPairWithSize(4096)

// PEM serialization
privPEM := gongajirsa.PrivateKeyToString(priv)
pubPEM, _ := gongajirsa.PublicKeyToString(pub)

// PEM parsing (auto-detect PKCS#1 + PKCS#8 + PKIX)
priv2, _ := gongajirsa.StringToPrivateKey(privPEM)
pub2, _ := gongajirsa.StringToPublicKey(pubPEM)
```

### Encrypt/Decrypt — OAEP (recommended)

```go
// New code: pakai OAEP/SHA-256
cipher, _ := gongajirsa.Encrypt(pub, []byte("rahasia"))
plain, _ := gongajirsa.Decrypt(priv, cipher)

// Convenience: PEM string + base64 envelope
b64, _ := gongajirsa.EncryptBase64(pubPEM, "halo")
str, _ := gongajirsa.DecryptBase64(privPEM, b64)
```

### Encrypt/Decrypt — PKCS1v15 (legacy interop)

Hanya pakai untuk decrypt data yang sudah di-encrypt dengan PKCS1v15 (sistem lama):

```go
cipher, _ := gongajirsa.EncryptPKCS1v15(pub, plaintext)
plain, _ := gongajirsa.DecryptPKCS1v15(priv, cipher)
```

### Gotchas

- 2048-bit minimum dipaksa. 1024-bit dianggap broken di 2026.
- `GenerateKeyPair` lambat (~50–200ms) untuk 2048-bit. Generate sekali, simpan PEM, reuse.
- OAEP ciphertext **tidak compatible** dengan PKCS1v15 decrypt. Pilih satu skema dan stick.

---

## Footer

Untuk pertanyaan / contoh use case yang tidak ada di sini, baca:

- [recipes.md](recipes.md) — resep umum lintas-package
- [getting-started.md](getting-started.md) — tutorial wiring framework dari awal
- Source code di repo — semua function punya doc comment
