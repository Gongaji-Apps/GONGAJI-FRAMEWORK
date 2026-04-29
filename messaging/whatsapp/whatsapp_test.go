package whatsapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	frameworkErr "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/httputil"
)

// noRetry disables httputil's default 3-attempt retry inside Client. We can't
// inject this directly through whatsapp.Config, so tests construct a Client
// with httputil clients that wrap our test server and do a single attempt.
func newTestClient(t *testing.T, v1, v2 *httptest.Server, sessionsV1 []SessionV1, sessionsV2 []SessionV2) *Client {
	t.Helper()
	c := &Client{
		cfg: Config{
			SessionsV1: sessionsV1,
			SessionsV2: sessionsV2,
		},
	}
	if v1 != nil {
		c.httpV1 = httputil.New(httputil.Config{
			BaseURL: v1.URL,
			Timeout: 2 * time.Second,
			Retry:   &httputil.RetryConfig{MaxAttempts: 1},
		})
	}
	if v2 != nil {
		c.httpV2 = httputil.New(httputil.Config{
			BaseURL: v2.URL,
			Timeout: 2 * time.Second,
			Headers: map[string]string{"Api_key": "test-key"},
			Retry:   &httputil.RetryConfig{MaxAttempts: 1},
		})
	}
	return c
}

// ---------- SendMessage ----------

func TestSendMessage_V1Success(t *testing.T) {
	var calls int32
	var session, receiver, message string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.URL.Path != "/message/send" {
			t.Errorf("path = %q, want /message/send", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		_ = r.ParseForm()
		session = r.FormValue("session")
		receiver = r.FormValue("receiver")
		message = r.FormValue("message")
		if r.FormValue("group") != "false" {
			t.Errorf("group = %q, want false", r.FormValue("group"))
		}
		_ = json.NewEncoder(w).Encode(v1Response{StatusCode: 200, Status: true, Message: "ok"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil, []SessionV1{"GONGAJI"}, nil)
	if err := c.SendMessage(context.Background(), "+628123", "halo"); err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
	if session != "GONGAJI" || receiver != "+628123" || message != "halo" {
		t.Errorf("unexpected payload: session=%q receiver=%q message=%q", session, receiver, message)
	}
}

func TestSendMessage_V1FallbackToSecondSession(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		_ = r.ParseForm()
		if n == 1 && r.FormValue("session") == "S1" {
			_ = json.NewEncoder(w).Encode(v1Response{StatusCode: 500, Status: false, Message: "session offline"})
			return
		}
		_ = json.NewEncoder(w).Encode(v1Response{StatusCode: 200, Status: true})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil, []SessionV1{"S1", "S2"}, nil)
	if err := c.SendMessage(context.Background(), "x", "y"); err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected 2 calls (fallback), got %d", got)
	}
}

func TestSendMessage_V1FallthroughToV2(t *testing.T) {
	var v1Calls, v2Calls int32
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&v1Calls, 1)
		_ = json.NewEncoder(w).Encode(v1Response{StatusCode: 500, Status: false})
	}))
	defer v1.Close()

	v2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&v2Calls, 1)
		if got := r.Header.Get("Api_key"); got != "test-key" {
			t.Errorf("Api_key = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["dev_code"] != "DEV1" || payload["app_code"] != "APP1" {
			t.Errorf("payload = %+v", payload)
		}
		_ = json.NewEncoder(w).Encode(v2Response{Success: true})
	}))
	defer v2.Close()

	c := newTestClient(t, v1, v2,
		[]SessionV1{"S1"},
		[]SessionV2{{DevCode: "DEV1", AppCode: "APP1"}},
	)

	if err := c.SendMessage(context.Background(), "x", "y"); err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if atomic.LoadInt32(&v1Calls) != 1 {
		t.Errorf("expected 1 V1 call, got %d", v1Calls)
	}
	if atomic.LoadInt32(&v2Calls) != 1 {
		t.Errorf("expected 1 V2 call, got %d", v2Calls)
	}
}

func TestSendMessage_AllFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(v1Response{StatusCode: 500, Status: false, Message: "down"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil, []SessionV1{"S1", "S2"}, nil)
	err := c.SendMessage(context.Background(), "x", "y")
	if err == nil {
		t.Fatal("expected error when all sessions fail")
	}
	appErr, ok := err.(*frameworkErr.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.Code != frameworkErr.InternalServerError {
		t.Errorf("Code = %q, want %q", appErr.Code, frameworkErr.InternalServerError)
	}
}

func TestSendMessage_NoSessionsConfigured(t *testing.T) {
	c := New(Config{})
	err := c.SendMessage(context.Background(), "x", "y")
	if err == nil {
		t.Fatal("expected error when no sessions")
	}
}

func TestSendMessage_BadRequestDoesNotMutateGlobals(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(v1Response{StatusCode: 400, Status: false, Message: "nomor invalid"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil, []SessionV1{"S1"}, nil)
	err := c.SendMessage(context.Background(), "x", "y")
	if err == nil {
		t.Fatal("expected error")
	}
	// Repeat with a different message to verify no global state was mutated.
	c2 := newTestClient(t, srv, nil, []SessionV1{"S1"}, nil)
	if err2 := c2.SendMessage(context.Background(), "x", "y"); err2 == nil {
		t.Fatal("expected error on second call")
	}
}

// ---------- CreateGroup ----------

func TestCreateGroup_Success_AppendsAllArrayValues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/group/create" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = r.ParseForm()
		if got := r.Form["group_admin[]"]; len(got) != 2 || got[0] != "a1" || got[1] != "a2" {
			t.Errorf("group_admin[] = %v, want [a1 a2]", got)
		}
		if got := r.Form["group_participant[]"]; len(got) != 3 {
			t.Errorf("group_participant[] = %v, want 3 entries", got)
		}
		resp := v1GroupCreateResponse{StatusCode: 200, Status: true}
		resp.Data.ID = "g123"
		resp.Data.InviteCode = "inv-xyz"
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil, []SessionV1{"S1"}, nil)
	id, code, err := c.CreateGroup(
		context.Background(),
		"My Group", "desc",
		[]string{"a1", "a2"},
		[]string{"m1", "m2", "m3"},
	)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if id != "g123" || code != "inv-xyz" {
		t.Errorf("got id=%q code=%q", id, code)
	}
}

func TestCreateGroup_NoV1Config(t *testing.T) {
	c := New(Config{})
	_, _, err := c.CreateGroup(context.Background(), "x", "y", nil, nil)
	if err == nil {
		t.Fatal("expected error without V1 config")
	}
}

// ---------- AddGroupMember ----------

func TestAddGroupMember_AppendsAllMembers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/group/add-member" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = r.ParseForm()
		if got := r.Form["group_member[]"]; len(got) != 3 || got[2] != "m3" {
			t.Errorf("group_member[] = %v, want 3 entries ending m3", got)
		}
		if r.FormValue("group_id") != "g1" {
			t.Errorf("group_id = %q", r.FormValue("group_id"))
		}
		if r.FormValue("group_role") != "member" {
			t.Errorf("group_role = %q", r.FormValue("group_role"))
		}
		_ = json.NewEncoder(w).Encode(v1Response{StatusCode: 200, Status: true})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil, []SessionV1{"S1"}, nil)
	if err := c.AddGroupMember(context.Background(), "g1", []string{"m1", "m2", "m3"}, "member"); err != nil {
		t.Fatalf("AddGroupMember: %v", err)
	}
}

// ---------- SendOTP ----------

func TestSendOTP_TemplatesMessage(t *testing.T) {
	var capturedMessage string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		capturedMessage = r.FormValue("message")
		_ = json.NewEncoder(w).Encode(v1Response{StatusCode: 200, Status: true})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, nil, []SessionV1{"S1"}, nil)
	if err := c.SendOTP(context.Background(), "x", "Budi", "123456"); err != nil {
		t.Fatalf("SendOTP: %v", err)
	}
	if !strings.Contains(capturedMessage, "Budi") || !strings.Contains(capturedMessage, "123456") {
		t.Errorf("message missing template variables: %q", capturedMessage)
	}
}

// ---------- Messenger interface compliance ----------

func TestClient_ImplementsMessenger(t *testing.T) {
	var _ Messenger = (*Client)(nil)
}
