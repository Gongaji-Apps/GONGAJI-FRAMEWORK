// Package env provides small helpers for reading process configuration from
// environment variables at startup.
//
// Di Cloud Run seluruh konfigurasi datang dari env var service (deploy/env.<env>.yaml
// + Secret Manager). Require memastikan variabel wajib sudah terisi saat start,
// supaya salah konfigurasi gagal cepat dengan pesan yang menyebut nama variabelnya —
// bukan container mati tanpa keterangan, atau lebih buruk, tersambung diam-diam ke
// DB yang salah.
//
// Contoh:
//
//	if err := env.Require("SECRET_KEY", "DB_POSTGRES_HOST", "DB_POSTGRES_PASSWORD"); err != nil {
//	    log.Fatalf("Konfigurasi: %v", err)
//	}
//
//	port := env.Get("PORT", "8080")
//	dryRun := env.Bool("NOTIF_DRY_RUN")
package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Require mengembalikan error yang menyebut SEMUA variabel yang kosong (bukan hanya
// yang pertama), supaya satu kali start gagal cukup untuk melengkapi konfigurasi.
// Nilai yang hanya berisi spasi dianggap kosong.
func Require(keys ...string) error {
	var missing []string

	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("env var wajib belum diisi: %s", strings.Join(missing, ", "))
	}

	return nil
}

// Get mengembalikan nilai env var tanpa spasi tepi, atau fallback bila kosong.
func Get(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

// Bool bernilai true hanya bila env var berisi "true" (tanpa memandang huruf besar/kecil).
// Nilai lain, termasuk kosong, "1", atau "yes", dianggap false — eksplisit lebih aman
// untuk saklar seperti NOTIF_DRY_RUN atau CLOUD_TASK_DISABLED.
func Bool(key string) bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "true")
}

// Int membaca env var bilangan bulat; fallback bila kosong atau bukan angka. Dipakai mis.
// EXPIRED_TOKEN (jam) → time.Duration(env.Int("EXPIRED_TOKEN", 24)) * time.Hour.
func Int(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))

	if err != nil {
		return fallback
	}

	return value
}
