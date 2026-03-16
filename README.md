# ShadowGo

Background recording service for **Screen**, **Audio**, and **Webcam** on Linux (Wayland/Hyprland). Designed for Arch Linux with a system-agnostic architecture.

## Features

- **Screenshots** via grim (primary use case) — full screen or region with slurp
- **LLM marketability analysis** — send screenshots to OpenAI-compatible vision APIs (OpenRouter, OpenAI)
- **PipeWire** screen capture + **PulseAudio/ALSA** audio via ffmpeg
- **Webcam** capture via Video4Linux2 (v4l2)
- **Region selection** via slurp (when available)
- Health check loop to verify ffmpeg is alive and writing to disk
- Clean shutdown with SIGINT/SIGTERM handling

## Requirements

- Go 1.22+
- ffmpeg
- PipeWire (screen capture)
- PulseAudio or ALSA (audio)
- Optional: slurp, grim (for region selection)

## Build & Install

**Pre-built binaries** (from [Releases](https://github.com/johnstryder/shadowgo/releases)):

```
https://github.com/johnstryder/shadowgo/releases/latest/download/shadowgo-<os>-<arch>
```

| Platform | Download |
|----------|----------|
| Linux amd64 | `shadowgo-linux-amd64` |
| Linux arm64 | `shadowgo-linux-arm64` |
| macOS amd64 | `shadowgo-darwin-amd64` |
| macOS arm64 (Apple Silicon) | `shadowgo-darwin-arm64` |
| Windows amd64 | `shadowgo-windows-amd64.exe` |

```bash
# Example: Linux
curl -sL https://github.com/johnstryder/shadowgo/releases/latest/download/shadowgo-linux-amd64 -o shadowgo
chmod +x shadowgo
sudo mv shadowgo /usr/local/bin/
```

**Build from source:**

```bash
# Build locally
go build -o shadowgo ./cmd/shadowgo

# Install to $GOPATH/bin (typically ~/go/bin)
go install ./cmd/shadowgo

# System-wide install (requires sudo)
sudo install -m 755 $(go env GOPATH)/bin/shadowgo /usr/local/bin/shadowgo
```

Ensure `~/go/bin` or `/usr/local/bin` is in your PATH.

## Usage

```bash
# Login to X (Twitter) for posting - opens browser, saves token to ~/.config/shadowgo/tokens/x.json
shadowgo login
shadowgo login x
shadowgo /login

# Screenshot (full screen)
./shadowgo -screenshot

# Screenshot with region selection
./shadowgo -screenshot -region

# Screenshot + LLM marketability analysis (OpenRouter/OpenAI)
./shadowgo -screenshot -analyze

# Screenshot + post to X with caption
./shadowgo -screenshot -post -caption "Check out this screenshot!"

# Screenshot, analyze, and post
./shadowgo -screenshot -analyze -post -caption "Marketability tested"

# With custom prompt
./shadowgo -screenshot -analyze -prompt "Is this UI clear and professional?"

# Full-screen video recording
./shadowgo

# Video with region selection
./shadowgo -region

# Include webcam
./shadowgo -webcam

# Custom webcam device
./shadowgo -webcam -webcam-dev /dev/video2

# All options
./shadowgo -region -webcam
```

## Configuration

- `SHADOWGO_OUTPUT_DIR` - Override output directory for videos (default: `~/Videos/shadowgo`)
- `SHADOWGO_SCREENSHOT_DIR` - Override output directory for screenshots (default: `~/Pictures/shadowgo`)
- `OPENROUTER_API_KEY` or `SHADOWGO_API_KEY` - API key for LLM vision (required for `-analyze`)
- `SHADOWGO_LLM_BASE_URL` - API base URL (default: `https://openrouter.ai/api/v1`)
- `SHADOWGO_LLM_MODEL` - Vision model (default: `openai/gpt-4-vision-preview`). OpenRouter uses provider syntax: `openai/gpt-4o`, `anthropic/claude-3-5-sonnet`, etc.
- `SHADOWGO_LLM_PROMPT` - Default prompt for marketability analysis
- `SHADOWGO_X_CLIENT_ID` - X (Twitter) OAuth Client ID (required for `login x`)
- `SHADOWGO_X_CLIENT_SECRET` - X OAuth Client Secret (optional, for confidential apps)
- `SHADOWGO_X_REDIRECT_URI` - X callback URL (default: `http://127.0.0.1:8080/callback`)
- `SHADOWGO_CONFIG_DIR` - Config directory (default: `~/.config/shadowgo`)
- Output files: `screenshot_YYYYMMDD_HHMMSS.png`, `screen_YYYYMMDD_HHMMSS.mp4`, `webcam_YYYYMMDD_HHMMSS.mp4`

## Architecture

**Design:** The host system controls *when* capture runs (cron, systemd timer, keybinding, etc.). ShadowGo handles the capture and processing. Scheduling, retries, and orchestration are the host's responsibility.

```
Host (cron/systemd/keybind)  →  shadowgo -screenshot -analyze  →  capture + LLM + output
```

- **Primary target:** Arch Linux + Hyprland/Wayland
- **Other platforms:** macOS, Windows, other Linux distros can get it working with platform-specific tools (e.g., different screenshot tools, different audio/video capture). The binary is portable; dependencies (grim, ffmpeg, PipeWire) vary by OS.

```
cmd/shadowgo/     - Entry point
internal/
  config/         - Paths, quality settings
  recorder/       - Recorder interface, PipeWireRecorder, WebcamRecorder
  orchestrator/   - Goroutine management, health checks, signal handling
```

## Security

- **Never commit** `.env`, `tokens/`, or any file containing API keys or OAuth tokens.
- Tokens are stored in `~/.config/shadowgo/tokens/` (outside the repo).
- Use `.env.example` as a template; copy to `.env` and fill in locally (`.env` is gitignored).

## License

MIT
