package orchestrator

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/agorator/shadowgo/internal/config"
	"github.com/agorator/shadowgo/internal/recorder"
)

// Orchestrator manages multiple capture sources and coordinates their lifecycle.
type Orchestrator struct {
	cfg         *config.Config
	recorders   []recorder.Recorder
	healthCheck time.Duration
	log         *slog.Logger
}

// Option configures the Orchestrator.
type Option func(*Orchestrator)

// WithHealthCheckInterval sets the health check interval.
func WithHealthCheckInterval(d time.Duration) Option {
	return func(o *Orchestrator) {
		o.healthCheck = d
	}
}

// WithLogger sets a custom logger.
func WithLogger(log *slog.Logger) Option {
	return func(o *Orchestrator) {
		o.log = log
	}
}

// New creates a new Orchestrator.
func New(cfg *config.Config, recorders []recorder.Recorder, opts ...Option) *Orchestrator {
	o := &Orchestrator{
		cfg:         cfg,
		recorders:   recorders,
		healthCheck: 5 * time.Second,
		log:         slog.Default(),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Run starts all recorders and blocks until shutdown signal.
func (o *Orchestrator) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start all recorders
	var wg sync.WaitGroup
	for i, rec := range o.recorders {
		wg.Add(1)
		go func(idx int, r recorder.Recorder) {
			defer wg.Done()
			if err := r.Start(ctx); err != nil {
				o.log.Error("recorder failed to start", "index", idx, "error", err)
			}
		}(i, rec)
	}

	// Give recorders time to start
	time.Sleep(500 * time.Millisecond)

	// Health check loop
	healthDone := make(chan struct{})
	go func() {
		defer close(healthDone)
		o.runHealthCheckLoop(ctx)
	}()

	// Wait for shutdown signal
	select {
	case sig := <-sigCh:
		o.log.Info("received signal, shutting down", "signal", sig.String())
	case <-ctx.Done():
		o.log.Info("context cancelled, shutting down")
	}

	cancel()

	// Stop all recorders
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()

	for i, rec := range o.recorders {
		if rec.Status() == recorder.StatusRunning {
			if err := rec.Stop(stopCtx); err != nil {
				o.log.Error("recorder failed to stop gracefully", "index", i, "error", err)
			}
		}
	}

	<-healthDone
	return nil
}

// runHealthCheckLoop periodically verifies recorders are healthy.
func (o *Orchestrator) runHealthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(o.healthCheck)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i, rec := range o.recorders {
				if rec.Status() != recorder.StatusRunning {
					continue
				}

				// Type-assert to get process for health check
				switch r := rec.(type) {
				case *recorder.PipeWireRecorder:
					if proc := r.Process(); proc != nil && proc.Process != nil {
						screenGlob := filepath.Join(o.cfg.OutputDir, "screen_*.mp4")
						checker := recorder.NewProcessHealthChecker(
							func() *os.Process { return proc.Process },
							func() string { return screenGlob },
						)
						result := checker.Check(ctx)
						if !result.Healthy {
							o.log.Warn("screen recorder unhealthy", "index", i, "reason", result.Reason)
						}
					}
				case *recorder.WebcamRecorder:
					if proc := r.Process(); proc != nil && proc.Process != nil {
						webcamGlob := filepath.Join(o.cfg.OutputDir, "webcam_*.mp4")
						checker := recorder.NewProcessHealthChecker(
							func() *os.Process { return proc.Process },
							func() string { return webcamGlob },
						)
						result := checker.Check(ctx)
						if !result.Healthy {
							o.log.Warn("webcam recorder unhealthy", "index", i, "reason", result.Reason)
						}
					}
				}
			}
		}
	}
}
