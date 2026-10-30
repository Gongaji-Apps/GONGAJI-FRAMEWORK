package binding

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/query"
)

type testPreload struct {
	Preload_Category bool
}

type testPayload struct {
	Name             string `form:"name"`
	Active           *bool  `form:"active"`
	Preload_Category bool   `form:"preload_category" gorm:"-"`
}

func (p testPayload) ToPreload() testPreload {
	return testPreload{Preload_Category: p.Preload_Category}
}

func listCtx(rawQuery string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?"+rawQuery, nil)
	return c
}

func TestListQuery_MengisiPayloadDanPagination(t *testing.T) {
	payload, request, err := ListQuery[testPayload, testPreload](listCtx("name=doa&limit=5&pagination=2&preload_category=true"))
	if err != nil {
		t.Fatal(err)
	}
	if payload.Name != "doa" || !payload.Preload_Category {
		t.Errorf("payload = %+v", payload)
	}
	if request.Pagination != (query.Pagination{Limit: 5, Pagination: 2}) {
		t.Errorf("pagination = %+v", request.Pagination)
	}
	if request.Query.Where != nil {
		t.Error("Where harus kosong sebelum FinishList")
	}
}

func TestListQuery_GalatBinding(t *testing.T) {
	if _, _, err := ListQuery[testPayload, testPreload](listCtx("limit=abc")); err == nil {
		t.Error("limit bukan angka harus error")
	}
}

func TestFinishList(t *testing.T) {
	payload, request, err := ListQuery[testPayload, testPreload](listCtx("name=doa&preload_category=true"))
	if err != nil {
		t.Fatal(err)
	}

	payload.Name = "" // kondisi khusus sudah diproses → dikosongkan agar tidak masuk WHERE
	if err := FinishList(&request, payload); err != nil {
		t.Fatal(err)
	}

	where, ok := request.Query.Where.(map[string]any)
	if !ok {
		t.Fatalf("Where harus map, dapat %T", request.Query.Where)
	}
	if _, ada := where["name"]; ada {
		t.Errorf("field kosong tidak boleh masuk WHERE: %v", where)
	}
	if !request.Preload.Preload_Category {
		t.Error("Preload harus terisi dari ToPreload()")
	}
}

func TestFinishWhere_TanpaPreload(t *testing.T) {
	type plain struct {
		Code string `form:"code"`
	}
	payload, request, err := ListQuery[plain, struct{}](listCtx("code=X"))
	if err != nil {
		t.Fatal(err)
	}
	if err := FinishWhere(&request, payload); err != nil {
		t.Fatal(err)
	}
	where := request.Query.Where.(map[string]any)
	if where["code"] != "X" {
		t.Errorf("Where = %v", where)
	}
}
