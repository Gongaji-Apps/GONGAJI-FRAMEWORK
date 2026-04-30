package cloudtask

import (
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
)

// newTestClient returns a *Client with no real cloudtasks client attached.
// It is sufficient for testing the pure helpers (QueuePath, buildTask, etc.).
func newTestClient() *Client {
	return &Client{
		project:  "proj-1",
		location: "asia-southeast2",
	}
}

func TestQueuePath(t *testing.T) {
	c := newTestClient()
	got := c.QueuePath("default")
	want := "projects/proj-1/locations/asia-southeast2/queues/default"
	if got != want {
		t.Errorf("QueuePath = %q, want %q", got, want)
	}
}

func TestParseMethod_Defaults(t *testing.T) {
	got, err := parseMethod("")
	if err != nil {
		t.Fatal(err)
	}
	if got != cloudtaskspb.HttpMethod_POST {
		t.Errorf("default = %v, want POST", got)
	}
}

func TestParseMethod_Supported(t *testing.T) {
	cases := map[string]cloudtaskspb.HttpMethod{
		"GET":     cloudtaskspb.HttpMethod_GET,
		"post":    cloudtaskspb.HttpMethod_POST,
		"Put":     cloudtaskspb.HttpMethod_PUT,
		"DELETE":  cloudtaskspb.HttpMethod_DELETE,
		"PATCH":   cloudtaskspb.HttpMethod_PATCH,
		"HEAD":    cloudtaskspb.HttpMethod_HEAD,
		"OPTIONS": cloudtaskspb.HttpMethod_OPTIONS,
	}
	for in, want := range cases {
		got, err := parseMethod(in)
		if err != nil {
			t.Errorf("parseMethod(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseMethod(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseMethod_Unsupported(t *testing.T) {
	if _, err := parseMethod("FOO"); err == nil {
		t.Error("expected error for unsupported method")
	}
}

func TestBuildTask_HappyPath(t *testing.T) {
	c := newTestClient()
	req := TaskRequest{
		QueueID: "default",
		URL:     "https://service/run",
		Method:  "POST",
		Payload: []byte(`{"id":1}`),
		Headers: map[string]string{"Content-Type": "application/json"},
	}
	task, err := c.buildTask(req)
	if err != nil {
		t.Fatal(err)
	}
	httpReq := task.GetHttpRequest()
	if httpReq == nil {
		t.Fatal("HttpRequest nil")
	}
	if httpReq.Url != req.URL {
		t.Errorf("URL = %q", httpReq.Url)
	}
	if httpReq.HttpMethod != cloudtaskspb.HttpMethod_POST {
		t.Errorf("method = %v", httpReq.HttpMethod)
	}
	if string(httpReq.Body) != `{"id":1}` {
		t.Errorf("body = %s", httpReq.Body)
	}
	if httpReq.Headers["Content-Type"] != "application/json" {
		t.Errorf("headers = %v", httpReq.Headers)
	}
	if task.ScheduleTime != nil {
		t.Errorf("schedule should be nil for ASAP, got %v", task.ScheduleTime)
	}
	if task.Name != "" {
		t.Errorf("name should be empty when not set, got %q", task.Name)
	}
}

func TestBuildTask_WithSchedule(t *testing.T) {
	c := newTestClient()
	at := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	task, err := c.buildTask(TaskRequest{
		QueueID:  "q",
		URL:      "https://x",
		Schedule: &at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.ScheduleTime == nil {
		t.Fatal("ScheduleTime should be set")
	}
	if !task.ScheduleTime.AsTime().Equal(at) {
		t.Errorf("ScheduleTime = %v, want %v", task.ScheduleTime.AsTime(), at)
	}
}

func TestBuildTask_ShortNameNormalizedToFullPath(t *testing.T) {
	c := newTestClient()
	task, err := c.buildTask(TaskRequest{
		QueueID: "default",
		URL:     "https://x",
		Name:    "task-abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "projects/proj-1/locations/asia-southeast2/queues/default/tasks/task-abc"
	if task.Name != want {
		t.Errorf("Name = %q, want %q", task.Name, want)
	}
}

func TestBuildTask_FullPathLeftAsIs(t *testing.T) {
	c := newTestClient()
	full := "projects/other/locations/us-central1/queues/q/tasks/t"
	task, err := c.buildTask(TaskRequest{
		QueueID: "default",
		URL:     "https://x",
		Name:    full,
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Name != full {
		t.Errorf("Name = %q, want unchanged %q", task.Name, full)
	}
}

func TestBuildTask_DefaultMethodIsPOST(t *testing.T) {
	c := newTestClient()
	task, err := c.buildTask(TaskRequest{QueueID: "q", URL: "https://x"})
	if err != nil {
		t.Fatal(err)
	}
	if task.GetHttpRequest().HttpMethod != cloudtaskspb.HttpMethod_POST {
		t.Errorf("default method should be POST")
	}
}

func TestBuildTask_RejectsBadMethod(t *testing.T) {
	c := newTestClient()
	if _, err := c.buildTask(TaskRequest{QueueID: "q", URL: "https://x", Method: "WALK"}); err == nil {
		t.Error("expected error for bad method")
	}
}

func TestNew_Validation(t *testing.T) {
	if _, err := New(nil, Config{}); err == nil || !strings.Contains(err.Error(), "ProjectID") {
		t.Errorf("expected ProjectID required error, got %v", err)
	}
	if _, err := New(nil, Config{ProjectID: "p"}); err == nil || !strings.Contains(err.Error(), "LocationID") {
		t.Errorf("expected LocationID required error, got %v", err)
	}
}

func TestClose_NilSafe(t *testing.T) {
	var c *Client
	if err := c.Close(); err != nil {
		t.Errorf("nil client Close should be no-op, got %v", err)
	}
	c2 := &Client{}
	if err := c2.Close(); err != nil {
		t.Errorf("Close with nil raw should be no-op, got %v", err)
	}
}
