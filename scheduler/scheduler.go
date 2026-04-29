// Package scheduler runs cron-style background jobs.
//
// It is a thin wrapper over github.com/robfig/cron/v3 that adds:
//   - context propagation to handlers (cancelled when the scheduler is stopped
//     or when the parent context passed to Start is cancelled),
//   - panic recovery so a misbehaving job cannot take down the scheduler,
//   - skip-if-still-running semantics (a long-running job will not overlap
//     with itself; the next tick is skipped),
//   - structured Info/Error logging via an injectable Logger.
//
// Example:
//
//	s := scheduler.New(scheduler.Config{Logger: lg})
//	s.Register(scheduler.Job{
//	    Name:     "settle-orders",
//	    Schedule: "0 */15 * * * *", // every 15 minutes (with seconds)
//	    Handler:  settleOrders,
//	})
//	if err := s.Start(ctx); err != nil { ... }
//	defer func() { <-s.Stop().Done() }()
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Logger is an optional sink for scheduler events.
type Logger interface {
	Info(msg string, fields ...any)
	Error(msg string, fields ...any)
}

// Config configures the Scheduler.
type Config struct {
	// Location is the timezone used to interpret cron expressions.
	// Default: time.UTC.
	Location *time.Location

	// Logger receives lifecycle and per-job events. Nil disables logging.
	Logger Logger

	// WithSeconds extends cron expressions with a leading seconds field.
	// Standard 5-field cron is used when false (the default).
	WithSeconds bool

	// AllowOverlap disables the default skip-if-still-running behavior.
	// When true, multiple invocations of the same job may run concurrently
	// if a previous run hasn't finished. Default: false (skip).
	AllowOverlap bool
}

// Job describes a scheduled handler.
type Job struct {
	Name     string
	Schedule string                                 // cron expression
	Handler  func(ctx context.Context) error
}

// Scheduler is a registry + runner for cron jobs.
type Scheduler struct {
	cfg     Config
	cron    *cron.Cron
	parser  cron.ScheduleParser

	mu       sync.Mutex
	jobs     []Job
	started  bool
	stopped  bool
	baseCtx  context.Context
	cancel   context.CancelFunc
}

// New constructs a Scheduler.
func New(cfg Config) *Scheduler {
	loc := cfg.Location
	if loc == nil {
		loc = time.UTC
	}

	opts := []cron.Option{cron.WithLocation(loc)}
	if cfg.WithSeconds {
		opts = append(opts, cron.WithSeconds())
	}
	if !cfg.AllowOverlap {
		opts = append(opts, cron.WithChain(cron.SkipIfStillRunning(cronLoggerAdapter{cfg.Logger})))
	}
	if cfg.Logger != nil {
		opts = append(opts, cron.WithLogger(cronLoggerAdapter{cfg.Logger}))
	}

	parser := cron.NewParser(stdParserOptions(cfg.WithSeconds))

	return &Scheduler{
		cfg:    cfg,
		cron:   cron.New(opts...),
		parser: parser,
	}
}

func stdParserOptions(withSeconds bool) cron.ParseOption {
	if withSeconds {
		return cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor
	}
	return cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor
}

// Register adds a job. Must be called before Start. The cron expression is
// validated immediately; an invalid expression returns an error.
func (s *Scheduler) Register(job Job) error {
	if job.Name == "" {
		return errors.New("scheduler: job Name is required")
	}
	if job.Schedule == "" {
		return errors.New("scheduler: job Schedule is required")
	}
	if job.Handler == nil {
		return errors.New("scheduler: job Handler is required")
	}
	if _, err := s.parser.Parse(job.Schedule); err != nil {
		return fmt.Errorf("scheduler: invalid cron expression %q for job %q: %w", job.Schedule, job.Name, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return errors.New("scheduler: cannot Register after Start")
	}
	s.jobs = append(s.jobs, job)
	return nil
}

// Start begins scheduling. Returns immediately; jobs run in cron's own
// goroutines. Pass a cancellable parent context to propagate shutdown into
// handlers.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return errors.New("scheduler: already started")
	}
	if s.stopped {
		s.mu.Unlock()
		return errors.New("scheduler: cannot restart a stopped scheduler")
	}

	s.baseCtx, s.cancel = context.WithCancel(ctx)

	for _, job := range s.jobs {
		job := job // capture for closure
		if _, err := s.cron.AddFunc(job.Schedule, s.wrapHandler(job)); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("scheduler: schedule %q: %w", job.Name, err)
		}
	}
	s.started = true
	s.mu.Unlock()

	s.cron.Start()
	return nil
}

// Stop signals the scheduler to stop dispatching new runs and returns a
// context that completes when all running jobs have finished. Block on
// `<-s.Stop().Done()` if you need a graceful shutdown wait.
//
// Calling Stop cancels the base context, so handlers observing ctx.Done()
// will be notified and can exit promptly.
func (s *Scheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		// Return a pre-cancelled context.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}
	s.stopped = true

	if s.cancel != nil {
		s.cancel()
	}
	if !s.started {
		// Nothing to drain.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}
	return s.cron.Stop()
}

// wrapHandler wraps the user handler with panic recovery + ctx propagation
// + lifecycle logging.
func (s *Scheduler) wrapHandler(job Job) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				s.logError("scheduler: job panic", "job", job.Name, "panic", r)
			}
		}()

		s.logInfo("scheduler: job start", "job", job.Name)
		start := time.Now()

		err := job.Handler(s.baseCtx)

		dur := time.Since(start)
		if err != nil {
			s.logError("scheduler: job error", "job", job.Name, "duration", dur, "err", err)
			return
		}
		s.logInfo("scheduler: job done", "job", job.Name, "duration", dur)
	}
}

func (s *Scheduler) logInfo(msg string, fields ...any) {
	if s.cfg.Logger != nil {
		s.cfg.Logger.Info(msg, fields...)
	}
}

func (s *Scheduler) logError(msg string, fields ...any) {
	if s.cfg.Logger != nil {
		s.cfg.Logger.Error(msg, fields...)
	}
}

// cronLoggerAdapter bridges a Logger into the cron.Logger interface so that
// internal cron events (skipped runs, parse warnings) flow through the same
// sink.
type cronLoggerAdapter struct {
	logger Logger
}

func (a cronLoggerAdapter) Info(msg string, keysAndValues ...interface{}) {
	if a.logger == nil {
		return
	}
	a.logger.Info(msg, keysAndValues...)
}

func (a cronLoggerAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
	if a.logger == nil {
		return
	}
	if err != nil {
		keysAndValues = append(keysAndValues, "err", err)
	}
	a.logger.Error(msg, keysAndValues...)
}
