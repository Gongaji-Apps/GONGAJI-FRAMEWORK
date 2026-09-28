package errors

import (
	"strings"
	"sync"
)

// ========================================================
// ==================== LABEL MANUSIAWI ===================
// ========================================================
//
// Pesan galat adalah untuk pengguna, bukan untuk log. Service mendaftarkan
// label manusiawi sekali saat start-up (mis. di main.go / routes.go) agar
// galat DB & validasi tak pernah memperlihatkan nama tabel, kolom, atau
// constraint mentah:
//
//	errors.RegisterTableLabels(map[string]string{"ngaji.exam_mstr": "Konfigurasi ujian"})
//	errors.RegisterFieldLabels(map[string]string{"batch_uuid": "Batch"})
//	errors.RegisterConstraintMessages(map[string]string{
//	    "uq_exam_mstr__batch_level": "Konfigurasi ujian untuk level ini sudah ada di batch tersebut.",
//	})
//
// Tanpa pendaftaran pun pesan tetap manusiawi (label diturunkan dari nama
// snake_case), hanya kurang spesifik.

var (
	labelMu          sync.RWMutex
	tableLabels      = map[string]string{}
	fieldLabels      = map[string]string{}
	constraintLabels = map[string]string{}
)

// RegisterTableLabels mendaftarkan label untuk nama tabel (boleh berskema,
// mis. "ngaji.exam_mstr", atau tanpa skema).
func RegisterTableLabels(m map[string]string) {
	labelMu.Lock()
	defer labelMu.Unlock()
	for k, v := range m {
		tableLabels[strings.ToLower(k)] = v
	}
}

// RegisterFieldLabels mendaftarkan label untuk nama medan/kolom (snake_case,
// sama dengan tag json / nama kolom DB).
func RegisterFieldLabels(m map[string]string) {
	labelMu.Lock()
	defer labelMu.Unlock()
	for k, v := range m {
		fieldLabels[strings.ToLower(k)] = v
	}
}

// RegisterConstraintMessages mendaftarkan pesan utuh per nama constraint /
// index unik Postgres. Pesan ini menang atas pesan bawaan.
func RegisterConstraintMessages(m map[string]string) {
	labelMu.Lock()
	defer labelMu.Unlock()
	for k, v := range m {
		constraintLabels[strings.ToLower(k)] = v
	}
}

// TableLabel mengembalikan label tabel; "" bila tak dikenal.
func TableLabel(table string) string {
	t := strings.ToLower(strings.TrimSpace(table))
	if t == "" {
		return ""
	}
	labelMu.RLock()
	defer labelMu.RUnlock()
	if v, ok := tableLabels[t]; ok {
		return v
	}
	if i := strings.LastIndex(t, "."); i >= 0 {
		if v, ok := tableLabels[t[i+1:]]; ok {
			return v
		}
	}
	return ""
}

// FieldLabel mengembalikan label medan: terdaftar, atau diturunkan dari
// snake_case ("batch_class_start_date" → "Batch class start date"; akhiran
// _uuid/_id dibuang: "level_uuid" → "Level").
func FieldLabel(field string) string {
	f := strings.ToLower(strings.TrimSpace(field))
	if f == "" {
		return "Isian"
	}
	labelMu.RLock()
	v, ok := fieldLabels[f]
	labelMu.RUnlock()
	if ok {
		return v
	}
	return HumanizeIdentifier(f)
}

func constraintMessage(name string) string {
	if name == "" {
		return ""
	}
	labelMu.RLock()
	defer labelMu.RUnlock()
	return constraintLabels[strings.ToLower(name)]
}

// HumanizeIdentifier mengubah identifier snake_case menjadi frasa kalimat.
func HumanizeIdentifier(id string) string {
	s := strings.ToLower(strings.TrimSpace(id))
	if i := strings.LastIndex(s, "."); i >= 0 {
		s = s[i+1:]
	}
	for _, suf := range []string{"_uuid", "_id", "_mstr", "_trx", "_log"} {
		if strings.HasSuffix(s, suf) && len(s) > len(suf) {
			s = strings.TrimSuffix(s, suf)
		}
	}
	s = strings.TrimSpace(strings.ReplaceAll(s, "_", " "))
	if s == "" {
		return "Data"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// labelAtauData: label tabel, atau "Data" bila tak terdaftar (nama tabel
// mentah tak pernah ditampilkan).
func labelAtauData(table string) string {
	if l := TableLabel(table); l != "" {
		return l
	}
	return "Data"
}
