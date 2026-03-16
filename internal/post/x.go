package post

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/agorator/shadowgo/internal/auth"
)

const (
	mediaUploadURL = "https://upload.twitter.com/1.1/media/upload.json"
	tweetsURL      = "https://api.twitter.com/2/tweets"
)

// PostImage uploads an image to X and posts it with the given caption.
func PostImage(ctx context.Context, token *auth.XToken, imagePath string, caption string) (tweetID string, err error) {
	mediaID, err := uploadMedia(ctx, token.AccessToken, imagePath)
	if err != nil {
		return "", fmt.Errorf("upload media: %w", err)
	}

	return createTweet(ctx, token.AccessToken, caption, []string{mediaID})
}

func uploadMedia(ctx context.Context, accessToken string, filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	part, err := w.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mediaUploadURL, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("media upload error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		MediaID int64 `json:"media_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse media response: %w", err)
	}
	return fmt.Sprintf("%d", result.MediaID), nil
}

func createTweet(ctx context.Context, accessToken string, text string, mediaIDs []string) (string, error) {
	payload := make(map[string]interface{})
	if text != "" {
		payload["text"] = text
	}
	if len(mediaIDs) > 0 {
		payload["media"] = map[string]interface{}{
			"media_ids": mediaIDs,
		}
	}
	if len(payload) == 0 {
		return "", fmt.Errorf("tweet must have text or media")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tweetsURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("tweet error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse tweet response: %w", err)
	}
	return result.Data.ID, nil
}
