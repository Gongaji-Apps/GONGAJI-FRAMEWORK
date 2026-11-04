package database

import (
	"strings"
	"testing"
	"time"
)

func TestPoolSettings(t *testing.T) {
	cases := []struct {
		name       string
		cfg        Config
		open, idle int
		lifetime   time.Duration
	}{
		{"nol → default aman, bukan tanpa batas", Config{}, DefaultMaxOpenConns, DefaultMaxIdleConns, DefaultMaxLifetime},
		{"nilai eksplisit dipakai apa adanya", Config{MaxOpenConns: 20, MaxIdleConns: 8, MaxLifetime: time.Minute}, 20, 8, time.Minute},
		{"idle dipangkas ke open", Config{MaxOpenConns: 3, MaxIdleConns: 10}, 3, 3, DefaultMaxLifetime},
		{"idle default dipangkas ke open kecil", Config{MaxOpenConns: 2}, 2, 2, DefaultMaxLifetime},
		{"negatif diperlakukan seperti nol", Config{MaxOpenConns: -1, MaxIdleConns: -1, MaxLifetime: -time.Second}, DefaultMaxOpenConns, DefaultMaxIdleConns, DefaultMaxLifetime},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			open, idle, lifetime := poolSettings(c.cfg)
			if open != c.open || idle != c.idle || lifetime != c.lifetime {
				t.Errorf("dapat open=%d idle=%d lifetime=%s, mau open=%d idle=%d lifetime=%s",
					open, idle, lifetime, c.open, c.idle, c.lifetime)
			}
		})
	}
}

func TestGormConfig(t *testing.T) {
	lama := gormConfig(Config{})
	if lama.SkipDefaultTransaction || lama.PrepareStmt {
		t.Errorf("default harus perilaku lama: SkipDefaultTransaction=%v PrepareStmt=%v",
			lama.SkipDefaultTransaction, lama.PrepareStmt)
	}
	if lama.Logger == nil {
		t.Error("Logger nil")
	}

	baru := gormConfig(Config{SkipDefaultTransaction: true, PrepareStmt: true})
	if !baru.SkipDefaultTransaction || !baru.PrepareStmt {
		t.Errorf("opt-in tidak terpasang: SkipDefaultTransaction=%v PrepareStmt=%v",
			baru.SkipDefaultTransaction, baru.PrepareStmt)
	}
}

func TestBuildDSN_ConnectTimeout(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		mau  string
	}{
		{"nol → default 10 detik", Config{}, "connect_timeout=10"},
		{"eksplisit", Config{ConnectTimeout: 5 * time.Second}, "connect_timeout=5"},
		{"pecahan dibulatkan ke atas, tak pernah 0 (0 = tanpa batas di libpq)", Config{ConnectTimeout: 300 * time.Millisecond}, "connect_timeout=1"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if dsn := buildDSN(c.cfg); !strings.Contains(dsn, c.mau) {
				t.Errorf("DSN tidak memuat %q: %s", c.mau, dsn)
			}
		})
	}

	if dsn := buildDSN(Config{StatementTimeout: 15 * time.Second}); !strings.Contains(dsn, "options='-c statement_timeout=15000'") {
		t.Errorf("statement_timeout hilang: %s", dsn)
	}
}
