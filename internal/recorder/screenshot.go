package recorder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/agorator/shadowgo/internal/config"
)

// CaptureScreenshot captures a screenshot using grim.
// If region is nil, captures full screen. Otherwise captures the specified region.
// Returns the path to the saved image.
func CaptureScreenshot(ctx context.Context, cfg *config.Config, region *Region) (string, error) {
	_, grimAvailable := DetectRegionTools()
	if !grimAvailable {
		return "", fmt.Errorf("grim not found in PATH (required for screenshots)")
	}

	outputDir := cfg.ScreenshotDir
	if outputDir == "" {
		outputDir = cfg.OutputDir
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	outputPath := filepath.Join(outputDir,
		time.Now().Format("screenshot_20060102_150405")+".png")

	args := []string{}
	if region != nil && region.W > 0 && region.H > 0 {
		// grim -g "x,y widthxheight"
		geometry := fmt.Sprintf("%d,%d %dx%d", region.X, region.Y, region.W, region.H)
		args = append(args, "-g", geometry)
	}
	args = append(args, outputPath)

	cmd := exec.CommandContext(ctx, "grim", args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("grim failed: %w", err)
	}

	return outputPath, nil
}
