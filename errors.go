package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ResponseError represents an error response from the API.
type ResponseError struct {
	StatusCode int
	Message    string
	Details    map[string]any
}

// Error satisfies the error interface.
func (re *ResponseError) Error() string {
	b := new(strings.Builder)

	if re.Message == "" {
		_, _ = fmt.Fprintf(b, "unexpected response status: %d", re.StatusCode)
	} else {
		_, _ = b.WriteString(re.Message)
	}

	if traceID := re.TraceID(); traceID != "" {
		_, _ = fmt.Fprintf(b, " (trace id: %s)", traceID)
	}

	return b.String()
}

// TraceID returns the identifier that can be given to Gradient Labs technical
// support to investigate an error.
func (re *ResponseError) TraceID() string {
	if re.Details == nil {
		return ""
	}

	traceID, ok := re.Details["trace_id"].(string)
	if !ok {
		return ""
	}
	return traceID
}

func responseError(rsp *http.Response) *ResponseError {
	if rsp.StatusCode < 200 || rsp.StatusCode > 299 {
		re := &ResponseError{StatusCode: rsp.StatusCode}

		var payload struct {
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		}
		if err := json.NewDecoder(rsp.Body).Decode(&payload); err == nil {
			re.Message = payload.Message
			re.Details = payload.Details
		}
		return re
	}
	return nil
}

// OAuthTokenError is returned when the token URL refuses to issue an access
// token, for example because the client ID or secret is wrong or the OAuth
// client was revoked.
type OAuthTokenError struct {
	StatusCode int

	// Code and Description are the OAuth error and error_description fields
	// of the response, if it had them.
	Code        string
	Description string
}

// Error satisfies the error interface.
func (te *OAuthTokenError) Error() string {
	b := new(strings.Builder)
	_, _ = fmt.Fprintf(b, "oauth token request failed with status %d", te.StatusCode)
	if te.Code != "" {
		_, _ = fmt.Fprintf(b, ": %s", te.Code)
	}
	if te.Description != "" {
		_, _ = fmt.Fprintf(b, " (%s)", te.Description)
	}
	return b.String()
}
