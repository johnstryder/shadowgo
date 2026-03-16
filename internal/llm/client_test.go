package llm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agorator/shadowgo/internal/config"
)

func TestAnalyzeImage_ShouldReturnErrorWhenAPIKeyMissing(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLMAPIKey = ""

	client := NewClient(cfg)
	_, err := client.AnalyzeImage(context.Background(), "/nonexistent.png", "test")
	if err == nil {
		t.Error("expected error when API key is missing")
	}
	if err != nil && !strings.Contains(err.Error(), "API key") {
		t.Errorf("expected API key error, got: %v", err)
	}
}

func TestEncodeImageBase64_ShouldDetectMimeType(t *testing.T) {
	// Create a minimal PNG file
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "test.png")
	if err := os.WriteFile(pngPath, []byte{0x89, 0x50, 0x4E, 0x47}, 0644); err != nil {
		t.Fatal(err)
	}

	dataURL, err := encodeImageBase64(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dataURL, "data:image/png;base64,") {
		t.Errorf("expected png mime, got: %s", dataURL[:min(50, len(dataURL))])
	}
}
