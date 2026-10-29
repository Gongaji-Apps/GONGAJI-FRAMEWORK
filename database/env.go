package database

import (
	"os"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Nama env var yang dibaca ConfigFromEnv. SATU set variabel yang sama di semua
// lingkungan; nilainya yang berbeda per lingkungan dan ditentukan saat deploy
// (deploy/env.<env>.yaml + Secret Manager), bukan dipilih oleh kode.
const (
	EnvHost     = "DB_POSTGRES_HOST"
	EnvPort     = "DB_POSTGRES_PORT"
	EnvUser     = "DB_POSTGRES_USER"
	EnvPassword = "DB_POSTGRES_PASSWORD"
	EnvDBName   = "DB_POSTGRES_DATABASE"
	EnvSSLMode  = "DB_POSTGRES_SSLMODE"
	EnvTimeZone = "DB_POSTGRES_TIMEZONE"
)

// Ukuran pool profil production. Lebih besar karena melayani trafik nyata;
// non-production memakai DefaultMaxOpenConns/DefaultMaxIdleConns supaya banyak
// revisi staging/lokal tidak menghabiskan slot koneksi DB bersama.
const (
	ProductionMaxOpenConns = 20
	ProductionMaxIdleConns = 10
)

// EnvOptions mengatur PERILAKU koneksi (pool, log, timeout) — bukan tujuan koneksi,
// yang selalu dibaca dari env var.
type EnvOptions struct {
	// Production: pool ProductionMaxOpenConns/ProductionMaxIdleConns dan log SQL
	// level Warn. false: pool DefaultMaxOpenConns/DefaultMaxIdleConns dan log Info
	// (SQL + parameter tercetak, membantu debugging staging/lokal).
	Production bool

	// QuietLog menahan log di level Warn juga di non-production. Pakai bila parameter
	// query service memuat data sensitif (token sesi, OTP, sandi terenkripsi) yang
	// tidak boleh tercetak ke Cloud Logging.
	QuietLog bool

	// StatementTimeout meneruskan Config.StatementTimeout. 0 = tidak diaktifkan.
	StatementTimeout time.Duration
}

// ConfigFromEnv menyusun Config dari env var DB_POSTGRES_* plus profil opts.
// MaxLifetime selalu DefaultMaxLifetime.
func ConfigFromEnv(opts EnvOptions) Config {
	cfg := Config{
		Host:             os.Getenv(EnvHost),
		Port:             os.Getenv(EnvPort),
		User:             os.Getenv(EnvUser),
		Password:         os.Getenv(EnvPassword),
		DBName:           os.Getenv(EnvDBName),
		SSLMode:          os.Getenv(EnvSSLMode),
		TimeZone:         os.Getenv(EnvTimeZone),
		MaxOpenConns:     DefaultMaxOpenConns,
		MaxIdleConns:     DefaultMaxIdleConns,
		MaxLifetime:      DefaultMaxLifetime,
		LogLevel:         logger.Info,
		StatementTimeout: opts.StatementTimeout,
	}

	if opts.Production {
		cfg.MaxOpenConns = ProductionMaxOpenConns
		cfg.MaxIdleConns = ProductionMaxIdleConns
	}

	if opts.Production || opts.QuietLog {
		cfg.LogLevel = logger.Warn
	}

	return cfg
}

// NewPostgresFromEnv = NewPostgres(ConfigFromEnv(opts)). Panggil env.Require untuk
// variabel wajib lebih dulu supaya pesan kesalahannya menyebut nama variabel, bukan
// galat koneksi yang samar.
func NewPostgresFromEnv(opts EnvOptions) (*gorm.DB, error) {
	return NewPostgres(ConfigFromEnv(opts))
}
