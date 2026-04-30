// Package cloudtask wraps Google Cloud Tasks for HTTP-target task
// creation and deletion.
//
// Example:
//
//	c, err := cloudtask.New(ctx, cloudtask.Config{
//	    ProjectID:  "my-project",
//	    LocationID: "asia-southeast2",
//	})
//	defer c.Close()
//
//	task, err := c.Create(ctx, cloudtask.TaskRequest{
//	    QueueID:  "default",
//	    URL:      "https://my-service.example.com/jobs/run",
//	    Payload:  payload,
//	    Headers:  map[string]string{"Content-Type": "application/json"},
//	    Schedule: &runAt,
//	})
package cloudtask

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Config configures the Cloud Tasks Client.
type Config struct {
	ProjectID  string
	LocationID string // e.g. "asia-southeast2"
}

// Client wraps the Cloud Tasks API client with helpers.
type Client struct {
	raw      *cloudtasks.Client
	project  string
	location string
}

// New constructs a Client. Pass option.ClientOption values for credentials,
// custom endpoints, etc. A nil opts slice uses Application Default Credentials.
func New(ctx context.Context, cfg Config, opts ...option.ClientOption) (*Client, error) {
	if cfg.ProjectID == "" {
		return nil, errors.New("cloudtask: ProjectID is required")
	}
	if cfg.LocationID == "" {
		return nil, errors.New("cloudtask: LocationID is required")
	}
	raw, err := cloudtasks.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("cloudtask: new client: %w", err)
	}
	return &Client{
		raw:      raw,
		project:  cfg.ProjectID,
		location: cfg.LocationID,
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *Client) Close() error {
	if c == nil || c.raw == nil {
		return nil
	}
	return c.raw.Close()
}

// QueuePath returns the fully qualified queue resource name.
func (c *Client) QueuePath(queueID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/queues/%s", c.project, c.location, queueID)
}

// TaskRequest describes an HTTP-target Cloud Task to enqueue.
type TaskRequest struct {
	QueueID  string            // required
	URL      string            // required (target HTTP endpoint)
	Method   string            // default POST
	Payload  []byte            // request body, optional
	Headers  map[string]string // request headers, optional
	Schedule *time.Time        // optional run-at time; nil = ASAP
	Name     string            // optional task ID (for de-dup); full or short name
}

// Create enqueues a new task.
func (c *Client) Create(ctx context.Context, req TaskRequest) (*cloudtaskspb.Task, error) {
	if req.QueueID == "" {
		return nil, errors.New("cloudtask: QueueID is required")
	}
	if req.URL == "" {
		return nil, errors.New("cloudtask: URL is required")
	}

	task, err := c.buildTask(req)
	if err != nil {
		return nil, err
	}

	out, err := c.raw.CreateTask(ctx, &cloudtaskspb.CreateTaskRequest{
		Parent: c.QueuePath(req.QueueID),
		Task:   task,
	})
	if err != nil {
		return nil, fmt.Errorf("cloudtask: create task: %w", err)
	}
	return out, nil
}

// Delete removes a task by its full resource name (as returned by Create).
func (c *Client) Delete(ctx context.Context, taskName string) error {
	if taskName == "" {
		return errors.New("cloudtask: task name is required")
	}
	if err := c.raw.DeleteTask(ctx, &cloudtaskspb.DeleteTaskRequest{Name: taskName}); err != nil {
		return fmt.Errorf("cloudtask: delete task: %w", err)
	}
	return nil
}

// buildTask is the pure (no-network) request-building stage. It is exposed
// for unit testing — tests assert on the constructed protobuf.
func (c *Client) buildTask(req TaskRequest) (*cloudtaskspb.Task, error) {
	method, err := parseMethod(req.Method)
	if err != nil {
		return nil, err
	}

	httpReq := &cloudtaskspb.HttpRequest{
		Url:        req.URL,
		HttpMethod: method,
		Headers:    req.Headers,
		Body:       req.Payload,
	}

	task := &cloudtaskspb.Task{
		MessageType: &cloudtaskspb.Task_HttpRequest{HttpRequest: httpReq},
	}

	if req.Schedule != nil {
		task.ScheduleTime = timestamppb.New(*req.Schedule)
	}
	if req.Name != "" {
		task.Name = c.resolveTaskName(req.QueueID, req.Name)
	}
	return task, nil
}

// resolveTaskName accepts either a fully qualified task resource name or a
// short ID and normalizes it to the full path.
func (c *Client) resolveTaskName(queueID, name string) string {
	if strings.HasPrefix(name, "projects/") {
		return name
	}
	return fmt.Sprintf("%s/tasks/%s", c.QueuePath(queueID), name)
}

func parseMethod(m string) (cloudtaskspb.HttpMethod, error) {
	if m == "" {
		return cloudtaskspb.HttpMethod_POST, nil
	}
	switch strings.ToUpper(m) {
	case "GET":
		return cloudtaskspb.HttpMethod_GET, nil
	case "POST":
		return cloudtaskspb.HttpMethod_POST, nil
	case "PUT":
		return cloudtaskspb.HttpMethod_PUT, nil
	case "DELETE":
		return cloudtaskspb.HttpMethod_DELETE, nil
	case "PATCH":
		return cloudtaskspb.HttpMethod_PATCH, nil
	case "HEAD":
		return cloudtaskspb.HttpMethod_HEAD, nil
	case "OPTIONS":
		return cloudtaskspb.HttpMethod_OPTIONS, nil
	default:
		return 0, fmt.Errorf("cloudtask: unsupported HTTP method %q", m)
	}
}
