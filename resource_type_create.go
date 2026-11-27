package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// ResourceTypeCreateParams represents the request to create a new resource type.
type ResourceTypeCreateParams struct {
	DisplayName     string          `json:"display_name"`
	Description     string          `json:"description,omitempty"`
	Scope           Scope           `json:"scope"`
	RefreshStrategy RefreshStrategy `json:"refresh_strategy"`
	SourceConfig    *SourceConfig   `json:"source_config,omitempty"`
	IsEnabled       bool            `json:"is_enabled,omitempty"`

	// BranchID is the ID of the branch to create the resource type on. Send
	// this or BranchName. The resource type goes live when the branch is
	// merged, and the returned ID is empty until the branch has taken it in.
	BranchID string `json:"branch_id,omitempty"`

	// BranchName is the git branch name of the branch to create on, as an
	// alternative to BranchID.
	BranchName string `json:"branch_name,omitempty"`
}

// CreateResourceType creates a new resource type.
//
// Note: requires a `Management` API key.
func (c *Client) CreateResourceType(ctx context.Context, req *ResourceTypeCreateParams) (*ResourceType, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "resource-types", req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var rt ResourceType
	if err := json.NewDecoder(rsp.Body).Decode(&rt); err != nil {
		return nil, err
	}
	return &rt, nil
}
