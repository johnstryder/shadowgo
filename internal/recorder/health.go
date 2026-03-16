package recorder

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// HealthCheckResult holds the result of a health check.
type HealthCheckResult struct {
	Healthy bool
	Reason  string
}

// ProcessHealthChecker checks if an ffmpeg process is alive and writing to disk.
type ProcessHealthChecker struct {
	getProcess func() *os.Process
	getOutput  func() string
	mu         sync.RWMutex
	lastSize   int64
	lastCheck  time.Time
}

// NewProcessHealthChecker creates a health checker for an ffmpeg-based recorder.
func NewProcessHealthChecker(getProcess func() *os.Process, getOutput func() string) *ProcessHealthChecker {
	return &ProcessHealthChecker{
		getProcess: getProcess,
		getOutput:  getOutput,
	}
}

// Check verifies the process is alive and the output file is growing.
func (h *ProcessHealthChecker) Check(ctx context.Context) HealthCheckResult {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.getProcess == nil || h.getOutput == nil {
		return HealthCheckResult{Healthy: false, Reason: "no process or output configured"}
	}

	proc := h.getProcess()
	if proc == nil {
		return HealthCheckResult{Healthy: false, Reason: "process is nil"}
	}

	// Signal(0) checks process existence without sending a signal (Unix convention)
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return HealthCheckResult{Healthy: false, Reason: "process not alive: " + err.Error()}
	}

	outputPath := h.getOutput()
	if outputPath == "" {
		return HealthCheckResult{Healthy: true, Reason: "process alive (no output path)"}
	}

	// Resolve glob patterns - output might use timestamp (e.g. /path/to/screen_*.mp4)
	matches, err := filepath.Glob(outputPath)
	if err != nil || len(matches) == 0 {
		// Try direct stat if not a glob
		if info, err := os.Stat(outputPath); err == nil && !info.IsDir() {
			matches = []string{outputPath}
		}
	}

	if len(matches) == 0 {
		return HealthCheckResult{Healthy: true, Reason: "process alive (output file not yet created)"}
	}

	// Use the most recently modified file
	var latestPath string
	var latestMod time.Time
	for _, p := range matches {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			if info.ModTime().After(latestMod) {
				latestMod = info.ModTime()
				latestPath = p
			}
		}
	}

	if latestPath == "" {
		return HealthCheckResult{Healthy: true, Reason: "process alive"}
	}

	info, err := os.Stat(latestPath)
	if err != nil {
		return HealthCheckResult{Healthy: false, Reason: "cannot stat output: " + err.Error()}
	}

	currentSize := info.Size()
	// If we have a previous size and enough time has passed, verify file is growing
	if h.lastSize > 0 && time.Since(h.lastCheck) > 2*time.Second {
		if currentSize <= h.lastSize {
			return HealthCheckResult{Healthy: false, Reason: "output file not growing"}
		}
	}

	h.lastSize = currentSize
	h.lastCheck = time.Now()
	return HealthCheckResult{Healthy: true, Reason: "process alive, writing to disk"}
}
