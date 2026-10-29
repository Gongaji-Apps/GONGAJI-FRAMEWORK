package query

import (
	"reflect"
	"testing"
)

func TestWheres(t *testing.T) {
	got := Wheres("a = 1", "b IS NULL")
	want := []Where{{Query: "a = 1"}, {Query: "b IS NULL"}}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wheres = %+v, mau %+v", got, want)
	}

	if got := Wheres(); got == nil || len(got) != 0 {
		t.Errorf("tanpa kondisi harus slice kosong (bukan nil), dapat %#v", got)
	}
}

func TestUUIDOrNil(t *testing.T) {
	if got := UUIDOrNil(""); got != nil {
		t.Errorf("kosong harus nil, dapat %#v", got)
	}

	if got := UUIDOrNil("abc"); got != "abc" {
		t.Errorf("terisi harus apa adanya, dapat %#v", got)
	}
}
