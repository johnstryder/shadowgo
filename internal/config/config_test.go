package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_ShouldSetSensibleDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.VideoQuality != 23 {
		t.Errorf("expected VideoQuality 23, got %d", cfg.VideoQuality)
	}
	if cfg.AudioBitrate != "192k" {
		t.Errorf("expected AudioBitrate 192k, got %s", cfg.AudioBitrate)
	}
	if cfg.AudioSource != "pulse" {
		t.Errorf("expected AudioSource pulse, got %s", cfg.AudioSource)
	}
	if cfg.VideoFramerate != 30 {
		t.Errorf("expected VideoFramerate 30, got %d", cfg.VideoFramerate)
	}
	if cfg.ScreenshotDir == "" {
		t.Error("expected ScreenshotDir to be set")
	}
}

func TestDefaultConfig_ShouldUseEnvOverrideForOutputDir(t *testing.T) {
	os.Setenv("SHADOWGO_OUTPUT_DIR", "/custom/path")
	defer os.Unsetenv("SHADOWGO_OUTPUT_DIR")

	cfg := DefaultConfig()

	if cfg.OutputDir != "/custom/path" {
		t.Errorf("expected OutputDir /custom/path, got %s", cfg.OutputDir)
	}
}

func TestScreenOutputPath_ShouldJoinDirAndFilename(t *testing.T) {
	cfg := &Config{
		OutputDir:      "/videos",
		ScreenFilename: "screen.mp4",
	}

	path := cfg.ScreenOutputPath()
	expected := filepath.Join("/videos", "screen.mp4")

	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestScreenshotOutputPath_ShouldUseScreenshotDirWhenSet(t *testing.T) {
	cfg := &Config{
		OutputDir:         "/videos",
		ScreenshotDir:     "/pictures",
		ScreenshotFilename: "shot.png",
	}

	path := cfg.ScreenshotOutputPath()
	expected := filepath.Join("/pictures", "shot.png")

	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestScreenshotOutputPath_ShouldFallbackToOutputDirWhenScreenshotDirEmpty(t *testing.T) {
	cfg := &Config{
		OutputDir:          "/videos",
		ScreenshotDir:      "",
		ScreenshotFilename: "shot.png",
	}

	path := cfg.ScreenshotOutputPath()
	expected := filepath.Join("/videos", "shot.png")

	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}
