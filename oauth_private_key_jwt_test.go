package client

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// keyTokenServer is the API with a token endpoint at /oauth/token that accepts
// client assertions signed by one public key, and records every assertion.
type keyTokenServer struct {
	*httptest.Server
	public crypto.PublicKey
	alg    string

	mu         sync.Mutex
	assertions []verifiedAssertion
	authHeader []string
}

type verifiedAssertion struct {
	header map[string]string
	claims clientAssertionClaims
}

func newKeyTokenServer(t *testing.T, public crypto.PublicKey, alg string) *keyTokenServer {
	t.Helper()
	ks := &keyTokenServer{public: public, alg: alg}
	ks.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			ks.mu.Lock()
			ks.authHeader = append(ks.authHeader, r.Header.Get("Authorization"))
			ks.mu.Unlock()
			fmt.Fprint(w, `{"id":"conv-1"}`)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		form := r.PostForm
		if form.Get("grant_type") != "client_credentials" ||
			form.Get("client_assertion_type") != clientAssertionType ||
			form.Get("client_id") != "oauthclient_123" ||
			form.Has("client_secret") {
			t.Errorf("unexpected token request form: %v", form)
		}
		a, err := verifyClientAssertion(form.Get("client_assertion"), ks.public, ks.alg)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"invalid_client","error_description":"Client authentication failed."}`)
			return
		}
		ks.mu.Lock()
		ks.assertions = append(ks.assertions, a)
		n := len(ks.assertions)
		ks.mu.Unlock()
		fmt.Fprintf(w, `{"access_token":"token-%d","token_type":"Bearer","expires_in":3600}`, n)
	}))
	t.Cleanup(ks.Close)
	return ks
}

func (ks *keyTokenServer) received() ([]verifiedAssertion, []string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	return append([]verifiedAssertion(nil), ks.assertions...), append([]string(nil), ks.authHeader...)
}

// verifyClientAssertion checks a compact JWS the way a token endpoint would,
// with the standard library only.
func verifyClientAssertion(raw string, public crypto.PublicKey, alg string) (verifiedAssertion, error) {
	var a verifiedAssertion
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return a, errors.New("not a compact JWS")
	}
	decode := func(s string, v any) error {
		b, err := base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, v)
	}
	if err := decode(parts[0], &a.header); err != nil {
		return a, err
	}
	if err := decode(parts[1], &a.claims); err != nil {
		return a, err
	}
	if a.header["alg"] != alg {
		return a, fmt.Errorf("alg %q, want %q", a.header["alg"], alg)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return a, err
	}

	input := []byte(parts[0] + "." + parts[1])
	digest := sha256.Sum256(input)
	switch alg {
	case AlgorithmES256:
		if len(sig) != 64 || !ecdsa.Verify(public.(*ecdsa.PublicKey), digest[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
			return a, errors.New("bad ES256 signature")
		}
	case AlgorithmEdDSA:
		if !ed25519.Verify(public.(ed25519.PublicKey), input, sig) {
			return a, errors.New("bad EdDSA signature")
		}
	case AlgorithmRS256:
		if err := rsa.VerifyPKCS1v15(public.(*rsa.PublicKey), crypto.SHA256, digest[:], sig); err != nil {
			return a, err
		}
	case AlgorithmPS256:
		if err := rsa.VerifyPSS(public.(*rsa.PublicKey), crypto.SHA256, digest[:], sig, nil); err != nil {
			return a, err
		}
	default:
		return a, fmt.Errorf("unexpected alg %q", alg)
	}
	return a, nil
}

func newRSAKey(t *testing.T, bits int) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func newP256Key(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestOAuthPrivateKeyJWTKeyTypes(t *testing.T) {
	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rsaKey := newRSAKey(t, 2048)

	for _, tc := range []struct {
		name string
		key  crypto.Signer
		opts []Option
		alg  string
	}{
		{name: "EC P-256", key: newP256Key(t), alg: AlgorithmES256},
		{name: "Ed25519", key: edKey, alg: AlgorithmEdDSA},
		{name: "RSA", key: rsaKey, alg: AlgorithmRS256},
		{name: "RSA-PSS", key: rsaKey, opts: []Option{WithOAuthAssertionAlgorithm(AlgorithmPS256)}, alg: AlgorithmPS256},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := newKeyTokenServer(t, tc.key.Public(), tc.alg)
			c, err := NewClient(append(tc.opts,
				WithURL(api.URL),
				WithOAuthPrivateKeyJWT("oauthclient_123", tc.key, "key-1"),
			)...)
			if err != nil {
				t.Fatal(err)
			}

			before := time.Now()
			if _, err := c.ReadConversation(context.Background(), "conv-1", &ReadParams{}); err != nil {
				t.Fatal(err)
			}

			assertions, headers := api.received()
			if len(assertions) != 1 {
				t.Fatalf("token requests = %d, want 1", len(assertions))
			}
			a := assertions[0]
			if a.header["kid"] != "key-1" || a.header["typ"] != "JWT" {
				t.Errorf("header = %v", a.header)
			}
			cl := a.claims
			if cl.Issuer != "oauthclient_123" || cl.Subject != "oauthclient_123" {
				t.Errorf("iss = %q, sub = %q, want the client ID", cl.Issuer, cl.Subject)
			}
			if want := api.URL + "/oauth"; cl.Audience != want {
				t.Errorf("aud = %q, want the issuer %q", cl.Audience, want)
			}
			if cl.IssuedAt < before.Unix() || cl.ExpiresAt-cl.IssuedAt != 60 {
				t.Errorf("iat = %d, exp = %d, want a 60s assertion issued now", cl.IssuedAt, cl.ExpiresAt)
			}
			if cl.JWTID == "" {
				t.Error("jti is empty")
			}
			if len(headers) != 1 || headers[0] != "Bearer token-1" {
				t.Errorf("Authorization headers = %q", headers)
			}
		})
	}
}

func TestOAuthPrivateKeyJWTSignsAFreshAssertionPerToken(t *testing.T) {
	key := newP256Key(t)
	api := newKeyTokenServer(t, key.Public(), AlgorithmES256)
	c, err := NewClient(WithURL(api.URL), WithOAuthPrivateKeyJWT("oauthclient_123", key, ""))
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	c.oauthTokenSource.now = func() time.Time { return now }

	ctx := context.Background()
	// The token lasts an hour and is renewed a minute early, at 59 minutes.
	var tokens []string
	for _, step := range []time.Duration{0, 59*time.Minute - time.Second, 2 * time.Second} {
		now = now.Add(step)
		token, err := c.oauthTokenSource.token(ctx)
		if err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, token)
	}
	if want := []string{"token-1", "token-1", "token-2"}; fmt.Sprint(tokens) != fmt.Sprint(want) {
		t.Errorf("tokens = %q, want %q", tokens, want)
	}

	assertions, _ := api.received()
	if len(assertions) != 2 {
		t.Fatalf("token requests = %d, want 2", len(assertions))
	}
	if assertions[0].claims.JWTID == assertions[1].claims.JWTID {
		t.Error("both token requests sent the same jti")
	}
	if _, ok := assertions[0].header["kid"]; ok {
		t.Error("kid was sent although none was set")
	}
	if got, want := assertions[1].claims.IssuedAt, now.Unix(); got != want {
		t.Errorf("second assertion iat = %d, want %d", got, want)
	}
}

func TestOAuthPrivateKeyJWTRejected(t *testing.T) {
	registered := newP256Key(t)
	api := newKeyTokenServer(t, registered.Public(), AlgorithmES256)
	c, err := NewClient(WithURL(api.URL), WithOAuthPrivateKeyJWT("oauthclient_123", newP256Key(t), ""))
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.ReadConversation(context.Background(), "conv-1", &ReadParams{})
	var te *OAuthTokenError
	if !errors.As(err, &te) {
		t.Fatalf("err = %v, want *OAuthTokenError", err)
	}
	if te.StatusCode != http.StatusUnauthorized || te.Code != "invalid_client" {
		t.Errorf("err = %+v", te)
	}
	if _, headers := api.received(); len(headers) != 0 {
		t.Errorf("API was called %d times, want 0", len(headers))
	}
}

func TestOAuthPrivateKeyJWTTokenURL(t *testing.T) {
	key := newP256Key(t)
	for _, tc := range []struct {
		name         string
		opts         []Option
		wantTokenURL string
		wantAudience string
	}{
		{
			name:         "default",
			wantTokenURL: "https://api.gradient-labs.ai/oauth/token",
			wantAudience: "https://api.gradient-labs.ai/oauth",
		},
		{
			name:         "follows WithURL",
			opts:         []Option{WithURL("http://localhost:4000/")},
			wantTokenURL: "http://localhost:4000/oauth/token",
			wantAudience: "http://localhost:4000/oauth",
		},
		{
			name:         "WithOAuthTokenURL moves only the token URL",
			opts:         []Option{WithOAuthTokenURL("https://proxy.example.com/token")},
			wantTokenURL: "https://proxy.example.com/token",
			wantAudience: "https://api.gradient-labs.ai/oauth",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewClient(append(tc.opts, WithOAuthPrivateKeyJWT("oauthclient_123", key, ""))...)
			if err != nil {
				t.Fatal(err)
			}
			if got := c.oauthTokenSource.tokenURL; got != tc.wantTokenURL {
				t.Errorf("token URL = %q, want %q", got, tc.wantTokenURL)
			}
			if got := c.oauthTokenSource.signer.audience; got != tc.wantAudience {
				t.Errorf("audience = %q, want %q", got, tc.wantAudience)
			}
		})
	}
}

func TestOAuthPrivateKeyJWTOptionErrors(t *testing.T) {
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	p256 := newP256Key(t)

	for _, tc := range []struct {
		name    string
		opts    []Option
		wantErr string
	}{
		{
			name:    "empty client ID",
			opts:    []Option{WithOAuthPrivateKeyJWT("", p256, "")},
			wantErr: "WithOAuthPrivateKeyJWT requires a client ID and private key",
		},
		{
			name:    "no key",
			opts:    []Option{WithOAuthPrivateKeyJWT("oauthclient_123", nil, "")},
			wantErr: "WithOAuthPrivateKeyJWT requires a client ID and private key",
		},
		{
			name:    "P-384 key",
			opts:    []Option{WithOAuthPrivateKeyJWT("oauthclient_123", p384, "")},
			wantErr: "WithOAuthPrivateKeyJWT requires an EC key on the P-256 curve",
		},
		{
			name:    "1024-bit RSA key",
			opts:    []Option{WithOAuthPrivateKeyJWT("oauthclient_123", newRSAKey(t, 1024), "")},
			wantErr: "WithOAuthPrivateKeyJWT requires an RSA key of at least 2048 bits",
		},
		{
			name:    "algorithm that doesn't match the key",
			opts:    []Option{WithOAuthPrivateKeyJWT("oauthclient_123", p256, ""), WithOAuthAssertionAlgorithm(AlgorithmPS256)},
			wantErr: "algorithm PS256 does not match the private key; use one of [ES256]",
		},
		{
			name:    "algorithm without a private key",
			opts:    []Option{WithOAuthClientCredentials("client_123", "secret"), WithOAuthAssertionAlgorithm(AlgorithmPS256)},
			wantErr: "WithOAuthAssertionAlgorithm requires WithOAuthPrivateKeyJWT",
		},
		{
			name:    "secret and private key",
			opts:    []Option{WithOAuthClientCredentials("client_123", "secret"), WithOAuthPrivateKeyJWT("oauthclient_123", p256, "")},
			wantErr: "use either WithOAuthClientCredentials or WithOAuthPrivateKeyJWT, not both",
		},
		{
			name:    "API key and private key",
			opts:    []Option{WithAPIKey("sk_123"), WithOAuthPrivateKeyJWT("oauthclient_123", p256, "")},
			wantErr: "use either WithAPIKey or an OAuth client, not both",
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
