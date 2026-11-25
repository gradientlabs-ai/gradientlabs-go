package client

import "time"

// Secret represents a stored secret with optional expiration and refresh configuration.
type Secret struct {
	// Name is the unique identifier for the secret.
	Name string `json:"name"`

	// Created is when the secret was first created.
	Created time.Time `json:"created"`

	// Updated is when the secret was last updated.
	Updated time.Time `json:"updated"`

	// Expiry is the optional expiration time for the secret.
	Expiry *time.Time `json:"expiry,omitempty"`

	// RefreshMechanismHTTP is the optional configuration for automatically refreshing
	// the secret value using an HTTP request (e.g., OAuth token refresh).
	RefreshMechanismHTTP *RefreshMechanismHTTP `json:"refresh_mechanism_http,omitempty"`
}

// RefreshMechanismHTTP defines how to automatically refresh a secret's value
// using an HTTP request. This is commonly used for OAuth access tokens that
// need periodic renewal.
type RefreshMechanismHTTP struct {
	// RequestDefinition specifies the HTTP request to make to refresh the secret.
	RequestDefinition SecretRefreshRequest `json:"request_definition"`

	// ResponseParamName is the JSON field name in the response that contains
	// the new secret value.
	ResponseParamName string `json:"response_param_name"`
}

// SecretRefreshRequest describes the HTTP call that refreshes a secret.
type SecretRefreshRequest struct {
	Method          string              `json:"method"`
	URLTemplate     string              `json:"url_template"`
	HeaderTemplates map[string]string   `json:"header_templates,omitempty"`
	Body            *HTTPBodyDefinition `json:"body,omitempty"`
	Retry           *SecretRefreshRetry `json:"retry,omitempty"`
}

type SecretRefreshRetry struct {
	RequestTimeoutSeconds *int `json:"request_timeout_seconds,omitempty"`
}
