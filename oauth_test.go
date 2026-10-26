package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newTokenServer(t *testing.T, requests *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.PostForm.Get("grant_type"); got != "client_credentials" {
			t.Errorf("grant_type = %q", got)
		}
		if r.PostForm.Get("client_id") != "client_123" || r.PostForm.Get("client_secret") != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"invalid_client","error_description":"Invalid client credentials."}`)
			return
		}
		n := atomic.AddInt32(requests, 1)
		fmt.Fprintf(w, `{"access_token":"token-%d","token_type":"Bearer","expires_in":300}`, n)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newAPIServer(t *testing.T, authHeaders *[]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*authHeaders = append(*authHeaders, r.Header.Get("Authorization"))
		fmt.Fprint(w, `{"id":"conv-1"}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestOAuthClientCredentials(t *testing.T) {
	var tokenRequests int32
	var authHeaders []string
	tokenSrv := newTokenServer(t, &tokenRequests)
	apiSrv := newAPIServer(t, &authHeaders)

	c, err := NewClient(
		WithURL(apiSrv.URL),
		WithOAuthClientCredentials("client_123", "secret"),
		WithOAuthTokenURL(tokenSrv.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	c.oauthTokenSource.now = func() time.Time { return now }

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := c.ReadConversation(ctx, "conv-1", &ReadParams{}); err != nil {
			t.Fatal(err)
		}
	}

	now = now.Add(4*time.Minute + time.Second)
	if _, err := c.ReadConversation(ctx, "conv-1", &ReadParams{}); err != nil {
		t.Fatal(err)
	}

	want := []string{"Bearer token-1", "Bearer token-1", "Bearer token-2"}
	if fmt.Sprint(authHeaders) != fmt.Sprint(want) {
		t.Errorf("Authorization headers = %q, want %q", authHeaders, want)
	}
	if tokenRequests != 2 {
		t.Errorf("token requests = %d, want 2", tokenRequests)
	}
}

func TestOAuthClientCredentialsRejected(t *testing.T) {
	var tokenRequests int32
	var authHeaders []string
	tokenSrv := newTokenServer(t, &tokenRequests)
	apiSrv := newAPIServer(t, &authHeaders)

	c, err := NewClient(
		WithURL(apiSrv.URL),
		WithOAuthClientCredentials("client_123", "wrong"),
		WithOAuthTokenURL(tokenSrv.URL),
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
	if len(authHeaders) != 0 {
		t.Errorf("API was called %d times, want 0", len(authHeaders))
	}
}

func TestAPIKey(t *testing.T) {
	var authHeaders []string
	apiSrv := newAPIServer(t, &authHeaders)

	c, err := NewClient(WithURL(apiSrv.URL), WithAPIKey("sk_123"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadConversation(context.Background(), "conv-1", &ReadParams{}); err != nil {
		t.Fatal(err)
	}
	if len(authHeaders) != 1 || authHeaders[0] != "Bearer sk_123" {
		t.Errorf("Authorization headers = %q", authHeaders)
	}
}

func TestNewClientAuthOptionErrors(t *testing.T) {
	if _, err := NewClient(WithAPIKey("sk_123"), WithOAuthClientCredentials("client_123", "secret")); err == nil {
		t.Error("API key and OAuth client credentials: want error")
	}
	if _, err := NewClient(WithAPIKey("sk_123"), WithOAuthTokenURL("https://example.com/token")); err == nil {
		t.Error("token URL without OAuth client credentials: want error")
	}
}
