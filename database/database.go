package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Default pool bila Config dibiarkan 0. Sebelumnya 0 berarti TANPA BATAS koneksi
// (perilaku database/sql). Di Cloud Run, total koneksi ke Postgres adalah
// max-instances × MaxOpenConns per service, dan jumlah seluruh service harus tetap
// di bawah max_connections server — tanpa batas, lonjakan trafik bisa menghabiskan
// slot koneksi dan menjatuhkan SEMUA service yang memakai DB yang sama.
const (
	DefaultMaxOpenConns = 10
	DefaultMaxIdleConns = 5
	DefaultMaxLifetime  = 30 * time.Minute
)

type Config struct {
	Host         string
	User         string
	Password     string
	DBName       string
	Port         string
	SSLMode      string
	TimeZone     string
	MaxOpenConns int           // 0 = DefaultMaxOpenConns
	MaxIdleConns int           // 0 = DefaultMaxIdleConns (dipangkas ke MaxOpenConns)
	MaxLifetime  time.Duration // 0 = DefaultMaxLifetime
	LogLevel     logger.LogLevel

	// StatementTimeout (opsional) menyetel server-side statement_timeout per koneksi
	// sebagai pengaman query lambat. 0 = tidak diaktifkan (perilaku lama).
	StatementTimeout time.Duration

	// SkipDefaultTransaction (opsional) mematikan transaksi implisit GORM pada setiap
	// Create/Update/Delete. Tiap operasi tulis jadi hemat 2 round-trip (BEGIN/COMMIT)
	// — terasa bila DB diakses lewat jaringan jauh. Operasi satu statement tetap
	// atomik; yang hilang hanya atomisitas Create/Save BERSAMA asosiasi (GORM menulis
	// beberapa statement) — bungkus kasus itu dengan BaseTransaction. false = perilaku lama.
	SkipDefaultTransaction bool

	// PrepareStmt (opsional) meng-cache prepared statement per koneksi sehingga query
	// berulang tak di-parse ulang oleh server. JANGAN aktifkan bila koneksi lewat
	// PgBouncer mode transaction/statement (prepared statement tak terbawa antar koneksi).
	// false = perilaku lama.
	PrepareStmt bool
}

// gormConfig dan poolSettings dipisah dari NewPostgres sebagai seam test:
// keduanya murni, jadi bisa diuji tanpa server Postgres.
func gormConfig(cfg Config) *gorm.Config {
	return &gorm.Config{
		Logger:                 logger.Default.LogMode(cfg.LogLevel),
		SkipDefaultTransaction: cfg.SkipDefaultTransaction,
		PrepareStmt:            cfg.PrepareStmt,
	}
}

func poolSettings(cfg Config) (maxOpen int, maxIdle int, maxLifetime time.Duration) {
	maxOpen = cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = DefaultMaxOpenConns
	}

	maxIdle = cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = DefaultMaxIdleConns
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}

	maxLifetime = cfg.MaxLifetime
	if maxLifetime <= 0 {
		maxLifetime = DefaultMaxLifetime
	}

	return maxOpen, maxIdle, maxLifetime
}

func NewPostgres(cfg Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.Port,
		cfg.SSLMode,
		cfg.TimeZone,
	)

	if cfg.StatementTimeout > 0 {
		dsn += fmt.Sprintf(" options='-c statement_timeout=%d'", cfg.StatementTimeout.Milliseconds())
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig(cfg))

	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	maxOpen, maxIdle, maxLifetime := poolSettings(cfg)

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(maxLifetime)

	return db, nil
}
