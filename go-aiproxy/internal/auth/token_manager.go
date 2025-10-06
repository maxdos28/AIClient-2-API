package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/aiproxy/go-aiproxy/pkg/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// TokenManager manages OAuth tokens with automatic refresh
type TokenManager struct {
	mu           sync.RWMutex
	config       *models.ProviderConfig
	tokenSource  oauth2.TokenSource
	currentToken *oauth2.Token
	expiryBuffer time.Duration
}

// NewTokenManager creates a new token manager
func NewTokenManager(config *models.ProviderConfig) (*TokenManager, error) {
	tm := &TokenManager{
		config:       config,
		expiryBuffer: 5 * time.Minute, // Refresh 5 minutes before expiry
	}

	// Initialize token source
	if err := tm.initializeTokenSource(context.Background()); err != nil {
		return nil, err
	}

	return tm, nil
}

// initializeTokenSource sets up the OAuth2 token source
func (tm *TokenManager) initializeTokenSource(ctx context.Context) error {
	var creds []byte
	var err error

	// Get credentials
	if tm.config.OAuthCredsBase64 != "" {
		// Decode base64 credentials
		creds, err = base64.StdEncoding.DecodeString(tm.config.OAuthCredsBase64)
		if err != nil {
			return fmt.Errorf("failed to decode base64 credentials: %w", err)
		}
	} else if tm.config.OAuthCredsFile != "" {
		// Read from file
		creds, err = ioutil.ReadFile(tm.config.OAuthCredsFile)
		if err != nil {
			return fmt.Errorf("failed to read credentials file: %w", err)
		}
	} else {
		return fmt.Errorf("no OAuth credentials provided")
	}

	// Parse credentials based on provider
	switch tm.config.Provider {
	case models.ProviderGemini:
		// Google OAuth2 for Gemini
		config, err := google.JWTConfigFromJSON(creds, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return fmt.Errorf("failed to create JWT config: %w", err)
		}
		tm.tokenSource = config.TokenSource(ctx)

	case models.ProviderKiro:
		// Custom OAuth2 for Kiro
		var oauthCreds struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			TokenURL     string `json:"token_url"`
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.Unmarshal(creds, &oauthCreds); err != nil {
			return fmt.Errorf("failed to parse Kiro credentials: %w", err)
		}

		config := &oauth2.Config{
			ClientID:     oauthCreds.ClientID,
			ClientSecret: oauthCreds.ClientSecret,
			Endpoint: oauth2.Endpoint{
				TokenURL: oauthCreds.TokenURL,
			},
		}

		// Create token source with refresh token
		token := &oauth2.Token{
			RefreshToken: oauthCreds.RefreshToken,
		}
		tm.tokenSource = config.TokenSource(ctx, token)

	case models.ProviderQwen:
		// Custom OAuth2 for Qwen
		var qwenCreds struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			TokenURL     string `json:"token_url"`
			RefreshToken string `json:"refresh_token"`
			Scope        string `json:"scope"`
		}
		if err := json.Unmarshal(creds, &qwenCreds); err != nil {
			return fmt.Errorf("failed to parse Qwen credentials: %w", err)
		}

		config := &oauth2.Config{
			ClientID:     qwenCreds.ClientID,
			ClientSecret: qwenCreds.ClientSecret,
			Scopes:       []string{qwenCreds.Scope},
			Endpoint: oauth2.Endpoint{
				TokenURL: qwenCreds.TokenURL,
			},
		}

		// Create token source with refresh token
		token := &oauth2.Token{
			RefreshToken: qwenCreds.RefreshToken,
		}
		tm.tokenSource = config.TokenSource(ctx, token)

	default:
		return fmt.Errorf("unsupported provider for OAuth: %s", tm.config.Provider)
	}

	// Get initial token
	token, err := tm.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to get initial token: %w", err)
	}

	tm.currentToken = token
	return nil
}

// GetToken returns a valid access token, refreshing if necessary
func (tm *TokenManager) GetToken(ctx context.Context) (*oauth2.Token, error) {
	tm.mu.RLock()
	token := tm.currentToken
	tm.mu.RUnlock()

	// Check if token needs refresh
	if tm.shouldRefresh(token) {
		return tm.RefreshToken(ctx)
	}

	return token, nil
}

// RefreshToken forces a token refresh
func (tm *TokenManager) RefreshToken(ctx context.Context) (*oauth2.Token, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Get new token from token source
	token, err := tm.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	tm.currentToken = token
	return token, nil
}

// IsTokenValid checks if the current token is valid
func (tm *TokenManager) IsTokenValid() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return !tm.shouldRefresh(tm.currentToken)
}

// shouldRefresh checks if token should be refreshed
func (tm *TokenManager) shouldRefresh(token *oauth2.Token) bool {
	if token == nil {
		return true
	}

	// Check if token is expired or about to expire
	expiryTime := token.Expiry
	if expiryTime.IsZero() {
		return false // No expiry, assume valid
	}

	return time.Now().Add(tm.expiryBuffer).After(expiryTime)
}

// GetExpiryTime returns the token expiry time
func (tm *TokenManager) GetExpiryTime() time.Time {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.currentToken == nil {
		return time.Time{}
	}

	return tm.currentToken.Expiry
}

// SetExpiryBuffer sets the buffer time before token expiry for refresh
func (tm *TokenManager) SetExpiryBuffer(buffer time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.expiryBuffer = buffer
}