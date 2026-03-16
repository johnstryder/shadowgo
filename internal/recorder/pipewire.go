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

// PipeWireRecorder captures screen via PipeWire and audio via PulseAudio/ALSA using ffmpeg.
type PipeWireRecorder struct {
	cfg    *config.Config
	region *Region
	cmd    *exec.Cmd
	mu     sync.RWMutex
	status Status
	err    error
}

// NewPipeWireRecorder creates a new PipeWireRecorder.
func NewPipeWireRecorder(cfg *config.Config, region *Region) *PipeWireRecorder {
	return &PipeWireRecorder{
		cfg:    cfg,
		region: region,
		status: StatusStopped,
	}
}

// Start begins screen and audio capture via ffmpeg.
func (p *PipeWireRecorder) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status == StatusRunning {
		return fmt.Errorf("recorder already running")
	}

	if err := os.MkdirAll(p.cfg.OutputDir, 0755); err != nil {
		p.status = StatusError
		p.err = err
		return err
	}

	// Build output path with timestamp
	outputPath := filepath.Join(p.cfg.OutputDir,
		time.Now().Format("screen_20060102_150405")+".mp4")

	// ffmpeg: PipeWire for screen, PulseAudio or ALSA for audio
	audioSource := p.cfg.AudioSource
	if audioSource == "" {
		audioSource = "pulse"
	}
	args := []string{
		"-y",
		"-f", "pipewire",
		"-i", "pipewire:0",
		"-f", audioSource,
		"-i", "default",
		"-c:v", "libx264",
		"-crf", fmt.Sprintf("%d", p.cfg.VideoQuality),
		"-preset", "fast",
		"-r", fmt.Sprintf("%d", p.cfg.VideoFramerate),
		"-c:a", "aac",
		"-b:a", p.cfg.AudioBitrate,
	}

	// Apply crop filter if region is specified
	if p.region != nil && p.region.W > 0 && p.region.H > 0 {
		cropFilter := fmt.Sprintf("crop=%d:%d:%d:%d", p.region.W, p.region.H, p.region.X, p.region.Y)
		args = append(args, "-vf", cropFilter)
	}

	args = append(args, outputPath)

	p.cmd = exec.CommandContext(ctx, "ffmpeg", args...)
	p.cmd.Stdout = nil
	p.cmd.Stderr = nil

	if err := p.cmd.Start(); err != nil {
		p.status = StatusError
		p.err = err
		return err
	}

	p.status = StatusRunning
	p.err = nil
	return nil
}

// Stop terminates the ffmpeg process.
func (p *PipeWireRecorder) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != StatusRunning || p.cmd == nil {
		p.status = StatusStopped
		return nil
	}

	// Send SIGINT for graceful ffmpeg shutdown (finishes encoding)
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(os.Interrupt)
	}

	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = p.cmd.Process.Kill()
		p.status = StatusStopped
		return ctx.Err()
	case err := <-done:
		p.status = StatusStopped
		p.cmd = nil
		return err
	}
}

// Status returns the current recorder state.
func (p *PipeWireRecorder) Status() Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// Process returns the underlying ffmpeg process for health checks.
func (p *PipeWireRecorder) Process() *exec.Cmd {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cmd
}
