package recorder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/agorator/shadowgo/internal/config"
)

func TestCaptureScreenshot_ShouldReturnErrorWhenGrimUnavailable(t *testing.T) {
	grimAvail, _ := DetectRegionTools()
	if grimAvail {
		t.Skip("grim available - cannot test failure path")
	}

	cfg := config.DefaultConfig()
	_, err := CaptureScreenshot(context.Background(), cfg, nil)
	if err == nil {
		t.Error("expected error when grim not available")
	}
}

func TestCaptureScreenshot_ShouldUseScreenshotDir(t *testing.T) {
	_, grimAvail := DetectRegionTools()
	if !grimAvail {
		t.Skip("grim required for screenshot test")
	}

	dir := t.TempDir()
	cfg := &config.Config{
		OutputDir:     "/tmp/videos",
		ScreenshotDir: dir,
	}

	path, err := CaptureScreenshot(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("CaptureScreenshot: %v", err)
	}

	if filepath.Dir(path) != dir {
		t.Errorf("expected path in %s, got %s", dir, path)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("screenshot file not created: %s", path)
	}
}
