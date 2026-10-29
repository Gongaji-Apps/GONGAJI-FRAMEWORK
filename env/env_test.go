package env

import (
	"strings"
	"testing"
)

func TestRequire(t *testing.T) {
	t.Setenv("ENV_TEST_ADA", "isi")
	t.Setenv("ENV_TEST_SPASI", "   ")
	t.Setenv("ENV_TEST_KOSONG", "")

	if err := Require("ENV_TEST_ADA"); err != nil {
		t.Fatalf("variabel terisi tidak boleh error: %v", err)
	}

	if err := Require(); err != nil {
		t.Fatalf("tanpa kunci tidak boleh error: %v", err)
	}

	err := Require("ENV_TEST_ADA", "ENV_TEST_SPASI", "ENV_TEST_KOSONG", "ENV_TEST_TIDAK_ADA")
	if err == nil {
		t.Fatal("variabel kosong harus error")
	}

	for _, want := range []string{"ENV_TEST_SPASI", "ENV_TEST_KOSONG", "ENV_TEST_TIDAK_ADA"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("pesan harus menyebut %s: %q", want, err.Error())
		}
	}

	if strings.Contains(err.Error(), "ENV_TEST_ADA") {
		t.Errorf("variabel terisi tidak boleh disebut: %q", err.Error())
	}
}

func TestGet(t *testing.T) {
	t.Setenv("ENV_TEST_GET", "  nilai ")
	t.Setenv("ENV_TEST_GET_KOSONG", " ")

	if got := Get("ENV_TEST_GET", "x"); got != "nilai" {
		t.Errorf("Get = %q, mau %q", got, "nilai")
	}

	if got := Get("ENV_TEST_GET_KOSONG", "fallback"); got != "fallback" {
		t.Errorf("kosong harus fallback, dapat %q", got)
	}

	if got := Get("ENV_TEST_GET_TIDAK_ADA", "fallback"); got != "fallback" {
		t.Errorf("tidak ada harus fallback, dapat %q", got)
	}
}

func TestBool(t *testing.T) {
	cases := map[string]bool{"true": true, "TRUE": true, " True ": true, "1": false, "yes": false, "": false, "false": false}

	for value, want := range cases {
		t.Setenv("ENV_TEST_BOOL", value)

		if got := Bool("ENV_TEST_BOOL"); got != want {
			t.Errorf("Bool(%q) = %v, mau %v", value, got, want)
		}
	}
}

func TestInt(t *testing.T) {
	t.Setenv("ENV_TEST_INT", " 36 ")
	t.Setenv("ENV_TEST_INT_RUSAK", "abc")

	if got := Int("ENV_TEST_INT", 24); got != 36 {
		t.Errorf("Int = %d, mau 36", got)
	}

	if got := Int("ENV_TEST_INT_RUSAK", 24); got != 24 {
		t.Errorf("bukan angka harus fallback, dapat %d", got)
	}

	if got := Int("ENV_TEST_INT_TIDAK_ADA", 24); got != 24 {
		t.Errorf("tidak ada harus fallback, dapat %d", got)
	}
}
