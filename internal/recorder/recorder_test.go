package recorder

import (
	"context"
	"testing"

	"github.com/agorator/shadowgo/internal/config"
)

func TestPipeWireRecorder_Status_ShouldBeStoppedInitially(t *testing.T) {
	cfg := config.DefaultConfig()
	rec := NewPipeWireRecorder(cfg, nil)

	if rec.Status() != StatusStopped {
		t.Errorf("expected StatusStopped, got %s", rec.Status())
	}
}

func TestWebcamRecorder_Status_ShouldBeStoppedInitially(t *testing.T) {
	cfg := config.DefaultConfig()
	rec := NewWebcamRecorder(cfg, "")

	if rec.Status() != StatusStopped {
		t.Errorf("expected StatusStopped, got %s", rec.Status())
	}
}

func TestWebcamRecorder_Stop_ShouldBeIdempotentWhenStopped(t *testing.T) {
	cfg := config.DefaultConfig()
	rec := NewWebcamRecorder(cfg, "/dev/nonexistent")

	ctx := context.Background()
	err := rec.Stop(ctx)
	if err != nil {
		t.Errorf("Stop on stopped recorder should not error: %v", err)
	}
}
