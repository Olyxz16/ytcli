package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Result holds the outcome of a successful implicit grant.
type Result struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int
	Scope       string
}

// FlowConfig configures the implicit grant flow.
type FlowConfig struct {
	HubURL      string // e.g. https://ytcli.youtrack.cloud/hub
	ClientID    string // Hub service ID for the CLI app
	Scope       string // YouTrack service ID in Hub
	RedirectURI string // auto-populated
}

// ImplicitFlow runs the OAuth 2.0 implicit grant flow.
// It starts a local HTTP server, opens the browser, and waits for the callback.
func ImplicitFlow(ctx context.Context, cfg FlowConfig) (*Result, error) {
	// Normalize Hub URL
	cfg.HubURL = strings.TrimSuffix(cfg.HubURL, "/")

	// Start local server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	cfg.RedirectURI = fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	resultCh := make(chan *Result, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", callbackHandler(state, resultCh, errCh))
	mux.HandleFunc("/callback/exchange", exchangeHandler(state, resultCh, errCh))

	server := &http.Server{
		Handler:     mux,
		ReadTimeout: 5 * time.Second,
		WriteTimeout:5 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server: %w", err)
		}
	}()

	// Build authorization URL
	authURL := fmt.Sprintf(
		"%s/api/rest/oauth2/auth?response_type=token&state=%s&redirect_uri=%s&request_credentials=default&client_id=%s&scope=%s",
		cfg.HubURL,
		url.QueryEscape(state),
		url.QueryEscape(cfg.RedirectURI),
		url.QueryEscape(cfg.ClientID),
		url.QueryEscape(cfg.Scope),
	)

	fmt.Println("Opening browser for OAuth authentication...")
	fmt.Printf("If the browser does not open, visit this URL manually:\n%s\n", authURL)
	openBrowser(authURL)

	// Wait for result or context cancellation
	select {
	case result := <-resultCh:
		_ = server.Shutdown(context.Background())
		return result, nil
	case err := <-errCh:
		_ = server.Shutdown(context.Background())
		return nil, err
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return nil, ctx.Err()
	}
}

// callbackHandler serves the HTML page that extracts the token from the URL fragment.
func callbackHandler(expectedState string, resultCh chan<- *Result, errCh chan<- error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = callbackTmpl.Execute(w, nil)
	}
}

// exchangeHandler receives the token data from the browser via POST.
func exchangeHandler(expectedState string, resultCh chan<- *Result, errCh chan<- error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			errCh <- fmt.Errorf("decode exchange: %w", err)
			return
		}

		if payload["error"] != "" {
			errCh <- fmt.Errorf("oauth error: %s - %s", payload["error"], payload["error_description"])
			return
		}

		if payload["state"] != expectedState {
			errCh <- fmt.Errorf("state mismatch: expected %q, got %q", expectedState, payload["state"])
			return
		}

		result := &Result{
			AccessToken: payload["access_token"],
			TokenType:   payload["token_type"],
			Scope:       payload["scope"],
		}
		if ei := payload["expires_in"]; ei != "" {
			fmt.Sscanf(ei, "%d", &result.ExpiresIn)
		}

		if result.AccessToken == "" {
			errCh <- fmt.Errorf("no access token in response")
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<!doctype html><html><body><h1>Authentication successful!</h1><p>You can close this window and return to the terminal.</p></body></html>`)
		resultCh <- result
	}
}

var callbackTmpl = template.Must(template.New("callback").Parse(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>ytcli OAuth</title>
<style>
body { font-family: sans-serif; max-width: 600px; margin: 40px auto; padding: 20px; text-align: center; }
.success { color: #2ea043; }
.error { color: #f85149; }
</style>
</head>
<body>
<div id="msg">Completing authentication...</div>
<script>
(function() {
	var hash = window.location.hash;
	if (!hash || hash.length < 2) {
		document.getElementById('msg').innerHTML = '<h1 class="error">No token found</h1><p>The authorization server did not return a token.</p>';
		return;
	}
	var params = new URLSearchParams(hash.slice(1));
	var data = {};
	params.forEach(function(v, k) { data[k] = v; });
	fetch('/callback/exchange', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	}).then(function(r) {
		if (r.ok) {
			document.getElementById('msg').innerHTML = '<h1 class="success">Authentication successful!</h1><p>You can close this window.</p>';
		} else {
			document.getElementById('msg').innerHTML = '<h1 class="error">Failed to complete authentication</h1><p>Check the terminal for details.</p>';
		}
	}).catch(function(e) {
		document.getElementById('msg').innerHTML = '<h1 class="error">Network error</h1><p>' + e.message + '</p>';
	});
})();
</script>
</body>
</html>`))

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	_ = exec.Command(cmd, args...).Start()
}
