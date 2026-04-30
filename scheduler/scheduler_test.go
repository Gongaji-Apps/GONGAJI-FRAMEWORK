package scheduler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recordingLogger captures log calls for assertions.
type recordingLogger struct {
	mu     sync.Mutex
	infos  []string
	errors []string
}

func (l *recordingLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infos = append(l.infos, msg)
}

func (l *recordingLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, msg)
}

func (l *recordingLogger) errorMsgs() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.errors))
	copy(out, l.errors)
	return out
}

// ---------- Register ----------

func TestRegister_RequiresFields(t *testing.T) {
	s := New(Config{})
	cases := []Job{
		{Schedule: "* * * * *", Handler: func(context.Context) error { return nil }}, // missing name
		{Name: "j", Handler: func(context.Context) error { return nil }},              // missing schedule
		{Name: "j", Schedule: "* * * * *"},                                            // missing handler
	}
	for i, j := range cases {
		if err := s.Register(j); err == nil {
			t.Errorf("case %d: expected error, got nil", i)
		}
	}
}

func TestRegister_InvalidSchedule(t *testing.T) {
	s := New(Config{})
	err := s.Register(Job{
		Name:     "bad",
		Schedule: "not a cron expression",
		Handler:  func(context.Context) error { return nil },
	})
	if err == nil {
		t.Fatal("expected error for invalid schedule")
	}
}

func TestRegister_AfterStartFails(t *testing.T) {
	s := New(Config{})
	if err := s.Register(Job{
		Name: "j", Schedule: "* * * * *",
		Handler: func(context.Context) error { return nil },
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { <-s.Stop().Done() }()

	if err := s.Register(Job{
		Name: "k", Schedule: "* * * * *",
		Handler: func(context.Context) error { return nil },
	}); err == nil {
		t.Fatal("expected error registering after Start")
	}
}

// ---------- Start lifecycle ----------

func TestStart_DoubleStartFails(t *testing.T) {
	s := New(Config{})
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { <-s.Stop().Done() }()
	if err := s.Start(context.Background()); err == nil {
		t.Fatal("expected error on double Start")
	}
}

func TestStart_AfterStopFails(t *testing.T) {
	s := New(Config{})
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	<-s.Stop().Done()
	if err := s.Start(context.Background()); err == nil {
		t.Fatal("expected error after Stop")
	}
}

// ---------- Stop semantics ----------

func TestStop_BeforeStartReturnsImmediately(t *testing.T) {
	s := New(Config{})
	done := s.Stop().Done()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop on never-started scheduler should return done context")
	}
}

func TestStop_CancelsBaseContext(t *testing.T) {
	var observed context.Context
	var wg sync.WaitGroup
	wg.Add(1)

	s := New(Config{WithSeconds: true})
	if err := s.Register(Job{
		Name:     "tick",
		Schedule: "*/1 * * * * *", // every second
		Handler: func(ctx context.Context) error {
			observed = ctx
			wg.Done()
			<-ctx.Done() // hold until the scheduler stops
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Wait for the handler to start
	wg.Wait()
	if observed.Err() != nil {
		t.Errorf("handler ctx prematurely cancelled: %v", observed.Err())
	}

	stopDone := s.Stop().Done()
	select {
	case <-stopDone:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not complete within 3s")
	}
	if observed.Err() == nil {
		t.Errorf("handler ctx should be cancelled after Stop")
	}
}

// ---------- Handler invocation ----------

func TestHandler_RunsAtSecondsCadence(t *testing.T) {
	var calls int32
	s := New(Config{WithSeconds: true})
	if err := s.Register(Job{
		Name:     "ping",
		Schedule: "*/1 * * * * *",
		Handler: func(context.Context) error {
			atomic.AddInt32(&calls, 1)
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Allow ~2.5s so we should see at least 2 invocations.
	time.Sleep(2500 * time.Millisecond)
	<-s.Stop().Done()

	if got := atomic.LoadInt32(&calls); got < 2 {
		t.Errorf("expected at least 2 calls, got %d", got)
	}
}

func TestHandler_PanicIsRecoveredAndLogged(t *testing.T) {
	logger := &recordingLogger{}
	s := New(Config{WithSeconds: true, Logger: logger})

	if err := s.Register(Job{
		Name:     "boom",
		Schedule: "*/1 * * * * *",
		Handler: func(context.Context) error {
			panic("kaboom")
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1500 * time.Millisecond)
	<-s.Stop().Done()

	found := false
	for _, m := range logger.errorMsgs() {
		if m == "scheduler: job panic" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected panic log, got %v", logger.errorMsgs())
	}
}

func TestHandler_ErrorIsLogged(t *testing.T) {
	logger := &recordingLogger{}
	s := New(Config{WithSeconds: true, Logger: logger})

	if err := s.Register(Job{
		Name:     "bad",
		Schedule: "*/1 * * * * *",
		Handler: func(context.Context) error {
			return errors.New("oops")
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1500 * time.Millisecond)
	<-s.Stop().Done()

	found := false
	for _, m := range logger.errorMsgs() {
		if m == "scheduler: job error" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error log, got %v", logger.errorMsgs())
	}
}

// ---------- StandardCron field count ----------

func TestRegister_StandardCronAcceptsFiveFields(t *testing.T) {
	s := New(Config{}) // no seconds
	if err := s.Register(Job{
		Name:     "ok",
		Schedule: "*/5 * * * *",
		Handler:  func(context.Context) error { return nil },
	}); err != nil {
		t.Errorf("expected 5-field cron to validate, got %v", err)
	}
}

func TestRegister_WithSecondsAcceptsSixFields(t *testing.T) {
	s := New(Config{WithSeconds: true})
	if err := s.Register(Job{
		Name:     "ok",
		Schedule: "*/10 * * * * *",
		Handler:  func(context.Context) error { return nil },
	}); err != nil {
		t.Errorf("expected 6-field cron to validate, got %v", err)
	}
}
