package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	xAuthURL   = "https://x.com/i/oauth2/authorize"
	xTokenURL  = "https://api.twitter.com/2/oauth2/token"
	xScopes    = "tweet.read tweet.write users.read offline.access"
	xStateLen  = 32
)

// XToken holds the OAuth token response from X.
type XToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// XLogin performs OAuth 2.0 PKCE login for X (Twitter).
func XLogin(ctx context.Context, clientID, clientSecret, redirectURI string) (*XToken, error) {
	if clientID == "" {
		return nil, fmt.Errorf("SHADOWGO_X_CLIENT_ID is required")
	}

	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("generate PKCE: %w", err)
	}

	state, err := randomBase64URL(xStateLen)
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	authURL := buildXAuthURL(clientID, redirectURI, challenge, state)
	code, err := runCallbackServer(ctx, authURL, redirectURI, state)
	if err != nil {
		return nil, err
	}

	token, err := exchangeXCode(ctx, clientID, clientSecret, redirectURI, code, verifier)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	return token, nil
}

func buildXAuthURL(clientID, redirectURI, challenge, state string) string {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("scope", xScopes)
	v.Set("state", state)
	v.Set("code_challenge", challenge)
	v.Set("code_challenge_method", "S256")
	return xAuthURL + "?" + v.Encode()
}

func exchangeXCode(ctx context.Context, clientID, clientSecret, redirectURI, code, verifier string) (*XToken, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", verifier)
	data.Set("client_id", clientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, xTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if clientSecret != "" {
		req.SetBasicAuth(clientID, clientSecret)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token API error %d: %s", resp.StatusCode, string(body))
	}

	var token XToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	return &token, nil
}

func randomBase64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SaveXToken persists the token to ~/.config/shadowgo/tokens/x.json.
func SaveXToken(configDir string, token *XToken) error {
	tokensDir := filepath.Join(configDir, "tokens")
	if err := os.MkdirAll(tokensDir, 0700); err != nil {
		return err
	}
	path := filepath.Join(tokensDir, "x.json")
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadXToken loads the token from ~/.config/shadowgo/tokens/x.json.
func LoadXToken(configDir string) (*XToken, error) {
	path := filepath.Join(configDir, "tokens", "x.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var token XToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}
