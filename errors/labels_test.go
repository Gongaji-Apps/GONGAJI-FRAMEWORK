package errors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestHumanizeIdentifier(t *testing.T) {
	cases := map[string]string{
		"level_uuid":             "Level",
		"batch_class_start_date": "Batch class start date",
		"ngaji.exam_mstr":        "Exam",
		"":                       "Data",
	}
	for in, want := range cases {
		if got := HumanizeIdentifier(in); got != want {
			t.Errorf("HumanizeIdentifier(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeDBError_TanpaNamaMentah(t *testing.T) {
	dup := &pgconn.PgError{Code: "23505", SchemaName: "ngaji", TableName: "exam_mstr", ConstraintName: "uq_contoh_tak_terdaftar"}
	err := NormalizeDBError(fmt.Errorf("gorm: %w", dup), "ngaji.exam_mstr")
	app, ok := err.(*AppError)
	if !ok || app.Code != Conflict {
		t.Fatalf("mau Conflict, dapat %#v", err)
	}
	for _, mentah := range []string{"[", "exam_mstr", "uq_", "Duplicate"} {
		if strings.Contains(app.Message, mentah) {
			t.Errorf("pesan memuat %q: %q", mentah, app.Message)
		}
	}
}

func TestNormalizeDBError_LabelDanConstraint(t *testing.T) {
	RegisterTableLabels(map[string]string{"ngaji.uji_label_mstr": "Konfigurasi uji"})
	RegisterConstraintMessages(map[string]string{"uq_uji_label": "Konfigurasi uji untuk level ini sudah ada."})

	dup := &pgconn.PgError{Code: "23505", SchemaName: "ngaji", TableName: "uji_label_mstr", ConstraintName: "lain"}
	if got := NormalizeDBError(dup, "").(*AppError).Message; !strings.HasPrefix(got, "Konfigurasi uji yang sama") {
		t.Errorf("label tabel tak dipakai: %q", got)
	}
	byName := &pgconn.PgError{Code: "23505", ConstraintName: "UQ_UJI_LABEL"}
	if got := NormalizeDBError(byName, "").(*AppError).Message; got != "Konfigurasi uji untuk level ini sudah ada." {
		t.Errorf("pesan constraint tak dipakai: %q", got)
	}
	fkDel := &pgconn.PgError{Code: "23503", SchemaName: "ngaji", TableName: "uji_label_mstr", Message: `update or delete on table "x" violates foreign key constraint "y" on table "z" ... is still referenced`}
	if got := NormalizeDBError(fkDel, "").(*AppError).Message; !strings.Contains(got, "tidak bisa dihapus") {
		t.Errorf("FK hapus: %q", got)
	}
	nn := &pgconn.PgError{Code: "23502", ColumnName: "slot_capacity"}
	if got := NormalizeDBError(nn, "").(*AppError).Message; got != "Slot capacity wajib diisi." {
		t.Errorf("not null: %q", got)
	}
}

func TestNormalizeDBError_AppErrorDiteruskan(t *testing.T) {
	asli := NewForbidden("Tidak boleh.")
	if got := NormalizeDBError(fmt.Errorf("bungkus: %w", asli), "t"); got != asli {
		t.Errorf("AppError harus diteruskan apa adanya, dapat %#v", got)
	}
}
