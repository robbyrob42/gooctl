package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
)

const (
	// Default redirect URI for local OAuth flow
	redirectURI = "http://localhost:8085/callback"
	// Token file name
	tokenFile = "token.json"
	// Credentials file name
	credentialsFile = "credentials.json"
)

// Config holds the OAuth2 configuration
type Config struct {
	configDir string
	oauth     *oauth2.Config
}

// NewConfig creates a new OAuth2 config
func NewConfig() (*Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Try to load credentials from file
	credPath := filepath.Join(configDir, credentialsFile)
	credData, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("credentials file not found at %s - please download OAuth credentials from Google Cloud Console and save them there", credPath)
	}

	config, err := google.ConfigFromJSON(credData,
		gmail.GmailReadonlyScope,
		gmail.GmailSendScope,
		gmail.GmailModifyScope,
		calendar.CalendarReadonlyScope,
		calendar.CalendarScope,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Override redirect URI for local flow
	config.RedirectURL = redirectURI

	return &Config{
		configDir: configDir,
		oauth:     config,
	}, nil
}

// getConfigDir returns the configuration directory path
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "gooctl"), nil
}

// GetConfigDir returns the configuration directory (public accessor)
func GetConfigDir() (string, error) {
	return getConfigDir()
}

// TokenPath returns the path to the token file
func (c *Config) TokenPath() string {
	return filepath.Join(c.configDir, tokenFile)
}

// HasValidToken checks if a valid token exists
func (c *Config) HasValidToken() bool {
	token, err := c.LoadToken()
	if err != nil {
		return false
	}
	return token.Valid()
}

// LoadToken loads the token from disk
func (c *Config) LoadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(c.TokenPath())
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// SaveToken saves the token to disk
func (c *Config) SaveToken(token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.TokenPath(), data, 0600)
}

// GetClient returns an authenticated HTTP client
func (c *Config) GetClient(ctx context.Context) (*http.Client, error) {
	token, err := c.LoadToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated - run 'gooctl auth' first")
	}

	// Create token source that auto-refreshes
	tokenSource := c.oauth.TokenSource(ctx, token)

	// Get potentially refreshed token
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Save refreshed token if it changed
	if newToken.AccessToken != token.AccessToken {
		if err := c.SaveToken(newToken); err != nil {
			// Log but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to save refreshed token: %v\n", err)
		}
	}

	return oauth2.NewClient(ctx, tokenSource), nil
}

// Authenticate performs the OAuth2 flow with browser
func (c *Config) Authenticate(ctx context.Context) error {
	// Create a channel to receive the auth code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start local server to receive callback
	server := &http.Server{Addr: ":8085"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code in callback")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		// Send success page
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>gooctl - Authentication Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #4285F4 0%%, #34A853 100%%);
        }
        .card {
            background: white;
            padding: 40px;
            border-radius: 16px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.15);
            text-align: center;
        }
        h1 { color: #34A853; margin-bottom: 10px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="card">
        <h1>✓ Authentication Successful!</h1>
        <p>You can close this window and return to your terminal.</p>
    </div>
</body>
</html>`)

		codeChan <- code
	})

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Generate auth URL
	authURL := c.oauth.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("Opening browser for authentication...")
	fmt.Printf("If the browser doesn't open, visit this URL:\n%s\n\n", authURL)

	// Open browser
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	// Wait for callback or timeout
	var code string
	select {
	case code = <-codeChan:
		// Success
	case err := <-errChan:
		server.Shutdown(ctx)
		return fmt.Errorf("authentication failed: %w", err)
	case <-time.After(5 * time.Minute):
		server.Shutdown(ctx)
		return fmt.Errorf("authentication timed out")
	case <-ctx.Done():
		server.Shutdown(ctx)
		return ctx.Err()
	}

	// Shutdown server
	server.Shutdown(ctx)

	// Exchange code for token
	token, err := c.oauth.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Save token
	if err := c.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// Logout removes the stored token
func (c *Config) Logout() error {
	tokenPath := c.TokenPath()
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return nil // Already logged out
	}
	return os.Remove(tokenPath)
}

// openBrowser opens the default browser to the specified URL
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
