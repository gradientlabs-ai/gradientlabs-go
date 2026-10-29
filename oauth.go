package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultOAuthTokenURL = "https://id.gradient-labs.ai/oauth2/token"

	// defaultTokenLifetime is assumed when the token response has no
	// expires_in.
	defaultTokenLifetime = 30 * time.Minute

	tokenRequestTimeout = 30 * time.Second

	// After a failed token request the client waits before asking again: the
	// response's Retry-After (at most maxRetryAfter), or else a delay that
	// doubles from minRetryDelay to maxRetryDelay with each failure in a row.
	minRetryDelay = time.Second
	maxRetryDelay = time.Minute
	maxRetryAfter = 5 * time.Minute

	maxTokenResponseBytes = 1 << 20
)

// oauthTokenSource fetches access tokens with the OAuth 2.0 client credentials
// grant, and reuses each one until shortly before it expires. Concurrent
// callers share one token request, and each stops waiting when its own context
// is done.
type oauthTokenSource struct {
	tokenURL     string
	clientID     string
	clientSecret string
	// signer, when set, authenticates each token request with a freshly
	// signed client assertion instead of clientSecret.
	signer *assertionSigner
	// httpClient never follows redirects, so a redirect can't carry the
	// secret or an assertion to another host.
	httpClient *http.Client
	now        func() time.Time
	// requestTimeout bounds a whole token request, signing included.
	requestTimeout time.Duration

	mu          sync.Mutex
	accessToken string
	refreshAt   time.Time
	// expiresAt is when accessToken stops working. A refresh that fails
	// before then falls back to accessToken.
	expiresAt time.Time
	// Until retryAt, callers get lastErr instead of a new token request.
	retryAt  time.Time
	lastErr  error
	failures int
	inflight *tokenFetch
}

type tokenFetch struct {
	done  chan struct{}
	token string
	err   error
}

func (s *oauthTokenSource) token(ctx context.Context) (string, error) {
	s.mu.Lock()
	now := s.now()
	if s.accessToken != "" && now.Before(s.refreshAt) {
		token := s.accessToken
		s.mu.Unlock()
		return token, nil
	}
	if now.Before(s.retryAt) {
		token, err := s.cachedTokenOr(now, s.lastErr)
		s.mu.Unlock()
		return token, err
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
		if f.err == nil {
			return f.token, nil
		}
		s.mu.Lock()
		token, err := s.cachedTokenOr(s.now(), f.err)
		s.mu.Unlock()
		return token, err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// cachedTokenOr returns the cached token while it is still valid, and err
// otherwise. s.mu must be held.
func (s *oauthTokenSource) cachedTokenOr(now time.Time, err error) (string, error) {
	if s.accessToken != "" && now.Before(s.expiresAt) {
		return s.accessToken, nil
	}
	return "", err
}

// invalidate drops token if it is still the cached one, so that the next call
// fetches a new token instead of reusing one the API has refused.
func (s *oauthTokenSource) invalidate(token string) {
	s.mu.Lock()
	if s.accessToken == token {
		s.accessToken = ""
		s.refreshAt = time.Time{}
		s.expiresAt = time.Time{}
	}
	s.mu.Unlock()
}

// fetch runs one token request on behalf of every waiting caller. It keeps the
// first caller's context values (for tracing) but not its cancellation, so one
// caller giving up doesn't fail the request for the others.
func (s *oauthTokenSource) fetch(ctx context.Context, f *tokenFetch) {
	ctx, cancel := context.WithTimeout(detachedContext{ctx}, s.requestTimeout)
	defer cancel()

	token, lifetime, err := s.requestWithoutPanics(ctx)
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		// Not wrapped, so that callers don't mistake it for their own deadline.
		err = fmt.Errorf("oauth token request: no answer within %s", s.requestTimeout)
	}

	s.mu.Lock()
	// Round(0) drops the monotonic clock reading, which stops while the
	// machine sleeps, so expiry is judged by the wall clock.
	now := s.now().Round(0)
	if err == nil {
		margin := lifetime / 2
		if margin > time.Minute {
			margin = time.Minute
		}
		s.accessToken = token
		s.refreshAt = now.Add(lifetime - margin)
		s.expiresAt = now.Add(lifetime)
		s.retryAt, s.lastErr, s.failures = time.Time{}, nil, 0
	} else {
		s.failures++
		s.retryAt = now.Add(retryDelay(s.failures, err))
		s.lastErr = err
	}
	s.inflight = nil
	s.mu.Unlock()

	f.token, f.err = token, err
	close(f.done)
}

// retryDelay is how long to wait after the failures-th failed token request in
// a row.
func retryDelay(failures int, err error) time.Duration {
	var te *OAuthTokenError
	if errors.As(err, &te) && te.RetryAfter > 0 {
		if te.RetryAfter > maxRetryAfter {
			return maxRetryAfter
		}
		return te.RetryAfter
	}
	delay := minRetryDelay
	for i := 1; i < failures && delay < maxRetryDelay; i++ {
		delay *= 2
	}
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	return delay
}

// requestWithoutPanics turns a panic in a custom signer or transport into an
// error, as it would otherwise crash the process from fetch's goroutine.
func (s *oauthTokenSource) requestWithoutPanics(ctx context.Context) (token string, lifetime time.Duration, err error) {
	defer func() {
		if r := recover(); r != nil {
			token, lifetime, err = "", 0, fmt.Errorf("oauth token request: panic: %v", r)
		}
	}()
	return s.request(ctx)
}

func (s *oauthTokenSource) request(ctx context.Context) (string, time.Duration, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", s.clientID)
	if s.signer != nil {
		assertion, err := s.signAssertion(ctx)
		if err != nil {
			return "", 0, fmt.Errorf("oauth token request: %w", err)
		}
		form.Set("client_assertion_type", clientAssertionType)
		form.Set("client_assertion", assertion)
	} else {
		form.Set("client_secret", s.clientSecret)
	}

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
	body := io.LimitReader(rsp.Body, maxTokenResponseBytes)

	if rsp.StatusCode < 200 || rsp.StatusCode > 299 {
		te := &OAuthTokenError{
			StatusCode: rsp.StatusCode,
			RetryAfter: parseRetryAfter(rsp.Header.Get("Retry-After"), s.now()),
		}
		var payload struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.NewDecoder(body).Decode(&payload); err == nil {
			te.Code = payload.Error
			te.Description = payload.ErrorDescription
		}
		return "", 0, te
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
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

// signAssertion stops waiting when ctx is done, because crypto.Signer takes no
// context and a signer such as a KMS client may never return. A signer that
// hangs is left behind in its goroutine.
func (s *oauthTokenSource) signAssertion(ctx context.Context) (string, error) {
	type result struct {
		assertion string
		err       error
	}
	done := make(chan result, 1)
	now := s.now()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: fmt.Errorf("client assertion: signer panicked: %v", r)}
			}
		}()
		assertion, err := s.signer.sign(now)
		done <- result{assertion, err}
	}()

	select {
	case r := <-done:
		return r.assertion, r.err
	case <-ctx.Done():
		return "", fmt.Errorf("client assertion: signing: %w", ctx.Err())
	}
}

// parseRetryAfter reads a Retry-After header given in seconds or as an HTTP
// date, and returns 0 when it is absent or unreadable.
func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil && at.After(now) {
		return at.Sub(now)
	}
	return 0
}

// detachedContext keeps a context's values but never expires or cancels.
type detachedContext struct{ context.Context }

func (detachedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (detachedContext) Done() <-chan struct{}       { return nil }
func (detachedContext) Err() error                  { return nil }
