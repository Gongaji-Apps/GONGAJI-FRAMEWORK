package contextx

import (
	"context"
	"testing"
)

func TestPrivileged(t *testing.T) {
	ctx := WithSubjectUUID(WithRoleCode(context.Background(), "USER"), "u-1")

	privileged := Privileged(ctx)

	if got := GetRoleCode(privileged); got != "" {
		t.Errorf("role harus kosong, dapat %q", got)
	}

	if got := GetSubjectUUID(privileged); got != "u-1" {
		t.Errorf("identitas subjek harus tetap terbawa, dapat %q", got)
	}

	if got := GetRoleCode(ctx); got != "USER" {
		t.Errorf("ctx asal tidak boleh berubah, dapat %q", got)
	}
}
