package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// ResourceSourceCreateParams represents the request to create a new resource source.
type ResourceSourceCreateParams struct {
	DisplayName           string                     `json:"display_name"`
	Description           string                     `json:"description,omitempty"`
	SourceType            SourceType                 `json:"source_type"`
	HTTPConfig            *ResourceHTTPDefinition    `json:"http_config,omitempty"`
	WebhookConfig         *ResourceWebhookDefinition `json:"webhook_config,omitempty"`
	AttributeDescriptions map[string]string          `json:"attribute_descriptions,omitempty"`

	// BranchID is the ID of the branch to create the resource source on. Send
	// this or BranchName. The resource source goes live when the branch is
	// merged, and the returned ID is empty until the branch has taken it in.
	BranchID string `json:"branch_id,omitempty"`

	// BranchName is the git branch name of the branch to create on, as an
	// alternative to BranchID.
	BranchName string `json:"branch_name,omitempty"`
}

// CreateResourceSource creates a new resource source.
//
// Note: requires a `Management` API key.
func (c *Client) CreateResourceSource(ctx context.Context, req *ResourceSourceCreateParams) (*ResourceSource, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "resource-sources", req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var rs ResourceSource
	if err := json.NewDecoder(rsp.Body).Decode(&rs); err != nil {
		return nil, err
	}
	return &rs, nil
}
