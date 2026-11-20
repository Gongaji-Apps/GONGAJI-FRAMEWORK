# Pedoman endpoint & DTO baru

Berlaku untuk endpoint, DTO, dan objek DB **baru** di semua service GONGAJI. Rute dan kunci
yang sudah dipakai aplikasi mobile **tidak** diubah demi pedoman ini (keputusan pemilik
produk 20 Nov 2026) — ubah hanya bila klien ikut diperbarui.

## Rute

- Ikuti pola yang sudah ada di service itu (sebagian besar: `GET /<sumber>/get`,
  `POST /<sumber>/create`, `PATCH /<sumber>/update/:primary_key`,
  `DELETE /<sumber>/delete/primary/:primary_key`). Jangan mencampur gaya REST murni dan
  gaya kata kerja di satu grup.
- Segmen path: kebab-case, kata benda tunggal, tanpa huruf besar dan tanpa garis miring
  di akhir.
- Parameter path: `:primary_key` untuk UUID baris utama sumber itu; `:<sumber>_code` untuk
  kode bisnis. Jangan membuat singkatan baru (`:doad_uuid`) atau nama yang menyesatkan.
- **Method harus sesuai efeknya**: baca = `GET`; mengubah data = `POST`/`PATCH`/`DELETE`.
  `GET` tidak boleh menulis data (kecuali pencatatan ringan seperti jumlah tayang yang
  sudah ada).
- Setiap rute yang menulis data wajib dijaga `auth` **dan** izin (`RequirePermission`)
  atau peran yang tepat — jangan hanya `auth`. Rute publik tanpa token harus disengaja,
  diberi komentar alasannya, dan dibatasi (rate limit / validasi).
- Tugas operasional (sinkronisasi, migrasi data, cron) bukan rute biasa: pakai
  `/system/*` dengan secret, skrip, atau migrasi DB.
- Jangan meninggalkan rute debug/uji (`/test`) di kode yang di-merge.

## Respons

- Selalu lewat paket `response`: objek → `Success`/`SuccessObject`, daftar →
  `SuccessArray` (`data`, `data_total`, `pagination`). Jangan membungkus `data` di dalam
  `data` (`gin.H{"data": …}` di dalam `Success`) dan jangan membuat amplop daftar sendiri
  (`rows`/`total`/`page`).
- Kunci JSON = nama kolom DB (snake_case, ber-prefiks tabel, mis. `notificationd_status`,
  `paymentt_amount`). Nilai turunan per baris memakai prefiks tabelnya juga
  (`meeting_substitute_offer_conflict`). Agregat/hitungan bebas diberi nama deskriptif
  (`count`, `sent`, `total_amount`).
- Kolom audit: `<p>_created_at/_updated_at/_deleted_at`, `<p>_created_by/_updated_by`
  (UUID). Rujukan pengguna: `<p>_user` (UUID).
- camelCase hanya untuk payload pihak ketiga (Apple, Midtrans, plugin notifikasi).

## Parameter query daftar

- Paging: `limit` + `pagination` (paket `query`, lewat `binding.ListQuery`).
- Pencarian: `keyword` + `type_search`.
- Filter eksak: nama kolom (`batch_uuid`, `<p>_status`).
- Parameter UUID pengguna tetap `user_uuid` / `actor_uuid`.

## Body request

- Kunci body = nama kolom yang diisi (`demor_reason`, bukan `reason`); hindari kunci
  generik satu kata (`uuid`, `type`, `note`, `on`) untuk data yang disimpan.

## Objek DB

Lihat CLAUDE.md repo GONGAJI-DB: prefiks kolom per tabel, `pk_<tabel>`,
`fk_<tabel>__<kolom>`, `ck_`/`uq_`/`ix_<tabel>__<makna>`, nama eksplisit (bukan bawaan
PostgreSQL). Nama indeks unik dipakai untuk pesan galat API (NGAJI `label_galat.go`).
