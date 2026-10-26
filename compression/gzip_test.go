package compression

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newRouter(cfg Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Gzip(cfg))

	big := map[string]any{"data": strings.Repeat("assalamualaikum ", 500)}
	small := map[string]any{"ok": true}

	r.GET("/big", func(c *gin.Context) { c.JSON(http.StatusOK, big) })
	r.GET("/small", func(c *gin.Context) { c.JSON(http.StatusOK, small) })
	r.GET("/big-401", func(c *gin.Context) { c.AbortWithStatusJSON(http.StatusUnauthorized, big) })
	r.GET("/no-content", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/abort", func(c *gin.Context) { c.AbortWithStatus(http.StatusForbidden) })
	r.GET("/binary", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/octet-stream", bytes.Repeat([]byte{1}, 5000))
	})
	r.GET("/pre-encoded", func(c *gin.Context) {
		c.Header("Content-Encoding", "br")
		c.Data(http.StatusOK, "application/json", bytes.Repeat([]byte("x"), 5000))
	})
	// Pola service lama: Content-Length dipasang manual lalu WriteHeader + Write.
	r.GET("/manual-length", func(c *gin.Context) {
		body := []byte(fmt.Sprintf(`{"data":%q}`, strings.Repeat("a", 4000)))
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("Content-Length", fmt.Sprintf("%d", len(body)))
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.Write(body)
	})
	r.GET("/excluded/big", func(c *gin.Context) { c.JSON(http.StatusOK, big) })
	return r
}

func do(r *gin.Engine, method, path string, gzipAccepted bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if gzipAccepted {
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func gunzip(t *testing.T, b []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("bukan gzip valid: %v", err)
	}
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gagal dekompresi: %v", err)
	}
	return out
}

func TestGzip_BesarDimampatkanIsiIdentik(t *testing.T) {
	r := newRouter(Config{})

	plain := do(r, http.MethodGet, "/big", false)
	zipped := do(r, http.MethodGet, "/big", true)

	if zipped.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q", zipped.Header().Get("Content-Encoding"))
	}
	if !strings.Contains(zipped.Header().Get("Vary"), "Accept-Encoding") {
		t.Errorf("Vary = %q", zipped.Header().Get("Vary"))
	}
	if zipped.Body.Len() >= plain.Body.Len() {
		t.Errorf("tidak mengecil: %d >= %d", zipped.Body.Len(), plain.Body.Len())
	}
	if got := gunzip(t, zipped.Body.Bytes()); !bytes.Equal(got, plain.Body.Bytes()) {
		t.Error("isi setelah dekompresi berbeda dari respons biasa")
	}
}

func TestGzip_TidakDimampatkan(t *testing.T) {
	r := newRouter(Config{ExcludedPaths: []string{"/excluded"}})

	cases := []struct {
		name, method, path string
		accept             bool
		status             int
	}{
		{"klien tak minta gzip", http.MethodGet, "/big", false, 200},
		{"di bawah MinLength", http.MethodGet, "/small", true, 200},
		{"204 tanpa body", http.MethodGet, "/no-content", true, 204},
		{"abort tanpa body", http.MethodGet, "/abort", true, 403},
		{"tipe biner", http.MethodGet, "/binary", true, 200},
		{"sudah ter-encode", http.MethodGet, "/pre-encoded", true, 200},
		{"path dikecualikan", http.MethodGet, "/excluded/big", true, 200},
		{"HEAD", http.MethodHead, "/big", true, 404},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := do(r, c.method, c.path, c.accept)
			if w.Code != c.status {
				t.Errorf("status = %d, mau %d", w.Code, c.status)
			}
			if ce := w.Header().Get("Content-Encoding"); ce == "gzip" {
				t.Errorf("tidak boleh gzip")
			}
		})
	}

	// Body respons kecil harus utuh walau sempat ditahan di buffer.
	if w := do(r, http.MethodGet, "/small", true); w.Body.String() != `{"ok":true}` {
		t.Errorf("body kecil = %q", w.Body.String())
	}
}

func TestGzip_ErrorBesarTetapStatusBenar(t *testing.T) {
	w := do(newRouter(Config{}), http.MethodGet, "/big-401", true)
	if w.Code != http.StatusUnauthorized || w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("status=%d encoding=%q", w.Code, w.Header().Get("Content-Encoding"))
	}
}

func TestGzip_ContentLengthManualDibuang(t *testing.T) {
	w := do(newRouter(Config{}), http.MethodGet, "/manual-length", true)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("tidak dimampatkan")
	}
	if cl := w.Header().Get("Content-Length"); cl != "" {
		t.Errorf("Content-Length lama masih terkirim: %s", cl)
	}
	if got := gunzip(t, w.Body.Bytes()); !bytes.HasPrefix(got, []byte(`{"data":"aaa`)) {
		t.Errorf("isi rusak: %.20q", got)
	}
}

// Header yang dipasang middleware lain SESUDAH handler mulai menulis (selama body
// masih ditahan) tetap terkirim — pola cache.go di service MUSHAF.
func TestGzip_HeaderMasihBisaDipasangSelamaDitahan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Gzip(Config{}))
	r.GET("/x", func(c *gin.Context) {
		_, _ = c.Writer.Write([]byte(`{"a":`))
		if c.Writer.Written() {
			t.Error("Written() true padahal body masih ditahan")
		}
		c.Header("X-Late", "1")
		_, _ = c.Writer.Write([]byte(`1}`))
	})
	w := do(r, http.MethodGet, "/x", true)
	if w.Header().Get("X-Late") != "1" || w.Body.String() != `{"a":1}` {
		t.Errorf("X-Late=%q body=%q", w.Header().Get("X-Late"), w.Body.String())
	}
}
