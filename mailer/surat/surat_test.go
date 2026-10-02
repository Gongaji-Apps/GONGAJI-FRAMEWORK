package surat

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func merekUji() Merek { return GoNgaji("https://learning.gongaji.id/") }

// Isi surel datang dari data (nama pengguna, judul pengumuman admin) — cetakan
// harus meng-escape-nya, bukan mempercayainya.
func TestHTMLMengEscapeIsi(t *testing.T) {
	s := Surat{Judul: `Halo <b>&`, Teks: "baris <script>satu</script>\n\nparagraf & dua"}
	h := s.HTML(merekUji())
	if strings.Contains(h, "<script>") {
		t.Fatalf("isi tidak di-escape: %q", h)
	}
	for _, mau := range []string{"Halo &lt;b&gt;&amp;", "baris &lt;script&gt;", "paragraf &amp; dua"} {
		if !strings.Contains(h, mau) {
			t.Fatalf("escape hilang, mencari %q", mau)
		}
	}
}

func TestHTMLTombolDanKode(t *testing.T) {
	s := Surat{Judul: "J", Teks: "isi", Kode: "AB12CD", CTALabel: "Buka", CTAURL: "https://x/y?a=1&b=2"}
	h := s.HTML(merekUji())
	if !strings.Contains(h, "AB12CD") {
		t.Fatal("kode tidak dirender")
	}
	// URL di atribut href harus ter-escape (& -> &amp;) dan tautan mentahnya ikut dicetak.
	if !strings.Contains(h, `href="https://x/y?a=1&amp;b=2"`) {
		t.Fatal("href CTA tidak ter-escape / hilang")
	}
	if strings.Count(h, "https://x/y") < 2 {
		t.Fatal("tautan mentah cadangan tidak dicetak di bawah tombol")
	}
}

// Tanpa CTA dan kode: tak boleh ada sisa markup tombol/kotak kosong.
func TestHTMLTanpaCTA(t *testing.T) {
	h := Surat{Judul: "J", Teks: "isi"}.HTML(merekUji())
	if strings.Contains(h, "Tombol tidak berfungsi") || strings.Contains(h, "Courier") {
		t.Fatal("markup tombol/kode muncul padahal tidak diminta")
	}
}

func TestTeksPolosMenurunkanParagrafHTML(t *testing.T) {
	s := Surat{Judul: "J", ParagrafHTML: []string{"<p>halo <strong>dunia</strong></p>", "baris &amp; dua"},
		CTALabel: "Buka", CTAURL: "https://x"}
	teks := s.TeksPolos()
	if strings.Contains(teks, "<") || !strings.Contains(teks, "halo dunia") || !strings.Contains(teks, "baris & dua") {
		t.Fatalf("teks polos salah: %q", teks)
	}
	if !strings.Contains(teks, "Buka: https://x") {
		t.Fatalf("CTA tak tercantum di teks polos: %q", teks)
	}
}

// Merek adalah parameter: warna & nama datang dari m, dan garis miring buntut
// basis URL dirapikan supaya tautan kaki tidak berujung "//faq".
func TestMerekDiterapkan(t *testing.T) {
	m := GoNgaji("https://contoh.id/")
	if m.BasisURL != "https://contoh.id" || m.TautanBantuan != "https://contoh.id/faq" {
		t.Fatalf("normalisasi basis URL salah: %+v", m)
	}
	m2 := Merek{Nama: "Toko <X>", WarnaPrimer: "#111111", WarnaPekat: "#222222", WarnaAksen: "#333333", BasisURL: "https://toko.id"}
	h := Surat{Judul: "J", Teks: "isi"}.HTML(m2)
	for _, mau := range []string{"#111111", "#222222", "#333333", "Toko &lt;X&gt;"} {
		if !strings.Contains(h, mau) {
			t.Fatalf("identitas merek tak diterapkan, mencari %q", mau)
		}
	}
	if strings.Contains(h, "Tanya Jawab") {
		t.Fatal("baris bantuan muncul padahal TautanBantuan kosong")
	}
}

// Preheader dipotong per HURUF: huruf multibyte (’ —) tak boleh terbelah jadi
// byte UTF-8 rusak seperti saat dipotong pre[:140].
func TestPreheaderDipotongPerRune(t *testing.T) {
	panjang := strings.Repeat("—’", 100) // 200 rune, 600 byte
	s := Surat{Judul: "J", Teks: panjang}
	pre := s.preheader()
	if !utf8Sah(pre) {
		t.Fatalf("preheader memuat UTF-8 rusak: %q", pre)
	}
	if n := len([]rune(pre)); n != batasPreheader {
		t.Fatalf("panjang preheader %d rune, mau %d", n, batasPreheader)
	}
	if (Surat{Judul: "J", Preheader: "eksplisit", Teks: "isi"}).preheader() != "eksplisit" {
		t.Fatal("Preheader eksplisit tidak dipakai")
	}
}

func utf8Sah(s string) bool { return utf8.ValidString(s) }

// Mode gelap: meta color-scheme light dark + lapisan <style>; palet inline
// tetap terang lengkap. Tak boleh ada hex 8 digit (Outlook desktop) dan warna
// redup lama yang gagal kontras.
func TestModeGelapDanPalet(t *testing.T) {
	h := Surat{Judul: "J", Teks: "isi", Status: "S", CTALabel: "B", CTAURL: "https://x"}.HTML(merekUji())
	for _, mau := range []string{`content="light dark"`, `prefers-color-scheme:dark`, `class="sb-kartu"`} {
		if !strings.Contains(h, mau) {
			t.Fatalf("lapisan mode gelap hilang, mencari %q", mau)
		}
	}
	if strings.Contains(h, "#8792a4") || strings.Contains(h, "#ffffffb3") {
		t.Fatal("warna lama (kontras rendah / hex 8 digit) masih dipakai")
	}
	for i := 0; i+9 < len(h); i++ {
		if h[i] == '#' && heksa(h[i+1:i+9]) && !heksa(h[i+9:i+10]) {
			t.Fatalf("hex 8 digit ditemukan: %q", h[i:i+9])
		}
	}
}

func heksa(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return s != ""
}

// Kerangka E4: chip status, sapaan, kotak kode + keterangan, tabel rincian,
// langkah bernomor, kotak bantuan, alasan — semuanya di-escape.
func TestKerangkaKartuRincian(t *testing.T) {
	s := Surat{
		Judul: "Beasiswamu diterima", Status: "Terpilih <x>", Nada: NadaSukses,
		Sapaan: SapaanSalam("  Ahmad   <b> "), Teks: "isi",
		Kode: "BSW-1", KodeKeterangan: "Bebas biaya penuh",
		Rincian: []Baris{{"Program", "Beasiswa & Batch 1"}, {"Kosong", ""}},
		Langkah: []string{"Satu", "", "Dua"},
		Bantuan: "Terkendala biaya?", BantuanLabel: "Lihat cicilan", BantuanURL: "https://x/biaya",
		Alasan: "Kamu menerima surel ini karena mengajukan beasiswa.",
	}
	h := s.HTML(merekUji())
	for _, mau := range []string{
		"Terpilih &lt;x&gt;", "sb-chip-sukses", "Assalamu’alaikum Ahmad &lt;b&gt;,", "BSW-1", "Bebas biaya penuh",
		"Beasiswa &amp; Batch 1", "Yang terjadi selanjutnya", ">2</div>", "Lihat cicilan", "mengajukan beasiswa",
		`alt="Logo Go Ngaji"`, "https://learning.gongaji.id/logogram-gongaji-white.png",
	} {
		if !strings.Contains(h, mau) {
			t.Fatalf("bagian kerangka hilang, mencari %q", mau)
		}
	}
	if strings.Contains(h, ">Kosong<") || strings.Contains(h, ">3</div>") {
		t.Fatal("baris rincian/langkah kosong ikut dirender")
	}
	teks := s.TeksPolos()
	for _, mau := range []string{"[Terpilih <x>]", "Program: Beasiswa & Batch 1", "1. Satu\n2. Dua", "Kode: BSW-1 (Bebas biaya penuh)", "Lihat cicilan: https://x/biaya"} {
		if !strings.Contains(teks, mau) {
			t.Fatalf("teks polos kurang %q:\n%s", mau, teks)
		}
	}
}

// Satu surel = satu bahasa: kerangka Inggris + tautan FAQ /en/faq; nilai
// bahasa tak dikenal jatuh ke ID.
func TestBahasaKerangka(t *testing.T) {
	en := Surat{Bahasa: "EN-us", Judul: "J", Teks: "isi", Langkah: []string{"a"}, CTALabel: "Go", CTAURL: "https://x"}.HTML(merekUji())
	for _, mau := range []string{`<html lang="en"`, "Button not working?", "What happens next", "please do not reply", "https://learning.gongaji.id/en/faq"} {
		if !strings.Contains(en, mau) {
			t.Fatalf("kerangka Inggris kurang %q", mau)
		}
	}
	if strings.Contains(en, "Tombol tidak berfungsi") {
		t.Fatal("string Indonesia bocor ke surel Inggris")
	}
	id := Surat{Bahasa: "fr", Judul: "J", Teks: "isi"}.HTML(merekUji())
	if !strings.Contains(id, `<html lang="id"`) || !strings.Contains(id, "mohon tidak membalas") {
		t.Fatal("bahasa tak dikenal tidak jatuh ke ID")
	}
	if SapaanSalam("") != "Assalamu’alaikum," {
		t.Fatal("sapaan tanpa nama salah")
	}
}

// Merek tanpa logo → tanpa <img>; basis kosong → tanpa tautan logo/bantuan.
func TestTanpaLogo(t *testing.T) {
	m := GoNgaji("")
	if m.LogoURL != "" || m.TautanBantuan != "" {
		t.Fatalf("basis kosong tak boleh menghasilkan URL relatif: %+v", m)
	}
	if strings.Contains(Surat{Judul: "J"}.HTML(m), "<img") {
		t.Fatal("<img> muncul padahal LogoURL kosong")
	}
}
