// Package surat merender surel HTML bermerek yang seragam lintas service
// GoNgaji — beserta versi text/plain untuk multipart.
//
// Kerangkanya "kartu + tabel rincian": pita kepala (logo + wordmark), garis
// aksen, lalu satu kartu berisi chip status, judul, sapaan bernama, paragraf,
// kotak kode, tabel rincian label/nilai, langkah bernomor ("yang terjadi
// selanjutnya"), tombol CTA + tautan mentah cadangan, kotak bantuan sekunder,
// dan catatan; di bawah kartu, kaki "surel otomatis" + alasan menerima. Satu
// kerangka untuk SEMUA tipe notifikasi: bagian yang kosong tidak dirender.
//
// HTML surel bukan HTML web: klien surel (terutama Outlook & Gmail) memangkas
// <style>, mengabaikan flex/grid, dan memblokir JavaScript sepenuhnya. Karena
// itu cetakannya tabel role=presentation + CSS inline, lebar 600px, tumpukan
// font sistem, tombol "antipeluru" (<td> berwarna + <a>), dan "interaktif"
// dalam surel berarti tombol tautan — bukan skrip. Satu-satunya <style> adalah
// lapisan mode gelap opsional: gaya inline tetap palet terang yang lengkap,
// sedangkan klien yang menghormati prefers-color-scheme (Apple Mail,
// Outlook.com) menimpanya lewat kelas sb-*.
//
// Bahasa: satu surel = satu bahasa (Surat.Bahasa, "id" bawaan atau "en").
// Kerangka (tautan cadangan, kaki, judul langkah) diterjemahkan di sini;
// konten (Judul, Teks, Rincian, ...) tetap tanggung jawab pemanggil.
//
// Identitas merek adalah parameter (bukan konstanta) supaya beberapa produk
// pada satu akun pengirim bisa berbagi struktur tanpa berbagi warna/nama.
// Package ini tidak membaca env — service pemanggil yang memutuskan mereknya.
//
// Contoh:
//
//	m := surat.GoNgaji("https://learning.gongaji.id")
//	s := surat.Surat{
//		Judul:   "Termin 2 jatuh tempo",
//		Sapaan:  surat.SapaanSalam("Budi"),
//		Status:  "Menunggu pembayaran",
//		Nada:    surat.NadaTunggu,
//		Teks:    "Termin 2 untuk kelas Pemula 1 jatuh tempo pekan ini.",
//		Rincian: []surat.Baris{{Label: "Jatuh tempo", Nilai: "Jumat, 28 Agu 2026"}},
//		CTALabel: "Lihat Tagihan",
//		CTAURL:   "https://learning.gongaji.id/tagihan",
//	}
//	msg := mailer.Message{To: []string{to}, Subject: s.Judul,
//		HTMLBody: s.HTML(m), TextBody: s.TeksPolos()}
package surat

import (
	"html"
	"strconv"
	"strings"
	"time"
)

// Bahasa yang didukung kerangka surel. Nilai lain jatuh ke ID.
const (
	ID = "id"
	EN = "en"
)

// NormalisasiBahasa memetakan nilai bebas ("en", "EN-us", "", "fr") ke bahasa
// yang didukung; apa pun selain Inggris menjadi ID.
func NormalisasiBahasa(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == EN || strings.HasPrefix(v, EN+"-") || strings.HasPrefix(v, EN+"_") {
		return EN
	}
	return ID
}

// Nada menentukan warna chip status. Warna bukan satu-satunya pembawa makna —
// teks Status selalu tercetak.
type Nada string

const (
	NadaInfo   Nada = "info"   // indigo: informasi netral (bawaan)
	NadaTunggu Nada = "tunggu" // kuning: sedang diproses/menunggu
	NadaSukses Nada = "sukses" // hijau: diterima/berhasil
	NadaGagal  Nada = "gagal"  // merah: ditolak/dibatalkan
)

// Baris adalah satu baris tabel rincian (label kiri redup, nilai kanan tebal).
type Baris struct {
	Label string
	Nilai string
}

// Merek adalah identitas visual pengirim: nama, logo, warna, dan tautan kaki.
// Nol value TIDAK siap pakai — bangun lewat GoNgaji() atau isi lengkap sendiri.
type Merek struct {
	Nama    string // wordmark di pita kepala, mis. "Go Ngaji"
	Tagline string // teks kecil di samping wordmark, mis. "Belajar Ngaji"

	// LogoURL (opsional) = URL absolut PNG yang di-host publik, tampil 32x32 di
	// kiri wordmark. PNG, bukan SVG: Gmail & Outlook tidak merender SVG. Varian
	// putih karena latarnya pita WarnaPekat. Kosong = wordmark teks saja.
	LogoURL string

	WarnaPrimer string // tombol CTA + tautan, mis. "#5145b9"
	WarnaPekat  string // pita kepala + teks kode, mis. "#3f3596"
	WarnaAksen  string // garis tipis di bawah kepala, mis. "#ffc748"

	// BasisURL menautkan wordmark di kaki surel; TautanBantuan (opsional)
	// menambah baris "Butuh bantuan?" dan TautanBantuanEN menggantikannya untuk
	// surel berbahasa Inggris (kosong = pakai TautanBantuan).
	BasisURL        string
	TautanBantuan   string
	TautanBantuanEN string

	// TautanPengaturan (opsional) menambah tautan "Atur notifikasi" di kaki.
	// Hanya isi bila halaman pengaturan notifikasi benar-benar ada.
	TautanPengaturan string
}

// GoNgaji memulangkan merek bawaan Go Ngaji (indigo + emas, palet web santri).
// basisURL = asal web publik produk; tautan bantuannya <basisURL>/faq (dan
// <basisURL>/en/faq untuk surel Inggris), logonya logogram putih yang
// disajikan web santri yang sama di <basisURL>/logogram-gongaji-white.png.
func GoNgaji(basisURL string) Merek {
	basisURL = strings.TrimRight(strings.TrimSpace(basisURL), "/")
	m := Merek{
		Nama:        "Go Ngaji",
		Tagline:     "Belajar Ngaji",
		WarnaPrimer: "#5145b9",
		WarnaPekat:  "#3f3596",
		WarnaAksen:  "#ffc748",
		BasisURL:    basisURL,
	}
	if basisURL != "" {
		m.LogoURL = basisURL + "/logogram-gongaji-white.png"
		m.TautanBantuan = basisURL + "/faq"
		m.TautanBantuanEN = basisURL + "/en/faq"
	}
	return m
}

// SapaanSalam merangkai sapaan bernama khas Go Ngaji ("Assalamu’alaikum Budi,").
// Nama kosong tetap menghasilkan salam tanpa nama. Dipakai untuk kedua bahasa —
// salamnya memang tidak diterjemahkan.
func SapaanSalam(nama string) string {
	nama = strings.Join(strings.Fields(nama), " ")
	if nama == "" {
		return "Assalamu’alaikum,"
	}
	return "Assalamu’alaikum " + nama + ","
}

// Surat adalah konten satu surel. Semua bidang opsional kecuali Judul; bagian
// yang kosong tidak dirender. Semua teks di-escape — pemanggil TIDAK perlu (dan
// tidak boleh) menyisipkan HTML kecuali lewat ParagrafHTML.
type Surat struct {
	// Bahasa kerangka surel: ID (bawaan) atau EN. Lihat NormalisasiBahasa.
	Bahasa string

	// Judul tampil sebagai kepala kartu (subjek surel diatur pemanggil,
	// biasanya sama).
	Judul string
	// Preheader = cuplikan di daftar kotak masuk (di samping subjek). Kosong =
	// diturunkan dari paragraf pertama. Dipotong 140 HURUF (rune), bukan byte.
	Preheader string

	// Status tampil sebagai chip kecil di atas judul (mis. "Sedang ditinjau");
	// Nada memilih warnanya (kosong = NadaInfo).
	Status string
	Nada   Nada

	// Sapaan = baris pembuka bernama, mis. SapaanSalam("Ahmad").
	Sapaan string

	// Teks = isi polos; baris kosong memisahkan paragraf.
	Teks string
	// ParagrafHTML dipakai pemanggil yang butuh markup di isi (mis. <strong>).
	// Bila terisi, Teks hanya dipakai untuk versi teks-polos multipart; bila
	// Teks kosong, versi polos diturunkan dari ParagrafHTML dengan tag dilucuti.
	ParagrafHTML []string

	// Kode ditampilkan besar di kotak tersendiri (OTP, voucher, kode unduh);
	// KodeKeterangan = baris kecil di bawahnya (mis. "Bebas biaya penuh").
	Kode           string
	KodeKeterangan string

	// Rincian = tabel 2 kolom label/nilai (program, tanggal, nominal, ...).
	Rincian []Baris

	// Langkah = daftar bernomor di bawah judul "Yang terjadi selanjutnya".
	Langkah []string

	// CTALabel+CTAURL menjadi tombol; kosong = tanpa tombol. Tautan mentahnya
	// ikut dicetak di bawah tombol — sebagian klien surel menggagalkan tombol
	// bergaya tapi selalu merender tautan teks.
	CTALabel string
	CTAURL   string

	// Bantuan = kotak sekunder di bawah tombol (mis. "Terkendala biaya?"),
	// opsional dengan tautan BantuanLabel → BantuanURL.
	Bantuan      string
	BantuanLabel string
	BantuanURL   string

	// Catatan tampil kecil & redup di bawah isi (masa berlaku kode, "abaikan
	// bila bukan kamu", dsb).
	Catatan string

	// Alasan = baris kaki "Kamu menerima surel ini karena ..." (kalimat utuh,
	// pemanggil yang merangkai sesuai bahasanya).
	Alasan string
}

// Warna netral struktur (kanvas, kartu, teks) bukan bagian identitas merek —
// konstanta, bukan bidang Merek, supaya dua produk beda warna tetap terasa
// satu keluarga. Semua hex 6 digit (Outlook desktop tak mengenal 8 digit).
// Kontras (WCAG): redup #5f6878 = 5,1:1 di kanvas & 5,6:1 di kartu putih.
const (
	warnaLatar   = "#f1f3f9"
	warnaKartu   = "#ffffff"
	warnaGaris   = "#e3e7f0"
	warnaLunak   = "#f6f7fb"
	warnaTinta   = "#1f2a37"
	warnaIsi     = "#4a5568"
	warnaRedup   = "#5f6878"
	warnaKodeBg  = "#ecebfb"
	warnaTagline = "#d6d3f0" // di atas pita #3f3596 = 6,7:1
	fontTumpukan = "-apple-system,'Segoe UI',Roboto,Helvetica,Arial,sans-serif"
	fontKode     = "'Courier New',Courier,monospace"
)

// chip: pasangan latar/tinta per nada (terang). Padanan gelapnya di gayaGelap.
var chip = map[Nada][2]string{
	NadaInfo:   {"#ecebfb", "#3f3596"},
	NadaTunggu: {"#fdf3e1", "#7a5b18"},
	NadaSukses: {"#e7f6ec", "#166534"},
	NadaGagal:  {"#fdecec", "#9b1c1c"},
}

// gayaGelap: satu-satunya <style>. Klien yang mengabaikannya tetap mendapat
// palet terang inline yang utuh; [data-ogsc]/[data-ogsb] = Outlook.com gelap.
const gayaGelap = `<style>` +
	`:root{color-scheme:light dark;supported-color-schemes:light dark;}` +
	`@media (prefers-color-scheme:dark){` + aturanGelap + `}` +
	`</style>`

const aturanGelap = `.sb-latar{background:#0f1420!important;}` +
	`.sb-kartu{background:#171d2b!important;border-color:#2a3447!important;}` +
	`.sb-lunak{background:#1c2333!important;}` +
	`.sb-kodebg{background:#262347!important;}` +
	`.sb-garis{border-color:#2a3447!important;}` +
	`.sb-tinta{color:#e6ebf2!important;}` +
	`.sb-isi{color:#c3cad6!important;}` +
	`.sb-redup{color:#97a1b0!important;}` +
	`.sb-tautan{color:#a5a0f5!important;}` +
	`.sb-kodeteks{color:#e6ebf2!important;}` +
	`.sb-nomor{background:#262347!important;color:#c9c5ea!important;}` +
	`.sb-chip-info{background:#262347!important;color:#c9c5ea!important;}` +
	`.sb-chip-tunggu{background:#2c2414!important;color:#fbbf24!important;}` +
	`.sb-chip-sukses{background:#15291f!important;color:#34d399!important;}` +
	`.sb-chip-gagal{background:#3a1a1e!important;color:#fca5a5!important;}`

// kamus: string kerangka per bahasa.
type kamus struct {
	tombolGagal string // "Tombol tidak berfungsi? Buka tautan ini:"
	otomatisA   string // "Surel otomatis dari "
	otomatisB   string // " — mohon tidak membalas."
	bantuanA    string // "Butuh bantuan? Kunjungi "
	tanyaJawab  string
	langkah     string
	kode        string
	aturNotif   string
	logoAlt     string // "Logo "
}

var kamusBahasa = map[string]kamus{
	ID: {
		tombolGagal: "Tombol tidak berfungsi? Buka tautan ini:",
		otomatisA:   "Surel otomatis dari ",
		otomatisB:   " — mohon tidak membalas.",
		bantuanA:    "Butuh bantuan? Kunjungi ",
		tanyaJawab:  "Tanya Jawab",
		langkah:     "Yang terjadi selanjutnya",
		kode:        "Kode",
		aturNotif:   "Atur notifikasi",
		logoAlt:     "Logo ",
	},
	EN: {
		tombolGagal: "Button not working? Open this link:",
		otomatisA:   "Automated email from ",
		otomatisB:   " — please do not reply.",
		bantuanA:    "Need help? Visit our ",
		tanyaJawab:  "FAQ",
		langkah:     "What happens next",
		kode:        "Code",
		aturNotif:   "Manage notifications",
		logoAlt:     "Logo ",
	},
}

func (s Surat) bahasa() string { return NormalisasiBahasa(s.Bahasa) }
func (s Surat) kamus() kamus   { return kamusBahasa[s.bahasa()] }

// tautanBantuan memilih tautan Tanya Jawab sesuai bahasa surel.
func (m Merek) tautanBantuan(bahasa string) string {
	if bahasa == EN && m.TautanBantuanEN != "" {
		return m.TautanBantuanEN
	}
	return m.TautanBantuan
}

// batasPreheader dalam HURUF. Dulu dipotong per byte (pre[:140]) sehingga
// huruf multibyte (’ — é) bisa terbelah jadi karakter rusak.
const batasPreheader = 140

// potongRune memotong s menjadi paling banyak n rune (bukan byte).
func potongRune(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

// preheader: Preheader eksplisit, atau paragraf pertama isi — spasi dirapikan,
// dipotong per rune.
func (s Surat) preheader() string {
	pre := s.Preheader
	if pre == "" {
		pre = s.Teks
		if len(s.ParagrafHTML) > 0 {
			pre = lucutiTag(s.ParagrafHTML[0])
		}
		if pre == "" {
			pre = s.Status
		}
	}
	return potongRune(strings.Join(strings.Fields(pre), " "), batasPreheader)
}

// HTML merender surat menjadi dokumen surel utuh dengan identitas m.
func (s Surat) HTML(m Merek) string {
	k := s.kamus()
	bhs := s.bahasa()
	var b strings.Builder
	b.Grow(8192)

	tulis := func(xs ...string) {
		for _, x := range xs {
			b.WriteString(x)
		}
	}
	esc := html.EscapeString
	font := func(berat, ukuran string) string { return `font:` + berat + ` ` + ukuran + ` ` + fontTumpukan + `;` }

	tulis(`<!doctype html><html lang="`, bhs, `" dir="ltr"><head><meta charset="utf-8">`,
		`<meta name="viewport" content="width=device-width,initial-scale=1">`,
		`<meta http-equiv="x-ua-compatible" content="ie=edge">`,
		`<meta name="color-scheme" content="light dark"><meta name="supported-color-schemes" content="light dark">`,
		`<title>`, esc(s.Judul), `</title>`, gayaGelap, `</head>`,
		`<body class="sb-latar" style="margin:0;padding:0;background:`, warnaLatar, `;-webkit-text-size-adjust:100%;">`)

	// Preheader: cuplikan di daftar masuk, di samping subjek. Tanpa ini yang
	// tampil adalah teks pertama dokumen — nama merek di pita kepala. Pengisi
	// &zwnj;&nbsp; mencegah klien menyambung cuplikan dengan teks isi.
	tulis(`<div style="display:none;max-height:0;overflow:hidden;mso-hide:all;font-size:1px;line-height:1px;color:`, warnaLatar, `;">`,
		esc(s.preheader()), strings.Repeat("&#847;&zwnj;&nbsp;", 24), `</div>`)

	tulis(`<table role="presentation" class="sb-latar" width="100%" cellpadding="0" cellspacing="0" border="0" style="background:`, warnaLatar, `;">`,
		`<tr><td align="center" style="padding:32px 16px;">`,
		`<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="max-width:600px;width:100%;">`)

	// Pita kepala: logo + wordmark di warna pekat (tetap gelap di kedua mode).
	tulis(`<tr><td style="background:`, m.WarnaPekat, `;border-radius:16px 16px 0 0;padding:20px 28px;">`,
		`<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>`)
	if m.LogoURL != "" {
		tulis(`<td style="padding-right:10px;vertical-align:middle;">`,
			`<img src="`, esc(m.LogoURL), `" width="32" height="32" alt="`, esc(k.logoAlt+m.Nama), `" `,
			`style="display:block;width:32px;height:32px;border:0;outline:none;text-decoration:none;color:#ffffff;`, font("600", "10px"), `"></td>`)
	}
	tulis(`<td style="vertical-align:middle;">`,
		`<span style="`, font("800", "19px"), `color:#ffffff;letter-spacing:.02em;">`, esc(m.Nama), `</span>`)
	if m.Tagline != "" {
		tulis(`<span style="`, font("600", "10.5px"), `color:`, warnaTagline, `;letter-spacing:.14em;text-transform:uppercase;">&nbsp;&nbsp;·&nbsp;&nbsp;`,
			esc(m.Tagline), `</span>`)
	}
	tulis(`</td></tr></table></td></tr>`,
		`<tr><td style="background:`, m.WarnaAksen, `;height:4px;font-size:0;line-height:0;">&nbsp;</td></tr>`)

	// Kartu isi.
	tulis(`<tr><td class="sb-kartu" style="background:`, warnaKartu, `;border:1px solid `, warnaGaris, `;border-top:0;border-radius:0 0 16px 16px;padding:28px;">`)

	if s.Status != "" {
		nada := s.Nada
		if _, ok := chip[nada]; !ok {
			nada = NadaInfo
		}
		w := chip[nada]
		tulis(`<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 10px;"><tr>`,
			`<td class="sb-chip-`, string(nada), `" style="background:`, w[0], `;color:`, w[1], `;border-radius:999px;padding:4px 10px;`,
			font("700", "11px"), `letter-spacing:.08em;text-transform:uppercase;">`, esc(s.Status), `</td></tr></table>`)
	}

	tulis(`<h1 class="sb-tinta" style="margin:0 0 12px;`, font("700", "21px/1.35"), `color:`, warnaTinta, `;">`, esc(s.Judul), `</h1>`)

	paragraf := func(isiHTML string) {
		tulis(`<p class="sb-isi" style="margin:0 0 12px;`, font("400", "15px/1.65"), `color:`, warnaIsi, `;">`, isiHTML, `</p>`)
	}
	if s.Sapaan != "" {
		paragraf(esc(s.Sapaan))
	}
	for _, p := range s.paragrafSemua() {
		paragraf(p)
	}

	if s.Kode != "" {
		tulis(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:4px 0 14px;"><tr>`,
			`<td class="sb-kodebg" align="center" style="background:`, warnaKodeBg, `;border-radius:12px;padding:16px;">`,
			`<span class="sb-kodeteks" style="font:700 24px `, fontKode, `;color:`, m.WarnaPekat, `;letter-spacing:.25em;word-break:break-all;">`, esc(s.Kode), `</span>`)
		if s.KodeKeterangan != "" {
			tulis(`<br><span class="sb-redup" style="`, font("400", "12px/1.8"), `color:`, warnaRedup, `;">`, esc(s.KodeKeterangan), `</span>`)
		}
		tulis(`</td></tr></table>`)
	}

	if baris := s.rincianTerisi(); len(baris) > 0 {
		tulis(`<table role="presentation" class="sb-lunak" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:6px 0 16px;background:`, warnaLunak, `;border-radius:12px;border-collapse:separate;">`)
		for i, r := range baris {
			garis := `border-bottom:1px solid ` + warnaGaris + `;`
			if i == len(baris)-1 {
				garis = ""
			}
			tulis(`<tr>`,
				`<td class="sb-redup sb-garis" width="42%" style="padding:9px 12px 9px 14px;vertical-align:top;`, garis, font("400", "14px/1.5"), `color:`, warnaRedup, `;">`, esc(r.Label), `</td>`,
				`<td class="sb-tinta sb-garis" style="padding:9px 14px 9px 0;vertical-align:top;`, garis, font("600", "14px/1.5"), `color:`, warnaTinta, `;">`, esc(r.Nilai), `</td>`,
				`</tr>`)
		}
		tulis(`</table>`)
	}

	if langkah := s.langkahTerisi(); len(langkah) > 0 {
		tulis(`<p class="sb-tinta" style="margin:0 0 6px;`, font("700", "15px/1.5"), `color:`, warnaTinta, `;">`, esc(k.langkah), `</p>`,
			`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 16px;">`)
		for i, l := range langkah {
			tulis(`<tr><td width="32" style="padding:5px 10px 5px 0;vertical-align:top;">`,
				`<div class="sb-nomor" style="width:22px;height:22px;border-radius:11px;background:`, warnaKodeBg, `;color:`, m.WarnaPrimer, `;text-align:center;`, font("700", "12px/22px"), `">`,
				strconv.Itoa(i+1), `</div></td>`,
				`<td class="sb-isi" style="padding:5px 0;vertical-align:top;`, font("400", "14px/1.55"), `color:`, warnaIsi, `;">`, esc(l), `</td></tr>`)
		}
		tulis(`</table>`)
	}

	if s.CTALabel != "" && s.CTAURL != "" {
		u := esc(s.CTAURL)
		tulis(`<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:8px 0 6px;"><tr>`,
			`<td align="center" bgcolor="`, m.WarnaPrimer, `" style="background:`, m.WarnaPrimer, `;border-radius:12px;">`,
			`<a href="`, u, `" target="_blank" style="display:inline-block;padding:13px 28px;`, font("600", "15px/1.2"), `color:#ffffff;text-decoration:none;border-radius:12px;">`,
			esc(s.CTALabel), `</a></td></tr></table>`,
			// Tautan mentah untuk klien yang mematikan gaya tombol.
			`<p class="sb-redup" style="margin:6px 0 0;`, font("400", "12px/1.6"), `color:`, warnaRedup, `;word-break:break-all;">`,
			esc(k.tombolGagal), ` <a class="sb-tautan" href="`, u, `" style="color:`, m.WarnaPrimer, `;">`, u, `</a></p>`)
	}

	if s.Bantuan != "" {
		tulis(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:16px 0 0;"><tr>`,
			`<td class="sb-lunak sb-isi" style="background:`, warnaLunak, `;border-radius:12px;padding:12px 14px;`, font("400", "13px/1.6"), `color:`, warnaIsi, `;">`,
			esc(s.Bantuan))
		if s.BantuanLabel != "" && s.BantuanURL != "" {
			tulis(` <a class="sb-tautan" href="`, esc(s.BantuanURL), `" style="color:`, m.WarnaPrimer, `;font-weight:600;">`, esc(s.BantuanLabel), `</a>`)
		}
		tulis(`</td></tr></table>`)
	}

	if s.Catatan != "" {
		tulis(`<p class="sb-redup sb-garis" style="margin:18px 0 0;padding-top:14px;border-top:1px solid `, warnaGaris, `;`, font("400", "13px/1.6"), `color:`, warnaRedup, `;">`,
			esc(s.Catatan), `</p>`)
	}

	tulis(`</td></tr>`)

	// Kaki: identitas, pengingat surel otomatis, alasan menerima, hak cipta.
	tautan := func(href, label string) string {
		return `<a class="sb-tautan" href="` + esc(href) + `" style="color:` + m.WarnaPrimer + `;text-decoration:none;font-weight:600;">` + esc(label) + `</a>`
	}
	tulis(`<tr><td align="center" style="padding:18px 12px 0;">`,
		`<p class="sb-redup" style="margin:0;`, font("400", "12px/1.7"), `color:`, warnaRedup, `;">`,
		esc(k.otomatisA))
	if m.BasisURL != "" {
		tulis(tautan(m.BasisURL, m.Nama))
	} else {
		tulis(esc(m.Nama))
	}
	tulis(esc(k.otomatisB))
	if tb := m.tautanBantuan(bhs); tb != "" {
		tulis(`<br>`, esc(k.bantuanA), tautan(tb, k.tanyaJawab), `.`)
	}
	if s.Alasan != "" || m.TautanPengaturan != "" {
		tulis(`<br>`)
		if s.Alasan != "" {
			tulis(esc(s.Alasan))
		}
		if m.TautanPengaturan != "" {
			if s.Alasan != "" {
				tulis(` `)
			}
			tulis(tautan(m.TautanPengaturan, k.aturNotif))
		}
	}
	tulis(`<br>© `, strconv.Itoa(time.Now().Year()), ` `, esc(m.Nama), `</p></td></tr>`)

	tulis(`</table></td></tr></table></body></html>`)
	return b.String()
}

// TeksPolos merender versi text/plain untuk bagian multipart — klien tanpa HTML
// dan penyaring spam sama-sama menghargainya. Urutannya mengikuti versi HTML.
func (s Surat) TeksPolos() string {
	k := s.kamus()
	var bagian []string
	tambah := func(x string) {
		if x = strings.TrimSpace(x); x != "" {
			bagian = append(bagian, x)
		}
	}
	if s.Status != "" {
		tambah("[" + s.Status + "]")
	}
	tambah(s.Sapaan)
	if s.Teks != "" {
		tambah(s.Teks)
	} else {
		for _, p := range s.ParagrafHTML {
			tambah(lucutiTag(p))
		}
	}
	if s.Kode != "" {
		kode := k.kode + ": " + s.Kode
		if s.KodeKeterangan != "" {
			kode += " (" + s.KodeKeterangan + ")"
		}
		tambah(kode)
	}
	if baris := s.rincianTerisi(); len(baris) > 0 {
		var r strings.Builder
		for i, x := range baris {
			if i > 0 {
				r.WriteString("\n")
			}
			r.WriteString(x.Label + ": " + x.Nilai)
		}
		tambah(r.String())
	}
	if langkah := s.langkahTerisi(); len(langkah) > 0 {
		var r strings.Builder
		r.WriteString(k.langkah + ":")
		for i, l := range langkah {
			r.WriteString("\n" + strconv.Itoa(i+1) + ". " + l)
		}
		tambah(r.String())
	}
	if s.CTALabel != "" && s.CTAURL != "" {
		tambah(s.CTALabel + ": " + s.CTAURL)
	}
	if s.Bantuan != "" {
		x := s.Bantuan
		if s.BantuanLabel != "" && s.BantuanURL != "" {
			x += " " + s.BantuanLabel + ": " + s.BantuanURL
		}
		tambah(x)
	}
	tambah(s.Catatan)
	tambah(s.Alasan)
	return strings.Join(bagian, "\n\n")
}

// rincianTerisi: baris rincian tanpa label/nilai kosong.
func (s Surat) rincianTerisi() []Baris {
	out := make([]Baris, 0, len(s.Rincian))
	for _, r := range s.Rincian {
		if strings.TrimSpace(r.Label) != "" && strings.TrimSpace(r.Nilai) != "" {
			out = append(out, r)
		}
	}
	return out
}

// langkahTerisi: langkah tanpa entri kosong.
func (s Surat) langkahTerisi() []string {
	out := make([]string, 0, len(s.Langkah))
	for _, l := range s.Langkah {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// paragrafSemua: isi kartu sebagai potongan HTML per paragraf, apa pun bentuk
// masukannya (Teks polos di-escape + dipecah di baris kosong; ParagrafHTML
// dipakai apa adanya).
func (s Surat) paragrafSemua() []string {
	if len(s.ParagrafHTML) > 0 {
		return s.ParagrafHTML
	}
	potongan := strings.Split(strings.ReplaceAll(s.Teks, "\r\n", "\n"), "\n\n")
	out := make([]string, 0, len(potongan))
	for _, p := range potongan {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, strings.ReplaceAll(html.EscapeString(p), "\n", "<br>"))
	}
	return out
}

// lucutiTag membuang tag HTML sederhana untuk versi teks-polos/preheader.
// Bukan sanitiser umum — masukannya markup milik service sendiri.
func lucutiTag(s string) string {
	var b strings.Builder
	dalam := false
	for _, r := range s {
		switch {
		case r == '<':
			dalam = true
		case r == '>':
			dalam = false
		case !dalam:
			b.WriteRune(r)
		}
	}
	return html.UnescapeString(strings.TrimSpace(b.String()))
}
