// Package client implements Go bindings for the [Gradient Labs API].
//
// [Gradient Labs API]: https://api-docs.gradient-labs.ai
package client

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const defaultURL = "https://api.gradient-labs.ai"

var userAgent = fmt.Sprintf("Gradient-Labs-Go (%s/%s)", version, runtime.Version())

// Client provides access to the Gradient Labs API. Use NewClient to create one.
type Client struct {
	url, apiKey      string
	oauthTokenSource *oauthTokenSource
	httpClient       *http.Client
	webhookVerifier  *WebhookVerifier
}

// NewClient creates a client with the given options.
//
// Note: to call the API, set one of WithAPIKey, WithOAuthClientCredentials or
// WithOAuthPrivateKeyJWT.
// WithWebhookSigningKey is required if you intend to receive webhooks.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		url:        defaultURL,
		httpClient: &http.Client{},
		webhookVerifier: &WebhookVerifier{
			leeway: defaultLeeway,
		},
	}

	var oauth *oauthClientCredentialsOption
	var privateKey *oauthPrivateKeyJWTOption
	var assertionAlgorithm *AssertionAlgorithm
	var oauthAudience *string
	var oauthTokenURL *string
	for _, opt := range opts {
		switch t := opt.(type) {
		case urlOption:
			c.url = t.url
		case apiKeyOption:
			c.apiKey = t.apiKey
		case oauthClientCredentialsOption:
			oauth = &t
		case oauthPrivateKeyJWTOption:
			privateKey = &t
		case oauthAssertionAlgorithmOption:
			assertionAlgorithm = &t.alg
		case oauthAudienceOption:
			oauthAudience = &t.audience
		case oauthTokenURLOption:
			oauthTokenURL = &t.url
		case transportOption:
			c.httpClient.Transport = t.transport
		case webhookSigningKeyOption:
			c.webhookVerifier.secret = t.signingKey
		case webhookLeewayOption:
			c.webhookVerifier.leeway = t.leeway
		}
	}

	if assertionAlgorithm != nil && privateKey == nil {
		return nil, errors.New("WithOAuthAssertionAlgorithm requires WithOAuthPrivateKeyJWT")
	}
	if assertionAlgorithm != nil && *assertionAlgorithm == "" {
		return nil, errors.New("WithOAuthAssertionAlgorithm requires an algorithm")
	}
	if oauthAudience != nil && privateKey == nil {
		return nil, errors.New("WithOAuthAudience requires WithOAuthPrivateKeyJWT")
	}
	if oauthAudience != nil && *oauthAudience == "" {
		return nil, errors.New("WithOAuthAudience requires an audience")
	}
	if oauth == nil && privateKey == nil {
		if oauthTokenURL != nil {
			return nil, errors.New("WithOAuthTokenURL requires WithOAuthClientCredentials or WithOAuthPrivateKeyJWT")
		}
		return c, nil
	}
	if oauth != nil && privateKey != nil {
		return nil, errors.New("use either WithOAuthClientCredentials or WithOAuthPrivateKeyJWT, not both")
	}
	if c.apiKey != "" {
		return nil, errors.New("use either WithAPIKey or an OAuth client, not both")
	}
	if oauthTokenURL != nil && *oauthTokenURL == "" {
		return nil, errors.New("WithOAuthTokenURL requires a URL")
	}

	tokenHTTPClient := &http.Client{
		Transport: c.httpClient.Transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	if privateKey != nil {
		issuer := strings.TrimSuffix(c.url, "/") + oauthIssuerPath
		audience := issuer
		if oauthAudience != nil {
			audience = *oauthAudience
		}
		var alg AssertionAlgorithm
		if assertionAlgorithm != nil {
			alg = *assertionAlgorithm
		}
		signer, err := newAssertionSigner(*privateKey, alg, audience)
		if err != nil {
			return nil, err
		}
		tokenURL := issuer + "/token"
		if oauthTokenURL != nil {
			tokenURL = *oauthTokenURL
		}
		c.oauthTokenSource = &oauthTokenSource{
			tokenURL:       tokenURL,
			clientID:       privateKey.clientID,
			signer:         signer,
			httpClient:     tokenHTTPClient,
			now:            time.Now,
			requestTimeout: tokenRequestTimeout,
		}
		return c, nil
	}

	if oauth.clientID == "" || oauth.clientSecret == "" {
		return nil, errors.New("WithOAuthClientCredentials requires a client ID and secret")
	}
	tokenURL := defaultOAuthTokenURL
	if oauthTokenURL != nil {
		tokenURL = *oauthTokenURL
	}
	c.oauthTokenSource = &oauthTokenSource{
		tokenURL:       tokenURL,
		clientID:       oauth.clientID,
		clientSecret:   oauth.clientSecret,
		httpClient:     tokenHTTPClient,
		now:            time.Now,
		requestTimeout: tokenRequestTimeout,
	}
	return c, nil
}

// WithURL overrides the default base URL.
func WithURL(url string) Option {
	return urlOption{url}
}

// WithTransport customises the HTTP client's Transport, which is useful for
// observability (e.g. capturing trace spaces, metrics) and in unit tests.
func WithTransport(rt http.RoundTripper) Option {
	return transportOption{rt}
}

// WithAPIKey sets the client's API key.
func WithAPIKey(key string) Option {
	return apiKeyOption{key}
}

// WithOAuthClientCredentials makes the client authenticate with an OAuth
// client instead of an API key. The client fetches access tokens from the
// token URL with the client credentials grant, and renews them before they
// expire.
func WithOAuthClientCredentials(clientID, clientSecret string) Option {
	return oauthClientCredentialsOption{clientID, clientSecret}
}

// WithOAuthPrivateKeyJWT authenticates with an OAuth client that holds a
// private key instead of a secret (private_key_jwt, RFC 7523), signing a fresh
// client assertion for each token request. key is an EC P-256, Ed25519 or RSA
// (2048 bits or more) private key, or a crypto.Signer for one held in a KMS.
// keyID is the public key's kid, and may be empty when the OAuth client has one
// key. Tokens come from the API URL plus /oauth/token, so WithURL moves it too.
func WithOAuthPrivateKeyJWT(clientID string, key crypto.Signer, keyID string) Option {
	return oauthPrivateKeyJWTOption{clientID, key, keyID}
}

// WithOAuthAssertionAlgorithm sets the algorithm WithOAuthPrivateKeyJWT signs
// with. It only needs setting for an RSA key registered for PS256 (use
// AlgorithmPS256); otherwise the algorithm follows from the key: ES256 for EC
// P-256, EdDSA for Ed25519 and RS256 for RSA.
func WithOAuthAssertionAlgorithm(alg AssertionAlgorithm) Option {
	return oauthAssertionAlgorithmOption{alg}
}

// WithOAuthAudience overrides the aud of the client assertions that
// WithOAuthPrivateKeyJWT signs. It defaults to the API URL plus /oauth, so it
// only needs setting when WithURL is not the API's own URL, for example a proxy.
func WithOAuthAudience(audience string) Option {
	return oauthAudienceOption{audience}
}

// WithOAuthTokenURL overrides the default URL that access tokens are fetched
// from when using WithOAuthClientCredentials or WithOAuthPrivateKeyJWT. If you
// use WithURL to call a different API URL with WithOAuthClientCredentials, set
// the token URL that goes with it too. With WithOAuthPrivateKeyJWT it doesn't
// change the assertions' aud; see WithOAuthAudience.
func WithOAuthTokenURL(url string) Option {
	return oauthTokenURLOption{url}
}

// WithWebhookSigningKey sets the client's webhook signing key. It will be used
// to verify webhook authenticity.
func WithWebhookSigningKey(key string) Option {
	return webhookSigningKeyOption{[]byte(key)}
}

// WithWebhookLeeway determines the maximum age of webhook that will be accepted
// by Client.VerifyWebhookRequest.
func WithWebhookLeeway(leeway time.Duration) Option {
	return webhookLeewayOption{leeway}
}

type urlOption struct{ url string }
type transportOption struct{ transport http.RoundTripper }
type apiKeyOption struct{ apiKey string }
type oauthClientCredentialsOption struct{ clientID, clientSecret string }
type oauthPrivateKeyJWTOption struct {
	clientID string
	key      crypto.Signer
	keyID    string
}
type oauthAssertionAlgorithmOption struct{ alg AssertionAlgorithm }
type oauthAudienceOption struct{ audience string }
type oauthTokenURLOption struct{ url string }
type webhookSigningKeyOption struct{ signingKey []byte }
type webhookLeewayOption struct{ leeway time.Duration }

func (urlOption) isClientOption()                     {}
func (transportOption) isClientOption()               {}
func (apiKeyOption) isClientOption()                  {}
func (oauthClientCredentialsOption) isClientOption()  {}
func (oauthPrivateKeyJWTOption) isClientOption()      {}
func (oauthAssertionAlgorithmOption) isClientOption() {}
func (oauthAudienceOption) isClientOption()           {}
func (oauthTokenURLOption) isClientOption()           {}
func (webhookSigningKeyOption) isClientOption()       {}
func (webhookLeewayOption) isClientOption()           {}

// Option customises the Client.
type Option interface{ isClientOption() }

func (c *Client) makeRequest(ctx context.Context, method string, path string, body any) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", c.url, path)

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	token, err := c.bearerToken(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", userAgent)

	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	rsp, err := c.httpClient.Do(req)
	if err == nil && rsp.StatusCode == http.StatusUnauthorized && c.oauthTokenSource != nil {
		c.oauthTokenSource.invalidate(token)
	}
	return rsp, err
}

func (c *Client) bearerToken(ctx context.Context) (string, error) {
	if c.oauthTokenSource != nil {
		return c.oauthTokenSource.token(ctx)
	}
	return c.apiKey, nil
}
