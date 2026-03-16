package recorder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/agorator/shadowgo/internal/config"
)

// DefaultWebcamDevice is the typical v4l2 device path on Linux.
const DefaultWebcamDevice = "/dev/video0"

// WebcamRecorder captures webcam via Video4Linux2 using ffmpeg.
type WebcamRecorder struct {
	cfg     *config.Config
	device  string
	cmd     *exec.Cmd
	mu      sync.RWMutex
	status  Status
	err     error
}

// NewWebcamRecorder creates a new WebcamRecorder.
func NewWebcamRecorder(cfg *config.Config, device string) *WebcamRecorder {
	if device == "" {
		device = DefaultWebcamDevice
	}
	return &WebcamRecorder{
		cfg:    cfg,
		device: device,
		status: StatusStopped,
	}
}

// Start begins webcam capture via ffmpeg v4l2 input.
func (w *WebcamRecorder) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.status == StatusRunning {
		return fmt.Errorf("recorder already running")
	}

	if _, err := os.Stat(w.device); os.IsNotExist(err) {
		w.status = StatusError
		w.err = fmt.Errorf("webcam device %s not found", w.device)
		return w.err
	}

	if err := os.MkdirAll(w.cfg.OutputDir, 0755); err != nil {
		w.status = StatusError
		w.err = err
		return err
	}

	outputPath := filepath.Join(w.cfg.OutputDir,
		time.Now().Format("webcam_20060102_150405")+".mp4")

	args := []string{
		"-y",
		"-f", "v4l2",
		"-i", w.device,
		"-c:v", "libx264",
		"-crf", fmt.Sprintf("%d", w.cfg.VideoQuality),
		"-r", fmt.Sprintf("%d", w.cfg.WebcamFPS),
		"-pix_fmt", "yuv420p",
		outputPath,
	}

	w.cmd = exec.CommandContext(ctx, "ffmpeg", args...)
	w.cmd.Stdout = nil
	w.cmd.Stderr = nil

	if err := w.cmd.Start(); err != nil {
		w.status = StatusError
		w.err = err
		return err
	}

	w.status = StatusRunning
	w.err = nil
	return nil
}

// Stop terminates the ffmpeg process.
func (w *WebcamRecorder) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.status != StatusRunning || w.cmd == nil {
		w.status = StatusStopped
		return nil
	}

	if w.cmd.Process != nil {
		_ = w.cmd.Process.Signal(os.Interrupt)
	}

	done := make(chan error, 1)
	go func() {
		done <- w.cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = w.cmd.Process.Kill()
		w.status = StatusStopped
		return ctx.Err()
	case err := <-done:
		w.status = StatusStopped
		w.cmd = nil
		return err
	}
}

// Status returns the current recorder state.
func (w *WebcamRecorder) Status() Status {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

// Process returns the underlying ffmpeg process for health checks.
func (w *WebcamRecorder) Process() *exec.Cmd {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cmd
}
