package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultOAuthTokenURL = "https://id.gradient-labs.ai/oauth2/token"

	// defaultTokenLifetime is assumed when the token response has no
	// expires_in.
	defaultTokenLifetime = 5 * time.Minute

	tokenRequestTimeout = 30 * time.Second
)

// oauthTokenSource fetches access tokens with the OAuth 2.0 client credentials
// grant, and reuses each one until shortly before it expires. Concurrent
// callers share one token request, and each stops waiting when its own context
// is done.
type oauthTokenSource struct {
	tokenURL     string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	now          func() time.Time

	mu          sync.Mutex
	accessToken string
	refreshAt   time.Time
	inflight    *tokenFetch
}

type tokenFetch struct {
	done  chan struct{}
	token string
	err   error
}

func (s *oauthTokenSource) token(ctx context.Context) (string, error) {
	s.mu.Lock()
	if s.accessToken != "" && s.now().Before(s.refreshAt) {
		token := s.accessToken
		s.mu.Unlock()
		return token, nil
	}
	f := s.inflight
	if f == nil {
		f = &tokenFetch{done: make(chan struct{})}
		s.inflight = f
		go s.fetch(ctx, f)
	}
	s.mu.Unlock()

	select {
	case <-f.done:
		return f.token, f.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// fetch runs one token request on behalf of every waiting caller. It keeps the
// first caller's context values (for tracing) but not its cancellation, so one
// caller giving up doesn't fail the request for the others.
func (s *oauthTokenSource) fetch(ctx context.Context, f *tokenFetch) {
	ctx, cancel := context.WithTimeout(detachedContext{ctx}, tokenRequestTimeout)
	defer cancel()

	token, lifetime, err := s.request(ctx)

	s.mu.Lock()
	if err == nil {
		margin := lifetime / 2
		if margin > time.Minute {
			margin = time.Minute
		}
		s.accessToken = token
		// Round(0) drops the monotonic clock reading, which stops while the
		// machine sleeps, so expiry is judged by the wall clock.
		s.refreshAt = s.now().Add(lifetime - margin).Round(0)
	}
	s.inflight = nil
	s.mu.Unlock()

	f.token, f.err = token, err
	close(f.done)
}

func (s *oauthTokenSource) request(ctx context.Context) (string, time.Duration, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", s.clientID)
	form.Set("client_secret", s.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("oauth token request: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", userAgent)

	rsp, err := s.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("oauth token request: %w", err)
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
		return "", 0, te
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&payload); err != nil {
		return "", 0, fmt.Errorf("oauth token request: decoding response: %w", err)
	}
	if payload.AccessToken == "" {
		return "", 0, errors.New("oauth token request: response has no access_token")
	}

	lifetime := time.Duration(payload.ExpiresIn) * time.Second
	if lifetime <= 0 {
		lifetime = defaultTokenLifetime
	}
	return payload.AccessToken, lifetime, nil
}

// detachedContext keeps a context's values but never expires or cancels.
type detachedContext struct{ context.Context }

func (detachedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (detachedContext) Done() <-chan struct{}       { return nil }
func (detachedContext) Err() error                  { return nil }
