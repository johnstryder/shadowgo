package config

import (
	"os"
	"path/filepath"
)

// Config holds file paths and quality settings for ShadowGo recording.
type Config struct {
	// Output paths
	OutputDir          string
	ScreenshotDir      string // Defaults to Pictures for screenshots
	ScreenFilename     string
	AudioFilename      string
	WebcamFilename     string
	ScreenshotFilename  string

	// Quality settings
	VideoQuality   int    // CRF for ffmpeg (0-51, lower = better, 23 default)
	AudioBitrate   string // e.g. "192k"
	AudioSource    string // "pulse" or "alsa" - ffmpeg -f input
	VideoFramerate int    // e.g. 30
	WebcamFPS      int    // Webcam frames per second

	// Region selection (slurp/grim)
	UseRegion      bool
	RegionX, RegionY, RegionW, RegionH int

	// LLM / Vision API (OpenAI-compatible, OpenRouter)
	LLMAPIKey   string
	LLMBaseURL  string // e.g. https://openrouter.ai/api/v1
	LLMModel    string // e.g. openai/gpt-4-vision-preview
	LLMPrompt   string // Default prompt for image analysis (marketability)

	// Social platform OAuth (env vars: SHADOWGO_X_CLIENT_ID, etc.)
	ConfigDir string // ~/.config/shadowgo
}

// DefaultConfig returns a config with sensible defaults for Hyprland/Arch.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	outputDir := filepath.Join(home, "Videos", "shadowgo")
	if d := os.Getenv("SHADOWGO_OUTPUT_DIR"); d != "" {
		outputDir = d
	}
	screenshotDir := filepath.Join(home, "Pictures", "shadowgo")
	if d := os.Getenv("SHADOWGO_SCREENSHOT_DIR"); d != "" {
		screenshotDir = d
	}

	// LLM: OpenRouter by default, supports any OpenAI-compatible provider
	llmBaseURL := "https://openrouter.ai/api/v1"
	if u := os.Getenv("SHADOWGO_LLM_BASE_URL"); u != "" {
		llmBaseURL = u
	}
	llmModel := "openai/gpt-4-vision-preview"
	if m := os.Getenv("SHADOWGO_LLM_MODEL"); m != "" {
		llmModel = m
	}
	llmAPIKey := os.Getenv("OPENROUTER_API_KEY")
	if k := os.Getenv("SHADOWGO_API_KEY"); k != "" {
		llmAPIKey = k
	}
	llmPrompt := "Analyze this screenshot for marketability. Consider: visual appeal, clarity, professionalism, composition, and potential use in marketing or advertising. Provide a brief assessment and actionable suggestions."
	if p := os.Getenv("SHADOWGO_LLM_PROMPT"); p != "" {
		llmPrompt = p
	}

	configDir := filepath.Join(home, ".config", "shadowgo")
	if d := os.Getenv("SHADOWGO_CONFIG_DIR"); d != "" {
		configDir = d
	}

	return &Config{
		OutputDir:          outputDir,
		ScreenshotDir:      screenshotDir,
		ScreenFilename:     "screen_%Y%m%d_%H%M%S.mp4",
		AudioFilename:      "audio_%Y%m%d_%H%M%S.m4a",
		WebcamFilename:     "webcam_%Y%m%d_%H%M%S.mp4",
		ScreenshotFilename:  "screenshot_%Y%m%d_%H%M%S.png",
		VideoQuality:   23,
		AudioBitrate:   "192k",
		AudioSource:    "pulse", // or "alsa" for ALSA
		VideoFramerate: 30,
		WebcamFPS:      30,
		UseRegion:      false,
		LLMAPIKey:      llmAPIKey,
		LLMBaseURL:     llmBaseURL,
		LLMModel:       llmModel,
		LLMPrompt:      llmPrompt,
		ConfigDir:      configDir,
	}
}

// SocialPlatformConfig returns OAuth config for a platform (from env vars).
func (c *Config) SocialPlatformConfig(platform string) map[string]string {
	m := make(map[string]string)
	switch platform {
	case "x", "twitter":
		m["client_id"] = os.Getenv("SHADOWGO_X_CLIENT_ID")
		m["client_secret"] = os.Getenv("SHADOWGO_X_CLIENT_SECRET")
		if r := os.Getenv("SHADOWGO_X_REDIRECT_URI"); r != "" {
			m["redirect_uri"] = r
		} else {
			m["redirect_uri"] = "http://127.0.0.1:8080/callback"
		}
	}
	return m
}

// ScreenOutputPath returns the full path for screen recording output.
func (c *Config) ScreenOutputPath() string {
	return filepath.Join(c.OutputDir, c.ScreenFilename)
}

// AudioOutputPath returns the full path for audio recording output.
func (c *Config) AudioOutputPath() string {
	return filepath.Join(c.OutputDir, c.AudioFilename)
}

// WebcamOutputPath returns the full path for webcam recording output.
func (c *Config) WebcamOutputPath() string {
	return filepath.Join(c.OutputDir, c.WebcamFilename)
}

// ScreenshotOutputPath returns the full path for screenshot output.
func (c *Config) ScreenshotOutputPath() string {
	dir := c.ScreenshotDir
	if dir == "" {
		dir = c.OutputDir
	}
	return filepath.Join(dir, c.ScreenshotFilename)
}
