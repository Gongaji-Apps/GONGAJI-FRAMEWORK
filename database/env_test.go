package database

import (
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

func setDBEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvHost, "db.internal")
	t.Setenv(EnvPort, "5432")
	t.Setenv(EnvUser, "svc")
	t.Setenv(EnvPassword, "rahasia")
	t.Setenv(EnvDBName, "gongaji")
	t.Setenv(EnvSSLMode, "require")
	t.Setenv(EnvTimeZone, "Asia/Jakarta")
}

func TestConfigFromEnv_MembacaVariabel(t *testing.T) {
	setDBEnv(t)

	cfg := ConfigFromEnv(EnvOptions{})

	if cfg.Host != "db.internal" || cfg.Port != "5432" || cfg.User != "svc" || cfg.Password != "rahasia" ||
		cfg.DBName != "gongaji" || cfg.SSLMode != "require" || cfg.TimeZone != "Asia/Jakarta" {
		t.Errorf("tujuan koneksi tidak terbaca dari env: %+v", cfg)
	}

	if cfg.MaxLifetime != DefaultMaxLifetime {
		t.Errorf("MaxLifetime = %s, mau %s", cfg.MaxLifetime, DefaultMaxLifetime)
	}
}

func TestConfigFromEnv_Profil(t *testing.T) {
	setDBEnv(t)

	cases := []struct {
		name       string
		opts       EnvOptions
		open, idle int
		level      logger.LogLevel
		timeout    time.Duration
	}{
		{"non-production: pool kecil, log Info", EnvOptions{}, DefaultMaxOpenConns, DefaultMaxIdleConns, logger.Info, 0},
		{"production: pool besar, log Warn", EnvOptions{Production: true}, ProductionMaxOpenConns, ProductionMaxIdleConns, logger.Warn, 0},
		{"QuietLog: tetap Warn di non-production", EnvOptions{QuietLog: true}, DefaultMaxOpenConns, DefaultMaxIdleConns, logger.Warn, 0},
		{"StatementTimeout diteruskan", EnvOptions{Production: true, StatementTimeout: 30 * time.Second}, ProductionMaxOpenConns, ProductionMaxIdleConns, logger.Warn, 30 * time.Second},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := ConfigFromEnv(c.opts)
			if cfg.MaxOpenConns != c.open || cfg.MaxIdleConns != c.idle || cfg.LogLevel != c.level || cfg.StatementTimeout != c.timeout {
				t.Errorf("dapat open=%d idle=%d level=%v timeout=%s, mau open=%d idle=%d level=%v timeout=%s",
					cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.LogLevel, cfg.StatementTimeout, c.open, c.idle, c.level, c.timeout)
			}
		})
	}
}

func TestConfigFromEnv_AnggaranPoolDariEnv(t *testing.T) {
	setDBEnv(t)

	t.Setenv(EnvMaxOpenConns, "4")
	t.Setenv(EnvMaxIdleConns, "2")

	cfg := ConfigFromEnv(EnvOptions{Production: true})
	if cfg.MaxOpenConns != 4 || cfg.MaxIdleConns != 2 {
		t.Errorf("env harus menimpa profil: open=%d idle=%d", cfg.MaxOpenConns, cfg.MaxIdleConns)
	}

	t.Setenv(EnvMaxOpenConns, "abc")
	t.Setenv(EnvMaxIdleConns, "0")

	cfg = ConfigFromEnv(EnvOptions{Production: true})
	if cfg.MaxOpenConns != ProductionMaxOpenConns || cfg.MaxIdleConns != ProductionMaxIdleConns {
		t.Errorf("nilai tidak sah harus diabaikan: open=%d idle=%d", cfg.MaxOpenConns, cfg.MaxIdleConns)
	}
}
