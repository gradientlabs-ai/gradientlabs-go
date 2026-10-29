package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type tokenServer struct {
	*httptest.Server
	requests int32

	// omitExpiresIn leaves expires_in out of the response.
	omitExpiresIn bool
	// started and release, when set, make each request signal started and
	// then wait for release before answering.
	started chan struct{}
	release chan struct{}

	mu sync.Mutex
	// failStatus, when set, answers every request with that status, a
	// temporarily_unavailable error and failRetryAfter as Retry-After.
	failStatus     int
	failRetryAfter string
}

func (ts *tokenServer) failWith(status int, retryAfter string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.failStatus, ts.failRetryAfter = status, retryAfter
}

func newTokenServer(t *testing.T) *tokenServer {
	t.Helper()
	ts := &tokenServer{}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if got := r.PostForm.Get("grant_type"); got != "client_credentials" {
			t.Errorf("grant_type = %q", got)
		}
		if r.PostForm.Get("client_id") != "client_123" || r.PostForm.Get("client_secret") != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"invalid_client","error_description":"Invalid client credentials."}`)
			return
		}
		n := atomic.AddInt32(&ts.requests, 1)
		ts.mu.Lock()
		failStatus, failRetryAfter := ts.failStatus, ts.failRetryAfter
		ts.mu.Unlock()
		if failStatus != 0 {
			if failRetryAfter != "" {
				w.Header().Set("Retry-After", failRetryAfter)
			}
			w.WriteHeader(failStatus)
			fmt.Fprint(w, `{"error":"temporarily_unavailable","error_description":"Try again later."}`)
			return
		}
		if ts.started != nil {
			ts.started <- struct{}{}
			<-ts.release
		}
		if ts.omitExpiresIn {
			fmt.Fprintf(w, `{"access_token":"token-%d","token_type":"Bearer"}`, n)
			return
		}
		fmt.Fprintf(w, `{"access_token":"token-%d","token_type":"Bearer","expires_in":300}`, n)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func (ts *tokenServer) requestCount() int32 {
	return atomic.LoadInt32(&ts.requests)
}

type apiServer struct {
	*httptest.Server
	mu          sync.Mutex
	authHeaders []string
}

func newAPIServer(t *testing.T) *apiServer {
	t.Helper()
	as := &apiServer{}
	as.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		as.mu.Lock()
		as.authHeaders = append(as.authHeaders, r.Header.Get("Authorization"))
		as.mu.Unlock()
		fmt.Fprint(w, `{"id":"conv-1"}`)
	}))
	t.Cleanup(as.Close)
	return as
}

func (as *apiServer) headers() []string {
	as.mu.Lock()
	defer as.mu.Unlock()
	return append([]string(nil), as.authHeaders...)
}

func newOAuthClient(t *testing.T, tokenURL string, opts ...Option) *Client {
	t.Helper()
	opts = append(opts,
		WithOAuthClientCredentials("client_123", "secret"),
		WithOAuthTokenURL(tokenURL),
	)
	c, err := NewClient(opts...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestOAuthClientCredentials(t *testing.T) {
	tokens := newTokenServer(t)
	api := newAPIServer(t)
	c := newOAuthClient(t, tokens.URL, WithURL(api.URL))

	now := time.Now()
	c.oauthTokenSource.now = func() time.Time { return now }

	ctx := context.Background()
	read := func() {
		t.Helper()
		if _, err := c.ReadConversation(ctx, "conv-1", &ReadParams{}); err != nil {
			t.Fatal(err)
		}
	}

	// The token lasts 300s and is renewed a minute early, at 240s.
	read()
	now = now.Add(4*time.Minute - time.Second)
	read()
	now = now.Add(2 * time.Second)
	read()

	want := []string{"Bearer token-1", "Bearer token-1", "Bearer token-2"}
	if got := api.headers(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("Authorization headers = %q, want %q", got, want)
	}
	if got := tokens.requestCount(); got != 2 {
		t.Errorf("token requests = %d, want 2", got)
	}
}

func TestOAuthTokenWithoutExpiresIn(t *testing.T) {
	tokens := newTokenServer(t)
	tokens.omitExpiresIn = true
	c := newOAuthClient(t, tokens.URL)

	now := time.Now()
	c.oauthTokenSource.now = func() time.Time { return now }

	ctx := context.Background()
	// The default 30 minute lifetime is renewed a minute early, at 29 minutes.
	var got []string
	for _, step := range []time.Duration{0, 29*time.Minute - time.Second, 2 * time.Second} {
		now = now.Add(step)
		token, err := c.oauthTokenSource.token(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, token)
	}

	want := []string{"token-1", "token-1", "token-2"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("tokens = %q, want %q", got, want)
	}
}

func TestOAuthTokenRefreshUsesWallClock(t *testing.T) {
	tokens := newTokenServer(t)
	c := newOAuthClient(t, tokens.URL)

	if _, err := c.oauthTokenSource.token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if refreshAt := c.oauthTokenSource.refreshAt; refreshAt != refreshAt.Round(0) {
		t.Errorf("refreshAt %v keeps a monotonic clock reading", refreshAt)
	}
}

func TestOAuthTokenConcurrentCallersShareOneRequest(t *testing.T) {
	tokens := newTokenServer(t)
	tokens.started = make(chan struct{}, 1)
	tokens.release = make(chan struct{})
	c := newOAuthClient(t, tokens.URL)

	const callers = 10
	results := make(chan string, callers)
	for i := 0; i < callers; i++ {
		go func() {
			token, err := c.oauthTokenSource.token(context.Background())
			if err != nil {
				t.Error(err)
			}
			results <- token
		}()
	}

	<-tokens.started
	close(tokens.release)
	for i := 0; i < callers; i++ {
		if got := <-results; got != "token-1" {
			t.Errorf("token = %q, want token-1", got)
		}
	}
	if got := tokens.requestCount(); got != 1 {
		t.Errorf("token requests = %d, want 1", got)
	}
}

func TestOAuthTokenCallerStopsWaitingAtItsDeadline(t *testing.T) {
	tokens := newTokenServer(t)
	tokens.started = make(chan struct{}, 1)
	tokens.release = make(chan struct{})
	c := newOAuthClient(t, tokens.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	impatient := make(chan error, 1)
	go func() {
		_, err := c.oauthTokenSource.token(ctx)
		impatient <- err
	}()
	<-tokens.started

	patient := make(chan string, 1)
	go func() {
		token, err := c.oauthTokenSource.token(context.Background())
		if err != nil {
			t.Error(err)
		}
		patient <- token
	}()

	if err := <-impatient; !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("impatient caller err = %v, want context.DeadlineExceeded", err)
	}

	// The request the impatient caller started still serves the other caller.
	close(tokens.release)
	if got := <-patient; got != "token-1" {
		t.Errorf("patient caller token = %q, want token-1", got)
	}
	if got := tokens.requestCount(); got != 1 {
		t.Errorf("token requests = %d, want 1", got)
	}
}

func TestOAuthClientCredentialsRejected(t *testing.T) {
	tokens := newTokenServer(t)
	api := newAPIServer(t)

	c, err := NewClient(
		WithURL(api.URL),
		WithOAuthClientCredentials("client_123", "wrong"),
		WithOAuthTokenURL(tokens.URL),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.ReadConversation(context.Background(), "conv-1", &ReadParams{})
	var te *OAuthTokenError
	if !errors.As(err, &te) {
		t.Fatalf("err = %v, want *OAuthTokenError", err)
	}
	if te.StatusCode != http.StatusUnauthorized || te.Code != "invalid_client" || te.Description != "Invalid client credentials." {
		t.Errorf("err = %+v", te)
	}
	if got := api.headers(); len(got) != 0 {
		t.Errorf("API was called %d times, want 0", len(got))
	}
}

type countingTransport struct{ requests int32 }

func (ct *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	atomic.AddInt32(&ct.requests, 1)
	return http.DefaultTransport.RoundTrip(r)
}

func TestOAuthTokenRequestUsesTransport(t *testing.T) {
	tokens := newTokenServer(t)
	transport := &countingTransport{}
	c := newOAuthClient(t, tokens.URL, WithTransport(transport))

	if _, err := c.oauthTokenSource.token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&transport.requests); got != 1 {
		t.Errorf("requests through transport = %d, want 1", got)
	}
}

func TestOAuthDefaultTokenURL(t *testing.T) {
	c, err := NewClient(WithOAuthClientCredentials("client_123", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.oauthTokenSource.tokenURL; got != defaultOAuthTokenURL {
		t.Errorf("token URL = %q, want %q", got, defaultOAuthTokenURL)
	}
}

func TestAPIKey(t *testing.T) {
	api := newAPIServer(t)

	c, err := NewClient(WithURL(api.URL), WithAPIKey("sk_123"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadConversation(context.Background(), "conv-1", &ReadParams{}); err != nil {
		t.Fatal(err)
	}
	if got := api.headers(); len(got) != 1 || got[0] != "Bearer sk_123" {
		t.Errorf("Authorization headers = %q", got)
	}
}

func TestNewClientAuthOptionErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opts    []Option
		wantErr string
	}{
		{
			name:    "API key and OAuth client credentials",
			opts:    []Option{WithAPIKey("sk_123"), WithOAuthClientCredentials("client_123", "secret")},
			wantErr: "use either WithAPIKey or an OAuth client, not both",
		},
		{
			name:    "token URL without OAuth client credentials",
			opts:    []Option{WithAPIKey("sk_123"), WithOAuthTokenURL("https://example.com/token")},
			wantErr: "WithOAuthTokenURL requires WithOAuthClientCredentials or WithOAuthPrivateKeyJWT",
		},
		{
			name:    "empty client ID",
			opts:    []Option{WithOAuthClientCredentials("", "secret")},
			wantErr: "WithOAuthClientCredentials requires a client ID and secret",
		},
		{
			name:    "empty client secret",
			opts:    []Option{WithOAuthClientCredentials("client_123", "")},
			wantErr: "WithOAuthClientCredentials requires a client ID and secret",
		},
		{
			name:    "empty token URL",
			opts:    []Option{WithOAuthClientCredentials("client_123", "secret"), WithOAuthTokenURL("")},
			wantErr: "WithOAuthTokenURL requires a URL",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewClient(tc.opts...)
			if err == nil || err.Error() != tc.wantErr {
				t.Errorf("err = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestOAuthTokenWaitsBeforeRetryingAFailure(t *testing.T) {
	tokens := newTokenServer(t)
	tokens.failWith(http.StatusServiceUnavailable, "")
	c := newOAuthClient(t, tokens.URL)
	src := c.oauthTokenSource
	now := time.Now()
	src.now = func() time.Time { return now }
	ctx := context.Background()

	// The delay doubles with each failure in a row: 1s, then 2s.
	for _, step := range []struct {
		advance  time.Duration
		requests int32
	}{
		{0, 1},
		{0, 1},
		{time.Second + time.Millisecond, 2},
		{time.Second, 2},
		{time.Second + time.Millisecond, 3},
	} {
		now = now.Add(step.advance)
		_, err := src.token(ctx)
		var te *OAuthTokenError
		if !errors.As(err, &te) || te.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("err = %v, want the 503 as *OAuthTokenError", err)
		}
		if got := tokens.requestCount(); got != step.requests {
			t.Fatalf("token requests = %d, want %d", got, step.requests)
		}
	}

	tokens.failWith(0, "")
	now = now.Add(5 * time.Second)
	if token, err := src.token(ctx); err != nil || token != "token-4" {
		t.Errorf("token, err = %q, %v, want token-4 once the token URL recovers", token, err)
	}
}

func TestOAuthTokenHonoursRetryAfter(t *testing.T) {
	tokens := newTokenServer(t)
	tokens.failWith(http.StatusTooManyRequests, "30")
	c := newOAuthClient(t, tokens.URL)
	src := c.oauthTokenSource
	now := time.Now()
	src.now = func() time.Time { return now }
	ctx := context.Background()

	_, err := src.token(ctx)
	var te *OAuthTokenError
	if !errors.As(err, &te) || te.RetryAfter != 30*time.Second {
		t.Fatalf("err = %v, want *OAuthTokenError with RetryAfter 30s", err)
	}
	for i := 0; i < 20; i++ {
		_, _ = src.token(ctx)
	}
	now = now.Add(29 * time.Second)
	_, _ = src.token(ctx)
	if got := tokens.requestCount(); got != 1 {
		t.Errorf("token requests within Retry-After = %d, want 1", got)
	}
	now = now.Add(2 * time.Second)
	_, _ = src.token(ctx)
	if got := tokens.requestCount(); got != 2 {
		t.Errorf("token requests after Retry-After = %d, want 2", got)
	}
}

func TestOAuthTokenFailedRefreshKeepsTheValidToken(t *testing.T) {
	tokens := newTokenServer(t)
	c := newOAuthClient(t, tokens.URL)
	src := c.oauthTokenSource
	now := time.Now()
	src.now = func() time.Time { return now }
	ctx := context.Background()

	if token, err := src.token(ctx); err != nil || token != "token-1" {
		t.Fatalf("token, err = %q, %v", token, err)
	}

	// The token lasts 300s and is renewed at 240s. A failed renewal at 250s
	// still serves it.
	tokens.failWith(http.StatusServiceUnavailable, "")
	now = now.Add(250 * time.Second)
	if token, err := src.token(ctx); err != nil || token != "token-1" {
		t.Errorf("token, err = %q, %v, want the still-valid token-1", token, err)
	}

	now = now.Add(51 * time.Second)
	if _, err := src.token(ctx); err == nil {
		t.Error("got a token after token-1 expired and the token URL kept failing")
	}
	if got := tokens.requestCount(); got != 3 {
		t.Errorf("token requests = %d, want 3", got)
	}
}

func TestOAuthUnauthorizedResponseDropsTheToken(t *testing.T) {
	tokens := newTokenServer(t)
	var calls int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"message":"Invalid Authorization header."}`)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-2" {
			t.Errorf("Authorization = %q, want a new token after the 401", got)
		}
		fmt.Fprint(w, `{"id":"conv-1"}`)
	}))
	t.Cleanup(api.Close)
	c := newOAuthClient(t, tokens.URL, WithURL(api.URL))

	ctx := context.Background()
	_, err := c.ReadConversation(ctx, "conv-1", &ReadParams{})
	var re *ResponseError
	if !errors.As(err, &re) || re.StatusCode != http.StatusUnauthorized {
		t.Fatalf("err = %v, want the API's 401", err)
	}
	if _, err := c.ReadConversation(ctx, "conv-1", &ReadParams{}); err != nil {
		t.Fatal(err)
	}
	if got := tokens.requestCount(); got != 2 {
		t.Errorf("token requests = %d, want 2", got)
	}
}

func TestOAuthTokenRequestDoesNotFollowRedirects(t *testing.T) {
	var elsewhere int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&elsewhere, 1)
		fmt.Fprint(w, `{"access_token":"other","expires_in":300}`)
	}))
	t.Cleanup(other.Close)
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirecting.Close)
	c := newOAuthClient(t, redirecting.URL)

	_, err := c.oauthTokenSource.token(context.Background())
	var te *OAuthTokenError
	if !errors.As(err, &te) || te.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("err = %v, want the 307 as *OAuthTokenError", err)
	}
	if got := atomic.LoadInt32(&elsewhere); got != 0 {
		t.Errorf("the redirect target got %d token requests, want 0", got)
	}
}

type panickingTransport struct{}

func (panickingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("transport bug")
}

func TestOAuthTokenRequestPanicIsAnError(t *testing.T) {
	c := newOAuthClient(t, "http://127.0.0.1:1/token", WithTransport(panickingTransport{}))
	_, err := c.oauthTokenSource.token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "panic: transport bug") {
		t.Errorf("err = %v, want the transport's panic as an error", err)
	}
}

func TestOAuthTokenRequestTimeoutIsNotTheCallersDeadline(t *testing.T) {
	tokens := newTokenServer(t)
	tokens.started = make(chan struct{}, 1)
	tokens.release = make(chan struct{})
	t.Cleanup(func() { close(tokens.release) })
	c := newOAuthClient(t, tokens.URL)
	c.oauthTokenSource.requestTimeout = 50 * time.Millisecond

	_, err := c.oauthTokenSource.token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no answer within 50ms") {
		t.Fatalf("err = %v, want the token request to time out", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Error("the internal timeout looks like the caller's own deadline")
	}
}
