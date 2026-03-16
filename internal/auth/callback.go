package auth

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const successHTML = `<!DOCTYPE html><html><head><title>Login Successful</title></head>
<body style="font-family:sans-serif;text-align:center;padding:4rem;">
<h1>Login Successful!</h1>
<p>You can close this window.</p>
</body></html>`

// runCallbackServer starts a local HTTP server, opens the browser to authURL,
// and returns the authorization code when the user completes the flow.
func runCallbackServer(ctx context.Context, authURL, redirectURI, expectedState string) (code string, err error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", fmt.Errorf("invalid redirect URI: %w", err)
	}

	host := u.Host
	if host == "" {
		host = "127.0.0.1:8080"
	}
	if !strings.Contains(host, ":") {
		host = "127.0.0.1:" + host
	}

	path := u.Path
	if path == "" {
		path = "/"
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errCh <- fmt.Errorf("state mismatch (possible CSRF)")
			return
		}

		code = r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code parameter", http.StatusBadRequest)
			errCh <- fmt.Errorf("missing code in callback")
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, successHTML)

		codeCh <- code
	})

	server := &http.Server{Addr: host, Handler: mux}
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", fmt.Errorf("listen on %s: %w", host, err)
	}

	go func() {
		_ = server.Serve(listener)
	}()

	defer func() {
		_ = server.Shutdown(context.Background())
	}()

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Open this URL in your browser: %s\n", authURL)
	}

	select {
	case code = <-codeCh:
		return code, nil
	case e := <-errCh:
		return "", e
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("login timed out")
	}
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
