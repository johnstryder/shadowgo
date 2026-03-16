// ShadowGo - Background recording service for Screen, Audio, and Webcam on Linux (Wayland/Hyprland)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/agorator/shadowgo/internal/auth"
	"github.com/agorator/shadowgo/internal/config"
	"github.com/agorator/shadowgo/internal/llm"
	"github.com/agorator/shadowgo/internal/orchestrator"
	"github.com/agorator/shadowgo/internal/post"
	"github.com/agorator/shadowgo/internal/recorder"
)

func main() {
	// Subcommand: login (like pi-coding-agent /login)
	if len(os.Args) > 1 {
		arg := strings.TrimPrefix(strings.ToLower(os.Args[1]), "/")
		if arg == "login" {
			runLogin(os.Args[2:])
			return
		}
	}

	screenshot := flag.Bool("screenshot", false, "Capture a screenshot (uses grim)")
	analyze := flag.Bool("analyze", false, "Send screenshot to LLM for marketability analysis (OpenAI/OpenRouter)")
	postX := flag.Bool("post", false, "Post screenshot to X (Twitter) after capture")
	caption := flag.String("caption", "", "Caption for the post (use with -post)")
	prompt := flag.String("prompt", "", "Custom prompt for LLM analysis (overrides SHADOWGO_LLM_PROMPT)")
	region := flag.Bool("region", false, "Use slurp to select screen region (requires slurp)")
	webcam := flag.Bool("webcam", false, "Also record webcam via v4l2")
	webcamDev := flag.String("webcam-dev", "/dev/video0", "Webcam v4l2 device path")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := config.DefaultConfig()

	// Screenshot mode: one-shot image capture
	if *screenshot {
		path, err := runScreenshotMode(context.Background(), cfg, *region, *analyze, *postX, *caption, *prompt, log)
		if err != nil {
			log.Error("screenshot failed", "error", err)
			os.Exit(1)
		}
		log.Info("screenshot saved", "path", path)
		return
	}

	// Video recording mode
	slurpAvail, grimAvail := recorder.DetectRegionTools()
	log.Info("region tools", "slurp", slurpAvail, "grim", grimAvail)

	var regionPtr *recorder.Region
	if *region && slurpAvail {
		r, err := recorder.GetRegionFromSlurp(context.Background())
		if err != nil {
			log.Warn("region selection failed, using full screen", "error", err)
		} else {
			regionPtr = r
			log.Info("region selected", "x", r.X, "y", r.Y, "w", r.W, "h", r.H)
		}
	} else if *region && !slurpAvail {
		log.Warn("--region requested but slurp not found, using full screen")
	}

	recorders := []recorder.Recorder{
		recorder.NewPipeWireRecorder(cfg, regionPtr),
	}

	if *webcam {
		recorders = append(recorders, recorder.NewWebcamRecorder(cfg, *webcamDev))
	}

	orch := orchestrator.New(cfg, recorders,
		orchestrator.WithLogger(log),
		orchestrator.WithHealthCheckInterval(5*time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	log.Info("ShadowGo starting", "output_dir", cfg.OutputDir)
	if err := orch.Run(ctx); err != nil {
		log.Error("orchestrator failed", "error", err)
		os.Exit(1)
	}
	log.Info("ShadowGo stopped")
}

func runScreenshotMode(ctx context.Context, cfg *config.Config, useRegion bool, analyze bool, postX bool, caption string, promptOverride string, log *slog.Logger) (string, error) {
	slurpAvail, _ := recorder.DetectRegionTools()

	var regionPtr *recorder.Region
	if useRegion && slurpAvail {
		r, err := recorder.GetRegionFromSlurp(ctx)
		if err != nil {
			return "", err
		}
		regionPtr = r
	} else if useRegion && !slurpAvail {
		return "", fmt.Errorf("--region requires slurp (not found in PATH)")
	}

	path, err := recorder.CaptureScreenshot(ctx, cfg, regionPtr)
	if err != nil {
		return "", err
	}

	if analyze {
		client := llm.NewClient(cfg)
		analysis, err := client.AnalyzeImage(ctx, path, promptOverride)
		if err != nil {
			return path, fmt.Errorf("LLM analysis failed: %w", err)
		}
		log.Info("marketability analysis", "result", analysis)
	}

	if postX {
		token, err := auth.LoadXToken(cfg.ConfigDir)
		if err != nil {
			return path, fmt.Errorf("load X token (run 'shadowgo login' first): %w", err)
		}
		// X allows media-only tweets; empty caption is fine
		tweetID, err := post.PostImage(ctx, token, path, caption)
		if err != nil {
			return path, fmt.Errorf("post to X failed: %w", err)
		}
		log.Info("posted to X", "tweet_id", tweetID)
	}

	return path, nil
}

func runLogin(args []string) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	platform := "x"
	if len(args) > 0 && args[0] != "" {
		if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
			fmt.Fprintln(os.Stderr, `Usage: shadowgo login [platform]

Authenticate with a social platform for posting screenshots.

Platforms:
  x, twitter    X (Twitter) - requires SHADOWGO_X_CLIENT_ID

Environment:
  SHADOWGO_X_CLIENT_ID        X app Client ID (from developer portal)
  SHADOWGO_X_CLIENT_SECRET     Optional, for confidential apps
  SHADOWGO_X_REDIRECT_URI      Callback URL (default: http://127.0.0.1:8080/callback)

Register http://127.0.0.1:8080/callback in your X app's callback URLs.`)
			return
		}
		platform = strings.ToLower(args[0])
	}

	cfg := config.DefaultConfig()
	pc := cfg.SocialPlatformConfig(platform)

	switch platform {
	case "x", "twitter":
		clientID := pc["client_id"]
		clientSecret := pc["client_secret"]
		redirectURI := pc["redirect_uri"]

		log.Info("logging in to X (Twitter)", "redirect_uri", redirectURI)

		token, err := auth.XLogin(context.Background(), clientID, clientSecret, redirectURI)
		if err != nil {
			log.Error("X login failed", "error", err)
			os.Exit(1)
		}

		if err := auth.SaveXToken(cfg.ConfigDir, token); err != nil {
			log.Error("failed to save token", "error", err)
			os.Exit(1)
		}

		log.Info("login successful", "token_path", cfg.ConfigDir+"/tokens/x.json")
	default:
		log.Error("unknown platform", "platform", platform, "supported", "x")
		os.Exit(1)
	}
}
