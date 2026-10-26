// Package compression provides response-compression middleware for gin.
//
// Gzip memampatkan respons teks (JSON, HTML, dll.) bila klien mengirim
// "Accept-Encoding: gzip". Isi respons setelah didekompresi identik byte demi byte;
// yang berubah hanya header transport (Content-Encoding, Vary, Content-Length).
// Klien Dart/Flutter dan browser mendekompresi otomatis.
//
// Contoh:
//
//	router := gin.New()
//	router.Use(gin.Recovery())
//	router.Use(compression.Gzip(compression.Config{}))
package compression

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultMinLength: respons di bawah ini tidak dimampatkan — header gzip dan
	// CPU-nya tak sebanding dengan hematnya.
	DefaultMinLength = 1024

	// DefaultLevel: BestSpeed memberi sebagian besar penghematan ukuran dengan CPU
	// paling murah (CPU Cloud Run ditagih per detik).
	DefaultLevel = gzip.BestSpeed
)

type Config struct {
	MinLength     int      // 0 = DefaultMinLength
	Level         int      // 0 = DefaultLevel; gzip.BestSpeed..gzip.BestCompression
	ExcludedPaths []string // prefix path yang tak pernah dimampatkan (mis. download biner, SSE)
}

// Gzip mengembalikan middleware kompresi. Pasang sesudah gin.Recovery() dan
// sebelum route, supaya semua handler tercakup.
func Gzip(cfg Config) gin.HandlerFunc {
	minLength := cfg.MinLength
	if minLength <= 0 {
		minLength = DefaultMinLength
	}

	level := cfg.Level
	if level == 0 || level < gzip.BestSpeed || level > gzip.BestCompression {
		level = DefaultLevel
	}

	pool := &sync.Pool{
		New: func() any {
			w, _ := gzip.NewWriterLevel(io.Discard, level)
			return w
		},
	}

	return func(c *gin.Context) {
		if !acceptsGzip(c.Request, cfg.ExcludedPaths) {
			c.Next()
			return
		}

		w := &gzipWriter{ResponseWriter: c.Writer, pool: pool, minLength: minLength}
		c.Writer = w
		defer w.finish()

		c.Next()
	}
}

func acceptsGzip(r *http.Request, excluded []string) bool {
	if r.Method == http.MethodHead {
		return false
	}
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
		return false
	}
	if r.Header.Get("Range") != "" {
		return false
	}
	for _, p := range excluded {
		if strings.HasPrefix(r.URL.Path, p) {
			return false
		}
	}
	return true
}

func compressibleType(contentType string) bool {
	ct := strings.ToLower(contentType)
	switch {
	case strings.HasPrefix(ct, "application/json"),
		strings.HasPrefix(ct, "text/"),
		strings.HasPrefix(ct, "application/javascript"),
		strings.HasPrefix(ct, "application/xml"),
		strings.HasPrefix(ct, "image/svg+xml"):
		return true
	}
	return false
}

// gzipWriter menahan body sampai MinLength byte sebelum memutuskan dimampatkan
// atau tidak. Selama ditahan, header belum terkirim sehingga Content-Encoding dan
// Content-Length masih bisa diubah — dan middleware lain (mis. cache) masih bisa
// memasang header.
type gzipWriter struct {
	gin.ResponseWriter

	pool      *sync.Pool
	minLength int

	buf      []byte
	decided  bool
	compress bool
	gz       *gzip.Writer
}

func (w *gzipWriter) decide(wantCompress bool) {
	w.decided = true

	h := w.Header()
	status := w.Status()
	eligible := status >= 200 &&
		status != http.StatusNoContent &&
		status != http.StatusNotModified &&
		h.Get("Content-Encoding") == "" &&
		compressibleType(h.Get("Content-Type"))

	if eligible {
		h.Add("Vary", "Accept-Encoding")
	}

	if !wantCompress || !eligible {
		return
	}

	w.compress = true
	h.Del("Content-Length") // panjang asli tak lagi benar setelah dimampatkan
	h.Set("Content-Encoding", "gzip")

	w.gz = w.pool.Get().(*gzip.Writer)
	w.gz.Reset(w.ResponseWriter)
}

// flushBuffer menulis body yang ditahan ke tujuan akhir.
func (w *gzipWriter) flushBuffer() error {
	if len(w.buf) == 0 {
		return nil
	}

	data := w.buf
	w.buf = nil

	if w.compress {
		_, err := w.gz.Write(data)
		return err
	}

	_, err := w.ResponseWriter.Write(data)
	return err
}

func (w *gzipWriter) Write(data []byte) (int, error) {
	if w.decided {
		if w.compress {
			return w.gz.Write(data)
		}
		return w.ResponseWriter.Write(data)
	}

	w.buf = append(w.buf, data...)

	if len(w.buf) >= w.minLength {
		w.decide(true)
		if err := w.flushBuffer(); err != nil {
			return 0, err
		}
	}

	return len(data), nil
}

func (w *gzipWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Written: header dianggap belum terkirim selama body masih ditahan.
func (w *gzipWriter) Written() bool {
	return w.decided && w.ResponseWriter.Written()
}

// WriteHeaderNow dipanggil untuk respons tanpa body (mis. AbortWithStatus).
func (w *gzipWriter) WriteHeaderNow() {
	if !w.decided {
		w.decide(false)
		_ = w.flushBuffer()
	}
	w.ResponseWriter.WriteHeaderNow()
}

// Flush dipakai respons streaming: putuskan saat itu juga (tanpa kompresi bila
// belum mencapai MinLength) supaya data benar-benar sampai ke klien.
func (w *gzipWriter) Flush() {
	if !w.decided {
		w.decide(false)
		_ = w.flushBuffer()
	}
	if w.compress {
		_ = w.gz.Flush()
	}
	w.ResponseWriter.Flush()
}

// finish dipanggil sesudah semua handler selesai.
func (w *gzipWriter) finish() {
	if !w.decided {
		// Tak pernah mencapai MinLength → kirim apa adanya.
		w.decide(false)
		_ = w.flushBuffer()
	}

	if w.compress {
		_ = w.gz.Close()
		w.gz.Reset(io.Discard)
		w.pool.Put(w.gz)
		w.gz = nil
	}
}
