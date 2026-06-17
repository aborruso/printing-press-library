package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	oauthTokenURL = "https://sanita.puglia.it/sanita-auth/oauth/token"
	// Basic YW9sLWNpZDphb2xAUFdEMjAxOUA= encodes "aol-cid:aol@PWD2019@"
	oauthBasicAuth = "Basic YW9sLWNpZDphb2xAUFdEMjAxOUA="
)

type oauthToken struct {
	value     string
	expiresAt time.Time
	mu        sync.Mutex
}

var cachedToken oauthToken

// fetchClientCredentialsToken obtains a new OAuth2 client_credentials token.
func fetchClientCredentialsToken() (string, error) {
	data := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequest("POST", oauthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("building oauth request: %w", err)
	}
	req.Header.Set("Authorization", oauthBasicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("oauth token: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing oauth response: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("oauth response missing access_token")
	}
	return result.AccessToken, nil
}

// getAutoToken returns a cached valid token, refreshing it when expired or missing.
func getAutoToken() (string, error) {
	cachedToken.mu.Lock()
	defer cachedToken.mu.Unlock()
	if cachedToken.value != "" && time.Now().Before(cachedToken.expiresAt) {
		return cachedToken.value, nil
	}
	token, err := fetchClientCredentialsToken()
	if err != nil {
		return "", err
	}
	cachedToken.value = token
	// Tokens typically expire in 1h; refresh 5 min early
	cachedToken.expiresAt = time.Now().Add(55 * time.Minute)
	return token, nil
}
