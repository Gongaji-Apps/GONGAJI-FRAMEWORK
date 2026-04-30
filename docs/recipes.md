# Recipes

Resep-resep konkret untuk task yang sering muncul di service GoNgaji. Tiap resep self-contained — Anda bisa copy-paste lalu adapt.

---

## Daftar Isi

**API & request flow:**
- [Endpoint baru: bind, validate, response](#endpoint-baru-bind-validate-response)
- [Pagination + filter](#pagination--filter)
- [Custom validator](#custom-validator)
- [Permission-protected endpoint](#permission-protected-endpoint)

**Database:**
- [Repository pattern](#repository-pattern)
- [Transaction multi-table](#transaction-multi-table)
- [Soft delete](#soft-delete)
- [Upsert (INSERT ON CONFLICT UPDATE)](#upsert-insert-on-conflict-update)

**Integrasi:**
- [Kirim OTP via WhatsApp](#kirim-otp-via-whatsapp)
- [Kirim email transaksional dengan attachment](#kirim-email-transaksional-dengan-attachment)
- [Push notification ke device](#push-notification-ke-device)
- [HTTP call ke external API](#http-call-ke-external-api)
- [File upload ke GCS](#file-upload-ke-gcs)

**Infrastructure:**
- [Cache result query (cache-aside)](#cache-result-query-cache-aside)
- [Schedule cron job](#schedule-cron-job)
- [Delayed task via Cloud Tasks](#delayed-task-via-cloud-tasks)
- [Rate limit OTP per phone](#rate-limit-otp-per-phone)

**Operasional:**
- [Migration dari pola lama (19 service existing)](#migration-dari-pola-lama-19-service-existing)
- [Graceful shutdown](#graceful-shutdown)
- [Multi-source auth (JWT + API key)](#multi-source-auth-jwt--api-key)

---

## Endpoint baru: bind, validate, response

Pola standar untuk endpoint POST yang menerima JSON body.

```go
type CreateOrderRequest struct {
    UserUUID  string  `json:"user_uuid"  binding:"required,uuid"`
    Total     int64   `json:"total"      binding:"required,gt=0"`
    Note      *string `json:"note"       binding:"omitempty,max=200" normalize:"nil"`
}

func (h *Handler) Create(c *gin.Context) {
    req, err := binding.JSON[CreateOrderRequest](c)
    if err != nil {
        response.Error(c, err)  // validation error sudah di-format
        return
    }

    order, err := h.svc.Create(c.Request.Context(), req)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.Created(c, order)
}
```

---

## Pagination + filter

```go
// Request
type ListOrdersQuery struct {
    query.Pagination
    Status    string `form:"status"     binding:"omitempty,oneof=pending paid cancelled"`
    Search    string `form:"search"     normalize:""`
    DateFrom  string `form:"date_from"  binding:"omitempty,datetime=2006-01-02"`
}

func (h *Handler) List(c *gin.Context) {
    q, err := binding.Query[ListOrdersQuery](c)
    if err != nil {
        response.Error(c, err)
        return
    }

    res, err := h.svc.List(c.Request.Context(), q)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.SuccessArray(c, *res)
}

// Service
func (s *Service) List(ctx context.Context, q ListOrdersQuery) (*result.ArrayResult[Order], error) {
    var wheres []query.Where
    if q.Status != "" {
        wheres = append(wheres, query.Where{Query: "status = ?", Args: []any{q.Status}})
    }
    if q.Search != "" {
        wheres = append(wheres, query.Where{Query: "code ILIKE ?", Args: []any{formatter.WrapLike(q.Search)}})
    }
    if q.DateFrom != "" {
        wheres = append(wheres, query.Where{Query: "created_at >= ?", Args: []any{q.DateFrom}})
    }

    qb := s.repo.BaseBuildQuery(ctx, query.Query{
        Wheres: wheres,
        Order:  "created_at DESC",
    }, nil)
    return s.repo.BaseGetArray(ctx, qb, q.Pagination)
}
```

Response:

```json
{
  "status_code": 200,
  "status": true,
  "data": [{...}, ...],
  "data_total": 142,
  "pagination": {"current": 2, "next": 3, "total": 8, "last": false}
}
```

---

## Custom validator

Tambahkan rule custom — misal `npwp_format` (format NPWP Indonesia: `XX.XXX.XXX.X-XXX.XXX`).

```go
package customvalidator

import (
    "regexp"

    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/validator"
    govalidator "github.com/go-playground/validator/v10"
)

var npwpRgx = regexp.MustCompile(`^\d{2}\.\d{3}\.\d{3}\.\d-\d{3}\.\d{3}$`)

func init() {
    validator.Register(func(v *govalidator.Validate) {
        v.RegisterValidation("npwp_format", func(fl govalidator.FieldLevel) bool {
            return npwpRgx.MatchString(fl.Field().String())
        })
    })
}
```

Pastikan package di-import di `main.go` supaya `init()` jalan:

```go
import _ "github.com/Gongaji-Apps/api-demo/customvalidator"
```

Pakai di struct:

```go
type Req struct {
    NPWP string `json:"npwp" binding:"required,npwp_format"`
}
```

---

## Permission-protected endpoint

```go
import (
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/middleware"
)

api := r.Group("/api/v1", authMW)

api.GET("/users",          userHdl.List)              // any authenticated user
api.POST("/users",
    middleware.RequirePermission("USER_CREATE"),
    userHdl.Create,
)
api.DELETE("/users/:uuid",
    middleware.RequirePermission("USER_DELETE"),
    userHdl.Delete,
)
```

Permission codes di-set oleh strategy (`AuthStrategy.Authenticate`) di `AuthClaims.PermissionCodes`. Lihat [authentication/middleware](packages.md#authenticationmiddleware).

---

## Repository pattern

Wrap `BaseRepository[T]` dengan custom helper untuk readability:

```go
type OrderRepository struct {
    repository.BaseRepository[Order]
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
    return &OrderRepository{
        BaseRepository: repository.NewBaseRepository[Order](db, "orders"),
    }
}

// Custom helpers
func (r *OrderRepository) FindByCode(ctx context.Context, code string, notFound bool) (*result.ObjectResult[Order], error) {
    qb := r.BaseBuildQuery(ctx, query.Query{
        Wheres: []query.Where{{Query: "code = ?", Args: []any{code}}},
    }, nil)
    return r.BaseGetObject(ctx, qb, "FIRST", notFound)
}

func (r *OrderRepository) MarkPaid(ctx context.Context, tx *gorm.DB, code string, paidAt time.Time) error {
    return r.BaseUpdate(ctx, tx, query.Query{
        Wheres: []query.Where{{Query: "code = ?", Args: []any{code}}},
        UpdateData: map[string]any{
            "status":  "paid",
            "paid_at": paidAt,
        },
    })
}
```

---

## Transaction multi-table

Operasi yang harus atomic across multiple tables:

```go
func (s *Service) CheckoutOrder(ctx context.Context, req CheckoutRequest) (*Order, error) {
    var order Order

    err := s.orderRepo.BaseTransaction(ctx, func(tx *gorm.DB) error {
        // 1. Insert order
        order = Order{Code: code, Total: req.Total, Status: "paid"}
        if err := s.orderRepo.BaseCreate(ctx, tx, order); err != nil {
            return err
        }

        // 2. Insert order items
        if err := s.itemRepo.BaseCreateBatch(ctx, tx, req.Items); err != nil {
            return err
        }

        // 3. Decrement stock per item
        for _, it := range req.Items {
            err := s.productRepo.BaseUpdate(ctx, tx, query.Query{
                Wheres: []query.Where{{Query: "uuid = ?", Args: []any{it.ProductUUID}}},
                UpdateData: map[string]any{
                    "stock": gorm.Expr("stock - ?", it.Qty),
                },
            })
            if err != nil {
                return err  // rollback otomatis
            }
        }
        return nil
    })

    if err != nil {
        return nil, err
    }
    return &order, nil
}
```

Semua operasi pakai parameter `tx` yang sama. Kalau handler return error, transaction otomatis rollback. Lihat [BaseTransaction](packages.md#databaserepository).

---

## Soft delete

GORM mendukung soft delete via `gorm.DeletedAt`. Kombinasikan dengan framework:

```go
type Order struct {
    UUID      string         `gorm:"primaryKey"`
    // ...
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BaseDelete sekarang otomatis soft-delete (UPDATE deleted_at = NOW())
err := repo.BaseDelete(ctx, nil, query.Query{
    Wheres: []query.Where{{Query: "uuid = ?", Args: []any{uuid}}},
})

// BaseGetArray, BaseGetObject otomatis exclude soft-deleted records
res, _ := repo.BaseGetArray(ctx, qb, p)

// Untuk include soft-deleted, override via custom callback
qb := repo.BaseBuildQuery(ctx, q, func(db *gorm.DB) *gorm.DB {
    return db.Unscoped()
})
```

---

## Upsert (INSERT ON CONFLICT UPDATE)

Untuk data yang either insert atau update, depending on conflict.

```go
// User dengan email unique:
err := userRepo.BaseUpsert(ctx, nil,
    User{Email: "alice@x.com", Name: "Alice", Phone: "+62812"},
    []string{"email"},                  // conflict columns
    []string{"name", "phone"},          // columns yang di-update on conflict
)
```

Pass `updateCols` empty slice → semua kolom di-update (pakai `UpdateAll: true`).

---

## Kirim OTP via WhatsApp

```go
import (
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/cache"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/messaging/whatsapp"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/random"
)

type AuthService struct {
    wa    *whatsapp.Client
    cache cache.Cache
}

func (s *AuthService) RequestOTP(ctx context.Context, user User) error {
    otp, err := random.OTP(6)
    if err != nil {
        return err
    }

    // Simpan OTP 5 menit untuk verifikasi
    if err := s.cache.Set(ctx, "otp:"+user.UUID, otp, 5*time.Minute); err != nil {
        return err
    }

    // Kirim via WA dengan template standar
    return s.wa.SendOTP(ctx, user.Phone, user.Name, otp)
}

func (s *AuthService) VerifyOTP(ctx context.Context, userUUID, attempted string) error {
    var stored string
    err := s.cache.Get(ctx, "otp:"+userUUID, &stored)
    if errors.Is(err, cache.ErrNotFound) {
        return errors.NewBadRequest("OTP sudah kedaluwarsa, silakan minta ulang.")
    }
    if err != nil {
        return err
    }
    if stored != attempted {
        return errors.NewBadRequest("Kode OTP tidak cocok.")
    }

    // Konsumsi OTP supaya tidak bisa dipakai 2x
    _ = s.cache.Delete(ctx, "otp:"+userUUID)
    return nil
}
```

---

## Kirim email transaksional dengan attachment

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/mailer"

func (s *Service) SendInvoice(ctx context.Context, order Order, pdfBytes []byte) error {
    return s.mailer.Send(ctx, mailer.Message{
        To:      []string{order.User.Email},
        Subject: fmt.Sprintf("Invoice #%s", order.Code),
        HTMLBody: fmt.Sprintf(`
            <h2>Halo %s,</h2>
            <p>Terlampir invoice untuk pesanan <b>%s</b>.</p>
            <p>Total: <b>Rp %s</b></p>
        `, order.User.Name, order.Code, formatter.Rupiah(int(order.Total))),
        TextBody: fmt.Sprintf("Halo %s, terlampir invoice pesanan %s.", order.User.Name, order.Code),
        Attachments: []mailer.Attachment{
            {
                Filename:    fmt.Sprintf("invoice-%s.pdf", order.Code),
                ContentType: "application/pdf",
                Data:        pdfBytes,
            },
        },
    })
}
```

---

## Push notification ke device

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/notification/fcm"

func (s *Service) NotifyOrderStatus(ctx context.Context, order Order) error {
    res, err := s.fcm.SendMulticast(ctx, order.User.DeviceTokens, fcm.Notification{
        Title:  "Status Pesanan Diperbarui",
        Body:   fmt.Sprintf("Pesanan #%s sekarang %s.", order.Code, order.Status),
        Module: "TRANSACTION",
        Type:   "ORDER_STATUS_UPDATE",
        Route:  fmt.Sprintf("/orders/%s", order.UUID),
        Data: map[string]string{
            "order_uuid": order.UUID,
            "status":     order.Status,
        },
    })
    if err != nil {
        return err
    }

    // Bersihkan token yang invalid
    for _, e := range res.Errors {
        if isInvalidToken(e.Err) {
            _ = s.tokenRepo.Delete(ctx, e.Token)
        }
    }
    return nil
}
```

---

## HTTP call ke external API

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/httputil"

type EverProClient struct {
    http *httputil.Client
}

func NewEverProClient(apiKey string) *EverProClient {
    return &EverProClient{
        http: httputil.New(httputil.Config{
            BaseURL: "https://api.everpro.id",
            Timeout: 30 * time.Second,
            Headers: map[string]string{
                "Api-Key":      apiKey,
                "Content-Type": "application/json",
            },
        }),
    }
}

type ShippingRateRequest struct {
    Origin      string `json:"origin"`
    Destination string `json:"destination"`
    Weight      int    `json:"weight"`
}

type ShippingRate struct {
    Service string  `json:"service"`
    Cost    float64 `json:"cost"`
    ETA     string  `json:"eta"`
}

func (c *EverProClient) GetRates(ctx context.Context, req ShippingRateRequest) ([]ShippingRate, error) {
    var resp struct {
        Data []ShippingRate `json:"data"`
    }
    if err := c.http.Post(ctx, "/shipping/rates", req, &resp); err != nil {
        if httpErr, ok := httputil.AsHTTPError(err); ok && httpErr.StatusCode == 400 {
            return nil, errors.NewBadRequest("Origin/destination tidak valid.")
        }
        return nil, errors.NewServiceUnavailable("EverPro tidak dapat dihubungi.")
    }
    return resp.Data, nil
}
```

---

## File upload ke GCS

```go
import (
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/storage"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/random"
)

type UploadAvatarRequest struct {
    File *multipart.FileHeader `form:"file" binding:"required"`
}

func (h *Handler) UploadAvatar(c *gin.Context) {
    req, err := binding.Form[UploadAvatarRequest](c)
    if err != nil {
        response.Error(c, err)
        return
    }

    // Generate path unik
    path := fmt.Sprintf("users/avatars/%s%s", random.UUID(), filepath.Ext(req.File.Filename))

    // Upload dengan validasi mime + size
    url, err := storage.UploadWithValidation(
        c.Request.Context(),
        h.storage,
        *req.File,
        path,
        "image/",   // hanya image/* diterima
        2_000,      // max 2 MB
    )
    if err != nil {
        response.Error(c, err)
        return
    }

    response.Success(c, gin.H{"url": url})
}
```

---

## Cache result query (cache-aside)

Pola "check cache → miss → load DB → write cache":

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/cache"

func (s *Service) GetUser(ctx context.Context, uuid string) (*User, error) {
    key := "user:" + uuid

    // 1. Cek cache
    var u User
    err := s.cache.Get(ctx, key, &u)
    if err == nil {
        return &u, nil  // hit
    }
    if !errors.Is(err, cache.ErrNotFound) {
        // Cache backend error — log tapi jangan gagalkan request
        log.Warn("cache error", "err", err)
    }

    // 2. Load dari DB
    res, err := s.repo.FindByUUID(ctx, uuid, true)
    if err != nil {
        return nil, err
    }

    // 3. Tulis cache (best-effort, jangan gagalkan request kalau cache gagal)
    if writeErr := s.cache.Set(ctx, key, res.Data, 10*time.Minute); writeErr != nil {
        log.Warn("cache write error", "err", writeErr)
    }

    return res.Data, nil
}

// Invalidasi saat update
func (s *Service) UpdateUser(ctx context.Context, uuid string, req UpdateRequest) error {
    if err := s.repo.UpdateByUUID(ctx, uuid, req); err != nil {
        return err
    }
    _ = s.cache.Delete(ctx, "user:"+uuid)
    return nil
}
```

---

## Schedule cron job

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/scheduler"

func RunScheduler(ctx context.Context, lg *logger.Logger, svc *OrderService) error {
    s := scheduler.New(scheduler.Config{
        Location:    time.FixedZone("WIB", 7*3600),
        Logger:      lg,
        WithSeconds: false,  // 5-field cron sudah cukup
    })

    // Setiap 15 menit
    s.Register(scheduler.Job{
        Name:     "settle-pending-orders",
        Schedule: "*/15 * * * *",
        Handler: func(ctx context.Context) error {
            return svc.SettlePending(ctx)
        },
    })

    // Setiap hari jam 7 pagi WIB
    s.Register(scheduler.Job{
        Name:     "daily-digest-email",
        Schedule: "0 7 * * *",
        Handler: func(ctx context.Context) error {
            return svc.SendDailyDigest(ctx)
        },
    })

    if err := s.Start(ctx); err != nil {
        return err
    }

    // Wait sampai ctx cancelled
    <-ctx.Done()

    // Graceful shutdown — wait running jobs finish
    <-s.Stop().Done()
    return nil
}
```

Di `main.go`:

```go
go func() {
    if err := RunScheduler(ctx, lg, orderSvc); err != nil {
        lg.Errorf("scheduler error: %v", err)
    }
}()
```

---

## Delayed task via Cloud Tasks

Worker terpisah yang dipanggil setelah delay (mis. retry payment 1 jam kemudian).

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/cloudtask"

// Producer: jadwalkan task
runAt := time.Now().Add(1 * time.Hour)
task, err := s.tasks.Create(ctx, cloudtask.TaskRequest{
    QueueID:  "retry-payment",
    URL:      "https://worker.gongaji.id/jobs/retry-payment",
    Method:   "POST",
    Payload:  payload,
    Headers:  map[string]string{"Content-Type": "application/json"},
    Schedule: &runAt,
    Name:     "retry-" + order.Code,  // dedup: tidak bisa duplikat
})
```

Worker (separate service / endpoint):

```go
r.POST("/jobs/retry-payment", func(c *gin.Context) {
    // Cloud Tasks send via HTTP, worker process synchronously
    var payload RetryPayload
    if err := c.ShouldBindJSON(&payload); err != nil {
        // 4xx → Cloud Tasks tidak retry
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    if err := s.RetryPayment(c.Request.Context(), payload); err != nil {
        // 5xx → Cloud Tasks akan retry dengan backoff
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.Status(204)
})
```

---

## Rate limit OTP per phone

Cegah abuse: max 1 request OTP per nomor per 60 detik.

```go
func (s *AuthService) RequestOTP(ctx context.Context, phone string) error {
    rateKey := "otp-rate:" + phone

    // Cek apakah ada request baru-baru ini
    exists, err := s.cache.Exists(ctx, rateKey)
    if err == nil && exists {
        return errors.NewBadRequest("Tunggu 60 detik sebelum meminta OTP lagi.")
    }

    // Set rate-limit marker (60s TTL)
    _ = s.cache.Set(ctx, rateKey, true, 60*time.Second)

    // Generate + kirim OTP (lihat resep "Kirim OTP via WhatsApp")
    return s.sendOTP(ctx, phone)
}
```

---

## Migration dari pola lama (19 service existing)

Service yang masih pakai naming snake_case (`Base_Repository`, `Object_Result`, dll.) atau function lama (`Bad_Request_Response`, `Convert_Token`, dll.). Pola migration:

### 1. Update `go.mod`

```bash
go get github.com/Gongaji-Apps/GONGAJI-FRAMEWORK@v0.4.0
go mod tidy
```

### 2. Find-replace bulk renames

PowerShell:

```powershell
$replacements = @{
    '\.Status_Code\b'   = '.StatusCode'
    '\.Data_Total\b'    = '.DataTotal'
    '\.Update_Data\b'   = '.UpdateData'
    '\.Table_Name\b'    = '.TableName'
    'Base_Repository'   = 'BaseRepository'
    'New_Base_Repository' = 'NewBaseRepository'
    'Base_Build_Query_From' = 'BaseBuildQueryFrom'
    'Base_Build_Query'  = 'BaseBuildQuery'
    'Base_Get_Array'    = 'BaseGetArray'
    'Base_Get_Object'   = 'BaseGetObject'
    'Base_Create'       = 'BaseCreate'
    'Base_Update'       = 'BaseUpdate'
    'Base_Delete'       = 'BaseDelete'
    'Object_Result'     = 'ObjectResult'
    'Array_Result'      = 'ArrayResult'
    'Generate_Slug'     = 'GenerateSlug'
    'Init_Validator'    = 'InitValidator'
}

Get-ChildItem -Recurse -Include *.go | ForEach-Object {
    $content = Get-Content $_.FullName -Raw
    foreach ($pattern in $replacements.Keys) {
        $content = [regex]::Replace($content, $pattern, $replacements[$pattern])
    }
    Set-Content -Path $_.FullName -Value $content -NoNewline
}
```

### 3. Replace function lama → framework equivalent

| Pola lama | Replacement |
|---|---|
| `function.Bad_Request_Response(c, msg)` | `response.Error(c, errors.NewBadRequest(msg))` |
| `function.Error_Response(c, err)` | `response.Error(c, err)` |
| `function.Success_Response(c, data)` | `response.Success(c, data)` |
| `function.Convert_Struct_To_Map(s)` | `converter.StructToMap(s)` |
| `function.Generate_Slug(s)` | `formatter.GenerateSlug(s)` |
| `function.Whatsapp.Send_Message_Handler(...)` | `whatsapp.Client.SendMessage(ctx, ...)` |
| `function.Send_Email(...)` | `mailer.Mailer.Send(ctx, ...)` |
| ad-hoc `http.Client` + manual JSON marshal | `httputil.Client.Post(ctx, path, body, &resp)` |

### 4. Verifikasi

```bash
go build ./...
go vet ./...
go test ./...
```

Cari pemakaian deprecated yang terlewat:

```powershell
Get-ChildItem -Recurse -Include *.go | Select-String -Pattern '(Bad_Request_Response|Error_Response|Success_Response|Convert_Struct_To_Map|function\.Send_Message)'
```

Empty output = migration selesai.

### 5. Smoke test

Deploy ke staging, smoke test endpoint utama:
- POST /auth/login (validation + JWT)
- GET /users (pagination)
- POST /orders (transaction multi-table)
- Endpoint yang kirim OTP / email / push (integrasi pihak ketiga)

---

## Graceful shutdown

```go
package main

import (
    "context"
    "errors"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    // ... setup ...

    srv := &http.Server{
        Addr:    ":8080",
        Handler: router,
    }

    // Signal channel
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

    // Run server
    go func() {
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Fatalf("server: %v", err)
        }
    }()

    // Run scheduler
    ctx, cancel := context.WithCancel(context.Background())
    go RunScheduler(ctx, lg, orderSvc)

    // Wait for signal
    <-stop
    log.Println("shutting down...")

    // 1. Stop accepting new HTTP requests, wait existing requests finish (max 30s)
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()
    if err := srv.Shutdown(shutdownCtx); err != nil {
        log.Printf("server shutdown: %v", err)
    }

    // 2. Stop scheduler — cancel ctx so handlers see Done()
    cancel()

    log.Println("bye")
}
```

---

## Multi-source auth (JWT + API key)

```go
// strategies/jwt.go
type JWTStrategy struct{ Secret string }
func (s JWTStrategy) Name() string { return "JWT" }
func (s JWTStrategy) CanHandle(c *gin.Context) bool {
    return strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ")
}
// ... ExtractToken, Authenticate

// strategies/apikey.go
type APIKeyStrategy struct{ Repo *APIKeyRepo }
func (s APIKeyStrategy) Name() string { return "API_KEY" }
func (s APIKeyStrategy) CanHandle(c *gin.Context) bool {
    return c.GetHeader("X-Api-Key") != ""
}
func (s APIKeyStrategy) ExtractToken(c *gin.Context) (string, error) {
    return c.GetHeader("X-Api-Key"), nil
}
func (s APIKeyStrategy) Authenticate(ctx context.Context, raw string) (*middleware.AuthClaims, error) {
    hash := authutils.HashApiKey(raw)
    rec, err := s.Repo.FindByHash(ctx, hash)
    if err != nil {
        return nil, errors.NewUnauthorized("API key tidak valid.")
    }
    return &middleware.AuthClaims{
        SubjectUUID: rec.OwnerUUID,
        Role:        "API_CONSUMER",
        PermissionCodes: rec.Permissions,
    }, nil
}

// main
authMW := middleware.Auth(
    JWTStrategy{Secret: cfg.JWTSecret},
    APIKeyStrategy{Repo: apiKeyRepo},
)
```

`middleware.Auth` akan coba JWT dulu (kalau ada `Authorization: Bearer ...`), kalau tidak match coba API key (kalau ada `X-Api-Key: ...`). Kalau dua-dua tidak match → 401.
