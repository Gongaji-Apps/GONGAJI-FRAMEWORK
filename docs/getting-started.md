# Getting Started

Tutorial end-to-end membangun **microservice REST API GoNgaji-style** dari nol — dari `go mod init` sampai endpoint pertama yang serve data dari Postgres.

Setelah selesai Anda akan mengerti pola dasar yang dipakai di semua 19+ service:

1. Struktur folder service
2. Konfigurasi (env-driven)
3. Wiring framework: validator, binding, response, errors
4. Repository pattern dengan `BaseRepository[T]`
5. Auth middleware
6. Routing & request flow

---

## Daftar Isi

- [0. Prasyarat](#0-prasyarat)
- [1. Init Project](#1-init-project)
- [2. Struktur Folder](#2-struktur-folder)
- [3. Konfigurasi & Bootstrap](#3-konfigurasi--bootstrap)
- [4. Domain & Repository](#4-domain--repository)
- [5. Service Layer](#5-service-layer)
- [6. Handler / Controller](#6-handler--controller)
- [7. Routing](#7-routing)
- [8. Auth Middleware](#8-auth-middleware)
- [9. Request Flow Recap](#9-request-flow-recap)
- [10. Selanjutnya](#10-selanjutnya)

---

## 0. Prasyarat

- Go ≥ 1.25
- PostgreSQL lokal (atau cloud) untuk demo
- Familiar dengan Go basics & Gin

```bash
go version  # harus >= 1.25
```

---

## 1. Init Project

```bash
mkdir api-demo && cd api-demo
go mod init github.com/Gongaji-Apps/api-demo
go get github.com/Gongaji-Apps/GONGAJI-FRAMEWORK@v0.4.0
go get github.com/gin-gonic/gin
go get gorm.io/gorm gorm.io/driver/postgres
```

---

## 2. Struktur Folder

Konvensi yang dipakai di semua service GoNgaji:

```
api-demo/
├── cmd/
│   └── api/
│       └── main.go              # Entry point
├── config/
│   └── config.go                # Env loader
├── domain/
│   └── user/
│       ├── entity.go            # GORM model + DTO
│       ├── repository.go        # Repository wrapper
│       ├── service.go           # Business logic
│       └── handler.go           # Gin handlers
├── routes/
│   └── routes.go                # Route registration
├── go.mod
└── go.sum
```

Domain di-split per **bounded context** (`user/`, `order/`, `payment/`, ...). Setiap bounded context isi-nya:
`entity.go` + `repository.go` + `service.go` + `handler.go`.

---

## 3. Konfigurasi & Bootstrap

### `config/config.go`

```go
package config

import (
    "os"
    "time"

    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/database"
    "gorm.io/gorm"
    gormlogger "gorm.io/gorm/logger"
)

type Config struct {
    AppPort   string
    JWTSecret string
}

func Load() Config {
    return Config{
        AppPort:   getenv("APP_PORT", "8080"),
        JWTSecret: os.Getenv("JWT_SECRET"),
    }
}

func ConnectDB() (*gorm.DB, error) {
    return database.NewPostgres(database.Config{
        Host:         getenv("DB_HOST", "localhost"),
        User:         getenv("DB_USER", "postgres"),
        Password:     os.Getenv("DB_PASSWORD"),
        DBName:       getenv("DB_NAME", "gongaji"),
        Port:         getenv("DB_PORT", "5432"),
        SSLMode:      getenv("DB_SSLMODE", "disable"),
        TimeZone:     getenv("DB_TZ", "Asia/Jakarta"),
        MaxOpenConns: 25,
        MaxIdleConns: 5,
        MaxLifetime:  5 * time.Minute,
        LogLevel:     gormlogger.Warn,
    })
}

func getenv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

### `cmd/api/main.go`

```go
package main

import (
    "log"

    "github.com/Gongaji-Apps/api-demo/config"
    "github.com/Gongaji-Apps/api-demo/domain/user"
    "github.com/Gongaji-Apps/api-demo/routes"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/validator"
    "github.com/gin-gonic/gin"
)

func main() {
    cfg := config.Load()

    db, err := config.ConnectDB()
    if err != nil {
        log.Fatalf("connect DB: %v", err)
    }

    // Daftar semua custom validator framework (URL, UUID, no_space, dll.)
    validator.InitValidator()

    // Auto-migrate schema (development only)
    if err := db.AutoMigrate(&user.User{}); err != nil {
        log.Fatalf("migrate: %v", err)
    }

    // Build dependency tree
    userRepo := user.NewRepository(db)
    userSvc := user.NewService(userRepo)
    userHdl := user.NewHandler(userSvc)

    r := gin.Default()
    routes.Register(r, userHdl)

    if err := r.Run(":" + cfg.AppPort); err != nil {
        log.Fatalf("server: %v", err)
    }
}
```

> **Catatan:** `validator.InitValidator()` harus dipanggil **sekali sebelum routing**. Tanpa ini, custom validator framework (`no_space`, `phone_id`, dll.) tidak ter-register.

---

## 4. Domain & Repository

### `domain/user/entity.go`

```go
package user

import "time"

// User adalah GORM model — direct mapping ke tabel "users".
type User struct {
    UUID      string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"uuid"`
    Name      string    `gorm:"size:100;not null"                              json:"name"`
    Email     string    `gorm:"size:150;uniqueIndex;not null"                  json:"email"`
    Phone     string    `gorm:"size:20"                                        json:"phone"`
    Active    bool      `gorm:"default:true"                                   json:"active"`
    CreatedAt time.Time `                                                      json:"created_at"`
    UpdatedAt time.Time `                                                      json:"updated_at"`
}

// CreateRequest adalah DTO untuk POST /users.
//
// Tag:
//   - binding:  validasi go-playground/validator
//   - normalize: framework akan auto-trim whitespace setelah bind
//   - "nil" option: kalau hasil trim string kosong, set field ke nil
type CreateRequest struct {
    Name  string  `json:"name"  binding:"required,min=2,max=100" normalize:""`
    Email string  `json:"email" binding:"required,email"         normalize:""`
    Phone *string `json:"phone" binding:"omitempty,phone_id"     normalize:"nil"`
}

type UpdateRequest struct {
    Name  *string `json:"name,omitempty"  binding:"omitempty,min=2,max=100" normalize:""`
    Phone *string `json:"phone,omitempty" binding:"omitempty,phone_id"      normalize:"nil"`
}
```

### `domain/user/repository.go`

```go
package user

import (
    "context"

    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/database/repository"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/query"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/result"
    "gorm.io/gorm"
)

// Repository membungkus framework's BaseRepository[User] dan menambahkan
// helper-helper specific user.
type Repository struct {
    repository.BaseRepository[User]
}

func NewRepository(db *gorm.DB) *Repository {
    return &Repository{
        BaseRepository: repository.NewBaseRepository[User](db, "users"),
    }
}

// Find mengembalikan paginated list. Tidak ada custom logic — pure delegation
// ke BaseGetArray dengan query yang sudah disusun caller.
func (r *Repository) Find(ctx context.Context, q query.Query, p query.Pagination) (*result.ArrayResult[User], error) {
    qb := r.BaseBuildQuery(ctx, q, nil)
    return r.BaseGetArray(ctx, qb, p)
}

// FindByUUID mengembalikan user by UUID. notFound=true → return 404, false → return empty.
func (r *Repository) FindByUUID(ctx context.Context, uuid string, notFound bool) (*result.ObjectResult[User], error) {
    qb := r.BaseBuildQuery(ctx, query.Query{
        Wheres: []query.Where{{Query: "uuid = ?", Args: []any{uuid}}},
    }, nil)
    return r.BaseGetObject(ctx, qb, "FIRST", notFound)
}

// EmailExists adalah custom helper untuk uniqueness check.
func (r *Repository) EmailExists(ctx context.Context, email string) (bool, error) {
    qb := r.BaseBuildQuery(ctx, query.Query{
        Wheres: []query.Where{{Query: "email = ?", Args: []any{email}}},
    }, nil)
    return r.BaseExists(ctx, qb)
}
```

### Field reference: `query.Query`

Lihat [packages.md#query](packages.md#query) untuk detail lengkap. Singkatnya:

| Field | Tipe | Fungsi |
|---|---|---|
| `Where` | `any` | Single where clause (struct atau map) |
| `Wheres` | `[]Where` | Multiple where clauses dengan args |
| `Order` | `string` | ORDER BY clause |
| `Havings` | `[]string` | HAVING clauses |
| `UpdateData` | `any` | Data untuk Update operation |

---

## 5. Service Layer

Service layer = business logic. Boleh memanggil multiple repository, validation rule yang tidak bisa di-express di struct tag, kirim notifikasi, dll.

### `domain/user/service.go`

```go
package user

import (
    "context"

    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/query"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/result"
)

type Service struct {
    repo *Repository
}

func NewService(repo *Repository) *Service {
    return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, p query.Pagination) (*result.ArrayResult[User], error) {
    return s.repo.Find(ctx, query.Query{Order: "created_at DESC"}, p)
}

func (s *Service) Get(ctx context.Context, uuid string) (*User, error) {
    res, err := s.repo.FindByUUID(ctx, uuid, true)
    if err != nil {
        return nil, err
    }
    return res.Data, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*User, error) {
    exists, err := s.repo.EmailExists(ctx, req.Email)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, errors.NewConflict("Email sudah terdaftar.")
    }

    u := User{
        Name:  req.Name,
        Email: req.Email,
    }
    if req.Phone != nil {
        u.Phone = *req.Phone
    }

    if err := s.repo.BaseCreate(ctx, nil, u); err != nil {
        return nil, err
    }

    // Re-fetch dengan UUID yang sudah di-generate DB
    return s.Get(ctx, u.UUID)
}
```

> **Pola error handling:** service tidak transform DB-level error. `BaseCreate` sudah otomatis return `errors.NewConflict(...)` kalau ada unique constraint violation, `errors.NewBadRequest(...)` kalau NOT NULL violated, dst. Lihat [errors](packages.md#errors).

---

## 6. Handler / Controller

### `domain/user/handler.go`

```go
package user

import (
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/binding"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/query"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/response"
    "github.com/gin-gonic/gin"
)

type Handler struct {
    svc *Service
}

func NewHandler(svc *Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) List(c *gin.Context) {
    p, err := binding.Query[query.Pagination](c)
    if err != nil {
        response.Error(c, err)
        return
    }

    list, err := h.svc.List(c.Request.Context(), p)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.SuccessArray(c, *list)
}

func (h *Handler) Get(c *gin.Context) {
    type uri struct {
        UUID string `uri:"uuid" binding:"required,uuid"`
    }
    u, err := binding.URI[uri](c)
    if err != nil {
        response.Error(c, err)
        return
    }

    user, err := h.svc.Get(c.Request.Context(), u.UUID)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.Success(c, user)
}

func (h *Handler) Create(c *gin.Context) {
    req, err := binding.JSON[CreateRequest](c)
    if err != nil {
        response.Error(c, err)
        return
    }

    user, err := h.svc.Create(c.Request.Context(), req)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.Created(c, user)
}
```

> **Pola handler:** handler **tipis**. Cuma binding → call service → `response.Success`/`response.Error`. Tidak ada business logic.

---

## 7. Routing

### `routes/routes.go`

```go
package routes

import (
    "github.com/Gongaji-Apps/api-demo/domain/user"
    "github.com/gin-gonic/gin"
)

func Register(r *gin.Engine, userHdl *user.Handler) {
    api := r.Group("/api/v1")
    {
        users := api.Group("/users")
        users.GET("",        userHdl.List)
        users.GET("/:uuid",  userHdl.Get)
        users.POST("",       userHdl.Create)
    }
}
```

---

## 8. Auth Middleware

Framework menyediakan **strategy-based** auth middleware. Anda implementasikan satu atau lebih `AuthStrategy`, framework yang pilih strategy yang cocok per request.

### Pakai JWT strategy bawaan

Framework sudah menyediakan `authentication/jwt` package — jangan tulis JWT strategy sendiri:

```go
// auth/jwt_validator.go
package auth

import (
    "context"

    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/jwt"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/middleware"
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
    "github.com/Gongaji-Apps/api-demo/domain/user"
)

// NewJWTStrategy build strategy dengan Validator yang lakukan DB lookup.
func NewJWTStrategy(secret string, repo *user.Repository) (*jwt.Strategy, error) {
    mgr, err := jwt.New(jwt.Config{
        Secret:     []byte(secret),
        Issuer:     "api-demo",
        DefaultTTL: 24 * time.Hour,
    })
    if err != nil {
        return nil, err
    }

    validator := func(ctx context.Context, c *jwt.Claims, raw string) (*middleware.AuthClaims, error) {
        // App-level checks: DB lookup, single-device, active flag
        res, err := repo.FindByUUID(ctx, c.SubjectUUID, true)
        if err != nil {
            return nil, err
        }
        u := res.Data
        if !u.Active {
            return nil, errors.NewUnauthorized("Akun Anda telah dinonaktifkan.")
        }
        // (Optional) single-device: bandingkan token tersimpan dengan raw
        // if u.Token == nil || *u.Token != raw {
        //     return nil, errors.NewUnauthorized("Token tidak valid.")
        // }

        return &middleware.AuthClaims{
            SubjectUUID: u.UUID,
            Role:        u.Role,
            Extra:       map[string]any{"user_full_name": u.Name},
        }, nil
    }

    return &jwt.Strategy{Manager: mgr, Validator: validator}, nil
}
```

Untuk generate token (mis. di endpoint login):

```go
mgr, _ := jwt.New(jwt.Config{Secret: []byte(cfg.JWTSecret), DefaultTTL: 24 * time.Hour})
token, err := mgr.Generate(jwt.Claims{
    SubjectUUID: user.UUID,
    Extra:       map[string]any{"role": user.Role},
})
// kirim token ke client
```

### Wire up

```go
// routes/routes.go
import (
    "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/authentication/middleware"
)

func Register(r *gin.Engine, cfg config.Config, userHdl *user.Handler, jwtStrategy *jwt.Strategy) {
    authMW := middleware.Auth(jwtStrategy)

    api := r.Group("/api/v1")
    {
        users := api.Group("/users", authMW)
        users.GET("",                                 userHdl.List)
        users.GET("/:uuid",                           userHdl.Get)
        users.POST("",   middleware.RequirePermission("USER_CREATE"), userHdl.Create)
        users.DELETE("/:uuid", middleware.AuthorizeRoles("ADMIN"),    userHdl.Delete)
    }
}
```

Setelah middleware sukses, claims tersedia di gin context:

```go
import "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/contextx"

func (h *Handler) WhoAmI(c *gin.Context) {
    uuid := contextx.GetSubjectUUID(c.Request.Context())
    role := contextx.GetRoleCode(c.Request.Context())
    response.Success(c, gin.H{"uuid": uuid, "role": role})
}
```

> **Catatan:** `contextx.GetSubjectUUID` baca dari `*gin.Context` via `c.Request.Context()`. Pastikan handler pakai `c.Request.Context()` (bukan bikin context baru).

---

## 9. Request Flow Recap

```
HTTP Request
    │
    ▼
Gin Router
    │
    ▼
Auth Middleware (jika protected)            ←  framework
    │  set ctx: subject_uuid, role, permissions
    ▼
RequirePermission Middleware (opsional)     ←  framework
    │
    ▼
Handler
    │
    ├─ binding.JSON / binding.Query / binding.URI    ←  framework
    │     │  bind + validate + normalize
    │     ▼
    │  → response.Error(c, err) jika gagal
    │
    ├─ service.X(ctx, req)
    │     │
    │     ├─ repo.BaseGetObject / BaseCreate / dll.   ←  framework
    │     │     │  GORM under the hood
    │     │     ▼
    │     │  → return *AppError (NotFound/Conflict/etc.)
    │     │
    │     └─ business logic
    │
    └─ response.Success / Created / Error            ←  framework
```

Format response selalu konsisten — service A dan service B kirim shape yang sama, error code yang sama, validation error format yang sama.

---

## 10. Selanjutnya

Anda sudah punya **service basic yang berfungsi**. Untuk fitur lanjutan, lihat:

| Topik | Dokumentasi |
|---|---|
| Reference lengkap setiap package | [packages.md](packages.md) |
| Resep umum (cron, kirim WA, cache, dll.) | [recipes.md](recipes.md) |
| Cara migrate dari pola lama | [recipes.md#migration-dari-pola-lama-19-service-existing](recipes.md#migration-dari-pola-lama-19-service-existing) |

**Saran exercises:**

1. Tambah endpoint `PUT /users/:uuid` (update user)
2. Tambah `PaginatedSearch` dengan filter `?name=&email=` (gunakan `query.Wheres` dengan `LIKE`)
3. Tambah cache di `Get` (gunakan [cache package](packages.md#cache))
4. Tambah scheduler yang setiap 1 jam kirim email digest (gunakan [scheduler](packages.md#scheduler) + [mailer](packages.md#mailer))
5. Tambah endpoint `POST /users/:uuid/notify` yang kirim FCM push (gunakan [notification/fcm](packages.md#notificationfcm))
