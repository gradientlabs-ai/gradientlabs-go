package client

import (
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
	"math/big"
	"time"
)

const (
	clientAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

	// clientAssertionLifetime is how long each signed assertion is valid for.
	// A fresh one is signed for every token request and each can only be used
	// once, so a few minutes costs nothing and tolerates a slow client clock.
	clientAssertionLifetime = 5 * time.Minute

	// oauthIssuerPath is where, under the API URL, the OAuth issuer for
	// clients that authenticate with a private key lives. Its token endpoint is
	// the issuer followed by /token.
	oauthIssuerPath = "/oauth"
)

// AssertionAlgorithm is an algorithm a client assertion can be signed with. It
// must match the one the public key was registered with.
type AssertionAlgorithm string

const (
	AlgorithmES256 AssertionAlgorithm = "ES256"
	AlgorithmEdDSA AssertionAlgorithm = "EdDSA"
	AlgorithmRS256 AssertionAlgorithm = "RS256"
	AlgorithmPS256 AssertionAlgorithm = "PS256"
)

// assertionSigner signs RFC 7523 client assertions for the private_key_jwt
// client authentication method.
type assertionSigner struct {
	clientID string
	key      crypto.Signer
	keyID    string
	alg      AssertionAlgorithm
	// audience is our OAuth issuer by default, which FAPI 2.0 and the RFC 7523
	// update require as the assertion's aud in place of the token URL.
	audience string
}

// newAssertionSigner signs with alg, or with the key's default algorithm when
// alg is empty.
func newAssertionSigner(o oauthPrivateKeyJWTOption, alg AssertionAlgorithm, audience string) (*assertionSigner, error) {
	if o.clientID == "" || o.key == nil {
		return nil, errors.New("WithOAuthPrivateKeyJWT requires a client ID and private key")
	}
	public, err := publicKeyOf(o.key)
	if err != nil {
		return nil, err
	}
	allowed, err := assertionAlgorithms(public)
	if err != nil {
		return nil, err
	}
	if alg == "" {
		alg = allowed[0]
	}
	if !containsAlgorithm(allowed, alg) {
		return nil, fmt.Errorf("algorithm %s does not match the private key; use one of %v", alg, allowed)
	}
	return &assertionSigner{
		clientID: o.clientID,
		key:      o.key,
		keyID:    o.keyID,
		audience: audience,
		alg:      alg,
	}, nil
}

// publicKeyOf returns an error rather than panicking for a typed nil key, such
// as a nil *ecdsa.PrivateKey passed as a crypto.Signer.
func publicKeyOf(key crypto.Signer) (public crypto.PublicKey, err error) {
	defer func() {
		if recover() != nil {
			public, err = nil, errors.New("WithOAuthPrivateKeyJWT requires a client ID and private key")
		}
	}()
	return key.Public(), nil
}

// assertionAlgorithms returns the algorithms a key can sign with, the default
// first.
func assertionAlgorithms(public crypto.PublicKey) ([]AssertionAlgorithm, error) {
	switch k := public.(type) {
	case *ecdsa.PublicKey:
		if k.Curve != elliptic.P256() {
			return nil, errors.New("WithOAuthPrivateKeyJWT requires an EC key on the P-256 curve")
		}
		return []AssertionAlgorithm{AlgorithmES256}, nil
	case ed25519.PublicKey:
		return []AssertionAlgorithm{AlgorithmEdDSA}, nil
	case *rsa.PublicKey:
		if k.N.BitLen() < 2048 {
			return nil, errors.New("WithOAuthPrivateKeyJWT requires an RSA key of at least 2048 bits")
		}
		return []AssertionAlgorithm{AlgorithmRS256, AlgorithmPS256}, nil
	default:
		return nil, fmt.Errorf("WithOAuthPrivateKeyJWT does not support %T keys; use EC P-256, Ed25519 or RSA", public)
	}
}

type clientAssertionClaims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Audience  string `json:"aud"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	JWTID     string `json:"jti"`
}

// sign returns a compact JWT naming the client as issuer and subject and our
// issuer as audience, with a random jti so it can only be used once.
func (s *assertionSigner) sign(now time.Time) (string, error) {
	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return "", fmt.Errorf("client assertion: generating jti: %w", err)
	}

	header := map[string]string{"alg": string(s.alg), "typ": "JWT"}
	if s.keyID != "" {
		header["kid"] = s.keyID
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("client assertion: encoding header: %w", err)
	}
	claimsJSON, err := json.Marshal(clientAssertionClaims{
		Issuer:    s.clientID,
		Subject:   s.clientID,
		Audience:  s.audience,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(clientAssertionLifetime).Unix(),
		JWTID:     base64.RawURLEncoding.EncodeToString(jti),
	})
	if err != nil {
		return "", fmt.Errorf("client assertion: encoding claims: %w", err)
	}

	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature, err := s.signature([]byte(signingInput))
	if err != nil {
		return "", fmt.Errorf("client assertion: signing: %w", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *assertionSigner) signature(input []byte) ([]byte, error) {
	if s.alg == AlgorithmEdDSA {
		return s.key.Sign(rand.Reader, input, crypto.Hash(0))
	}

	digest := sha256.Sum256(input)
	switch s.alg {
	case AlgorithmRS256:
		return s.key.Sign(rand.Reader, digest[:], crypto.SHA256)
	case AlgorithmPS256:
		return s.key.Sign(rand.Reader, digest[:], &rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
			Hash:       crypto.SHA256,
		})
	case AlgorithmES256:
		der, err := s.key.Sign(rand.Reader, digest[:], crypto.SHA256)
		if err != nil {
			return nil, err
		}
		// crypto.Signer returns an ASN.1 signature, but JWS wants R and S
		// as two fixed-size big-endian integers.
		var sig struct{ R, S *big.Int }
		rest, err := asn1.Unmarshal(der, &sig)
		if err != nil {
			return nil, fmt.Errorf("decoding ECDSA signature: %w", err)
		}
		// A custom signer's output is checked, as FillBytes panics on an
		// integer longer than 32 bytes.
		if len(rest) != 0 || !isP256Scalar(sig.R) || !isP256Scalar(sig.S) {
			return nil, errors.New("the signer returned a malformed ECDSA P-256 signature")
		}
		out := make([]byte, 64)
		sig.R.FillBytes(out[:32])
		sig.S.FillBytes(out[32:])
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported algorithm %s", s.alg)
	}
}

func isP256Scalar(n *big.Int) bool {
	return n != nil && n.Sign() > 0 && n.BitLen() <= 256
}

func containsAlgorithm(values []AssertionAlgorithm, v AssertionAlgorithm) bool {
	for _, value := range values {
		if value == v {
			return true
		}
	}
	return false
}
