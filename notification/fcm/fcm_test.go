package fcm

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"firebase.google.com/go/v4/messaging"
)

// fakeMessaging implements messagingClient for hermetic testing.
type fakeMessaging struct {
	sendCalls      []*messaging.Message
	multicastCalls []*messaging.MulticastMessage

	sendErr      error
	sendResp     string
	multicastErr error
	multicastFn  func(*messaging.MulticastMessage) *messaging.BatchResponse
}

func (f *fakeMessaging) Send(ctx context.Context, m *messaging.Message) (string, error) {
	f.sendCalls = append(f.sendCalls, m)
	if f.sendErr != nil {
		return "", f.sendErr
	}
	if f.sendResp != "" {
		return f.sendResp, nil
	}
	return "msg-id", nil
}

func (f *fakeMessaging) SendEachForMulticast(ctx context.Context, m *messaging.MulticastMessage) (*messaging.BatchResponse, error) {
	f.multicastCalls = append(f.multicastCalls, m)
	if f.multicastErr != nil {
		return nil, f.multicastErr
	}
	if f.multicastFn != nil {
		return f.multicastFn(m), nil
	}
	// default: all succeed
	resp := &messaging.BatchResponse{
		SuccessCount: len(m.Tokens),
		Responses:    make([]*messaging.SendResponse, len(m.Tokens)),
	}
	for i := range m.Tokens {
		resp.Responses[i] = &messaging.SendResponse{Success: true}
	}
	return resp, nil
}

func newClient() (*Client, *fakeMessaging) {
	f := &fakeMessaging{}
	return &Client{msg: f}, f
}

// ---------- buildData / buildNotification ----------

func TestBuildNotification_EmptyReturnsNil(t *testing.T) {
	if got := buildNotification(Notification{}); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestBuildNotification_TitleBody(t *testing.T) {
	got := buildNotification(Notification{Title: "T", Body: "B"})
	if got == nil || got.Title != "T" || got.Body != "B" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestBuildData_OnlyMetadata(t *testing.T) {
	got := buildData(Notification{Module: "TX", Type: "ORDER", Route: "/r"})
	want := map[string]string{"module": "TX", "type": "ORDER", "route": "/r"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildData_CustomDataMergedAndOverridesMetadata(t *testing.T) {
	got := buildData(Notification{
		Module: "TX",
		Data:   map[string]string{"module": "OVERRIDE", "extra": "1"},
	})
	if got["module"] != "OVERRIDE" {
		t.Errorf("module = %q, want OVERRIDE (custom Data wins)", got["module"])
	}
	if got["extra"] != "1" {
		t.Errorf("extra missing")
	}
}

func TestBuildData_EmptyReturnsNil(t *testing.T) {
	if got := buildData(Notification{}); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// ---------- chunkTokens ----------

func TestChunkTokens(t *testing.T) {
	cases := []struct {
		in   []string
		size int
		want [][]string
	}{
		{nil, 5, nil},
		{[]string{"a"}, 5, [][]string{{"a"}}},
		{[]string{"a", "b", "c"}, 2, [][]string{{"a", "b"}, {"c"}}},
		{[]string{"a", "b", "c", "d"}, 2, [][]string{{"a", "b"}, {"c", "d"}}},
		{[]string{"a"}, 0, [][]string{{"a"}}}, // size <= 0 falls back to default (still 1 chunk for 1 token)
	}
	for _, tc := range cases {
		got := chunkTokens(tc.in, tc.size)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("chunkTokens(%v, %d) = %v, want %v", tc.in, tc.size, got, tc.want)
		}
	}
}

// ---------- Send ----------

func TestSend_RequiresToken(t *testing.T) {
	c, _ := newClient()
	if _, err := c.Send(context.Background(), "", Notification{Title: "x"}); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestSend_BuildsMessage(t *testing.T) {
	c, fake := newClient()
	fake.sendResp = "id-123"

	id, err := c.Send(context.Background(), "tok-1", Notification{
		Title: "Pesanan", Body: "siap", Module: "TX", Route: "/r",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "id-123" {
		t.Errorf("id = %q", id)
	}
	if len(fake.sendCalls) != 1 {
		t.Fatalf("expected 1 Send call, got %d", len(fake.sendCalls))
	}
	msg := fake.sendCalls[0]
	if msg.Token != "tok-1" {
		t.Errorf("Token = %q", msg.Token)
	}
	if msg.Notification == nil || msg.Notification.Title != "Pesanan" {
		t.Errorf("Notification = %+v", msg.Notification)
	}
	if msg.Data["module"] != "TX" || msg.Data["route"] != "/r" {
		t.Errorf("Data = %v", msg.Data)
	}
	if msg.Topic != "" {
		t.Errorf("Topic should be empty, got %q", msg.Topic)
	}
}

func TestSend_PropagatesError(t *testing.T) {
	c, fake := newClient()
	fake.sendErr = errors.New("upstream down")
	_, err := c.Send(context.Background(), "tok", Notification{Title: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------- SendToTopic ----------

func TestSendToTopic_RequiresTopic(t *testing.T) {
	c, _ := newClient()
	if _, err := c.SendToTopic(context.Background(), "", Notification{Title: "x"}); err == nil {
		t.Fatal("expected error for empty topic")
	}
}

func TestSendToTopic_BuildsMessage(t *testing.T) {
	c, fake := newClient()
	if _, err := c.SendToTopic(context.Background(), "news", Notification{Title: "x"}); err != nil {
		t.Fatal(err)
	}
	msg := fake.sendCalls[0]
	if msg.Topic != "news" {
		t.Errorf("Topic = %q", msg.Topic)
	}
	if msg.Token != "" {
		t.Errorf("Token should be empty for topic, got %q", msg.Token)
	}
}

// ---------- SendMulticast ----------

func TestSendMulticast_Empty(t *testing.T) {
	c, _ := newClient()
	if _, err := c.SendMulticast(context.Background(), nil, Notification{}); err == nil {
		t.Fatal("expected error for empty tokens")
	}
}

func TestSendMulticast_AllSuccess(t *testing.T) {
	c, fake := newClient()
	tokens := []string{"a", "b", "c"}

	res, err := c.SendMulticast(context.Background(), tokens, Notification{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.SuccessCount != 3 || res.FailureCount != 0 {
		t.Errorf("counts = (%d,%d), want (3,0)", res.SuccessCount, res.FailureCount)
	}
	if len(fake.multicastCalls) != 1 {
		t.Errorf("expected 1 batch, got %d", len(fake.multicastCalls))
	}
}

func TestSendMulticast_ChunksOverMaxBatch(t *testing.T) {
	c, fake := newClient()
	tokens := make([]string, MaxTokensPerBatch+5)
	for i := range tokens {
		tokens[i] = "tok"
	}
	res, err := c.SendMulticast(context.Background(), tokens, Notification{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.multicastCalls) != 2 {
		t.Errorf("expected 2 batches, got %d", len(fake.multicastCalls))
	}
	if res.SuccessCount != len(tokens) {
		t.Errorf("Success = %d, want %d", res.SuccessCount, len(tokens))
	}
}

func TestSendMulticast_PartialFailureReportsTokens(t *testing.T) {
	c, fake := newClient()
	fake.multicastFn = func(m *messaging.MulticastMessage) *messaging.BatchResponse {
		// fail second token
		return &messaging.BatchResponse{
			SuccessCount: 2,
			FailureCount: 1,
			Responses: []*messaging.SendResponse{
				{Success: true},
				{Success: false, Error: errors.New("invalid token")},
				{Success: true},
			},
		}
	}

	res, err := c.SendMulticast(context.Background(), []string{"a", "b", "c"}, Notification{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.SuccessCount != 2 || res.FailureCount != 1 {
		t.Errorf("counts = (%d,%d)", res.SuccessCount, res.FailureCount)
	}
	if len(res.Errors) != 1 || res.Errors[0].Token != "b" {
		t.Errorf("Errors = %+v", res.Errors)
	}
}
