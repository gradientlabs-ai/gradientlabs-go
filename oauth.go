package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultOAuthTokenURL = "https://id.gradient-labs.ai/oauth2/token"

// oauthTokenSource fetches access tokens with the OAuth 2.0 client credentials
// grant, and reuses each one until shortly before it expires.
type oauthTokenSource struct {
	tokenURL     string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	now          func() time.Time

	mu          sync.Mutex
	accessToken string
	refreshAt   time.Time
}

func (s *oauthTokenSource) token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accessToken != "" && s.now().Before(s.refreshAt) {
		return s.accessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", s.clientID)
	form.Set("client_secret", s.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", userAgent)

	rsp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode < 200 || rsp.StatusCode > 299 {
		te := &OAuthTokenError{StatusCode: rsp.StatusCode}
		var payload struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.NewDecoder(rsp.Body).Decode(&payload); err == nil {
			te.Code = payload.Error
			te.Description = payload.ErrorDescription
		}
		return "", te
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", errors.New("oauth token response has no access_token")
	}

	lifetime := time.Duration(payload.ExpiresIn) * time.Second
	margin := lifetime / 2
	if margin > time.Minute {
		margin = time.Minute
	}

	s.accessToken = payload.AccessToken
	s.refreshAt = s.now().Add(lifetime - margin)
	return s.accessToken, nil
}
