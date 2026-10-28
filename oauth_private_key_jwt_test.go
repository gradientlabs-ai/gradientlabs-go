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
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// keyTokenServer is the API with a token endpoint at /oauth/token that accepts
// client assertions signed by one public key, and records every assertion.
type keyTokenServer struct {
	*httptest.Server
	public crypto.PublicKey
	alg    AssertionAlgorithm

	mu         sync.Mutex
	assertions []verifiedAssertion
	authHeader []string
}

type verifiedAssertion struct {
	header map[string]string
	claims clientAssertionClaims
}

func newKeyTokenServer(t *testing.T, public crypto.PublicKey, alg AssertionAlgorithm) *keyTokenServer {
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
		// The rules our token endpoint enforces on the request itself.
		form := r.PostForm
		if form.Get("grant_type") != "client_credentials" ||
			form.Get("client_assertion_type") != clientAssertionType ||
			form.Get("client_id") != "oauthclient_123" ||
			form.Has("client_secret") || form.Has("scope") ||
			r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected token request: form %v, Authorization %q", form, r.Header.Get("Authorization"))
		}
		for name, values := range form {
			if len(values) != 1 {
				t.Errorf("parameter %s sent %d times", name, len(values))
			}
		}
		a, err := verifyClientAssertion(form.Get("client_assertion"), ks.public, ks.alg)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
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
func verifyClientAssertion(raw string, public crypto.PublicKey, alg AssertionAlgorithm) (verifiedAssertion, error) {
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
	if a.header["alg"] != string(alg) {
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
		// RFC 7518 section 3.5 requires the salt to be as long as the hash.
		opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256}
		if err := rsa.VerifyPSS(public.(*rsa.PublicKey), crypto.SHA256, digest[:], sig, opts); err != nil {
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

// wrappedSigner hides the key's concrete type, as a KMS client's signer does.
type wrappedSigner struct{ crypto.Signer }

func TestOAuthPrivateKeyJWTKeyTypes(t *testing.T) {
	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rsaKey := newRSAKey(t, 2048)
	ecKey := newP256Key(t)

	for _, tc := range []struct {
		name string
		key  crypto.Signer
		opts []Option
		alg  AssertionAlgorithm
	}{
		{name: "EC P-256", key: ecKey, alg: AlgorithmES256},
		{name: "EC P-256 behind a wrapping crypto.Signer", key: wrappedSigner{ecKey}, alg: AlgorithmES256},
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
			if cl.IssuedAt < before.Unix() || cl.ExpiresAt-cl.IssuedAt != 300 {
				t.Errorf("iat = %d, exp = %d, want a 5 minute assertion issued now", cl.IssuedAt, cl.ExpiresAt)
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
	if te.StatusCode != http.StatusBadRequest || te.Code != "invalid_client" {
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
		{
			name:         "WithOAuthAudience behind a proxy",
			opts:         []Option{WithURL("https://egress.example.com/glabs"), WithOAuthAudience("https://api.gradient-labs.ai/oauth")},
			wantTokenURL: "https://egress.example.com/glabs/oauth/token",
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

// fakeSigner is a crypto.Signer for a real P-256 public key whose Sign does
// whatever the test needs.
type fakeSigner struct {
	public crypto.PublicKey
	sign   func() ([]byte, error)
	calls  int32
}

func (f *fakeSigner) Public() crypto.PublicKey { return f.public }

func (f *fakeSigner) Sign(io.Reader, []byte, crypto.SignerOpts) ([]byte, error) {
	atomic.AddInt32(&f.calls, 1)
	return f.sign()
}

func newFakeSignerClient(t *testing.T, sign func() ([]byte, error)) (*Client, *fakeSigner) {
	t.Helper()
	signer := &fakeSigner{public: newP256Key(t).Public(), sign: sign}
	c, err := NewClient(WithURL("http://127.0.0.1:1"), WithOAuthPrivateKeyJWT("oauthclient_123", signer, ""))
	if err != nil {
		t.Fatal(err)
	}
	return c, signer
}

func TestOAuthPrivateKeyJWTMalformedECDSASignature(t *testing.T) {
	encode := func(r, s *big.Int, trailing []byte) []byte {
		der, err := asn1.Marshal(struct{ R, S *big.Int }{r, s})
		if err != nil {
			t.Fatal(err)
		}
		return append(der, trailing...)
	}
	one := big.NewInt(1)
	for _, tc := range []struct {
		name string
		der  []byte
	}{
		{"R longer than 32 bytes", encode(new(big.Int).Lsh(one, 260), one, nil)},
		{"zero S", encode(one, big.NewInt(0), nil)},
		{"trailing bytes", encode(one, one, []byte{0})},
		{"not ASN.1", []byte("not a signature")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newFakeSignerClient(t, func() ([]byte, error) { return tc.der, nil })
			_, err := c.oauthTokenSource.token(context.Background())
			if err == nil || !strings.Contains(err.Error(), "ECDSA") {
				t.Errorf("err = %v, want an ECDSA signature error", err)
			}
		})
	}
}

func TestOAuthPrivateKeyJWTPanickingSigner(t *testing.T) {
	c, _ := newFakeSignerClient(t, func() ([]byte, error) { panic("kms client bug") })
	_, err := c.oauthTokenSource.token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "signer panicked: kms client bug") {
		t.Errorf("err = %v, want the signer's panic as an error", err)
	}
}

func TestOAuthPrivateKeyJWTHangingSigner(t *testing.T) {
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	c, signer := newFakeSignerClient(t, func() ([]byte, error) {
		<-block
		return nil, errors.New("released")
	})
	src := c.oauthTokenSource
	src.requestTimeout = 50 * time.Millisecond
	now := time.Now()
	src.now = func() time.Time { return now }

	_, err := src.token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no answer within 50ms") {
		t.Fatalf("err = %v, want the token request to time out", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Error("the internal timeout looks like the caller's own deadline")
	}

	// The stuck signer doesn't wedge the token source: after the retry delay
	// a new token request signs again.
	now = now.Add(2 * time.Second)
	if _, err := src.token(context.Background()); err == nil {
		t.Fatal("second token request succeeded with a hanging signer")
	}
	if got := atomic.LoadInt32(&signer.calls); got != 2 {
		t.Errorf("Sign calls = %d, want 2", got)
	}
}

func TestOAuthPrivateKeyJWTOptionErrors(t *testing.T) {
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	p256 := newP256Key(t)
	var typedNil *ecdsa.PrivateKey

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
			name:    "typed nil key",
			opts:    []Option{WithOAuthPrivateKeyJWT("oauthclient_123", typedNil, "")},
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
			name:    "empty algorithm",
			opts:    []Option{WithOAuthPrivateKeyJWT("oauthclient_123", p256, ""), WithOAuthAssertionAlgorithm("")},
			wantErr: "WithOAuthAssertionAlgorithm requires an algorithm",
		},
		{
			name:    "algorithm without a private key",
			opts:    []Option{WithOAuthClientCredentials("client_123", "secret"), WithOAuthAssertionAlgorithm(AlgorithmPS256)},
			wantErr: "WithOAuthAssertionAlgorithm requires WithOAuthPrivateKeyJWT",
		},
		{
			name:    "audience without a private key",
			opts:    []Option{WithOAuthClientCredentials("client_123", "secret"), WithOAuthAudience("https://api.gradient-labs.ai/oauth")},
			wantErr: "WithOAuthAudience requires WithOAuthPrivateKeyJWT",
		},
		{
			name:    "empty audience",
			opts:    []Option{WithOAuthPrivateKeyJWT("oauthclient_123", p256, ""), WithOAuthAudience("")},
			wantErr: "WithOAuthAudience requires an audience",
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
