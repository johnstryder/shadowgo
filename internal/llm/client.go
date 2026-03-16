package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/agorator/shadowgo/internal/config"
)

// Client is an OpenAI-compatible vision API client (works with OpenRouter, OpenAI, etc.).
type Client struct {
	cfg    *config.Config
	client *http.Client
}

// NewClient creates a new LLM client.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg:    cfg,
		client: &http.Client{},
	}
}

// AnalyzeImage sends an image to the vision API and returns the analysis text.
func (c *Client) AnalyzeImage(ctx context.Context, imagePath string, prompt string) (string, error) {
	if c.cfg.LLMAPIKey == "" {
		return "", fmt.Errorf("LLM API key not set (OPENROUTER_API_KEY or SHADOWGO_API_KEY)")
	}

	dataURL, err := encodeImageBase64(imagePath)
	if err != nil {
		return "", fmt.Errorf("encode image: %w", err)
	}

	if prompt == "" {
		prompt = c.cfg.LLMPrompt
	}

	// OpenAI / OpenRouter chat completions format
	reqBody := chatRequest{
		Model: c.cfg.LLMModel,
		Messages: []message{
			{
				Role: "user",
				Content: []contentPart{
					{Type: "text", Text: prompt},
					{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
				},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := strings.TrimSuffix(c.cfg.LLMBaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.cfg.LLMAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/agorator/shadowgo") // OpenRouter likes this

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := chatResp.Choices[0].Message.Content
	if content == "" {
		return "", fmt.Errorf("empty response from model")
	}

	return content, nil
}

// OpenAI / OpenRouter chat completions types
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type     string   `json:"type"`
	Text     string   `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func encodeImageBase64(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(imagePath))
	mime := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}