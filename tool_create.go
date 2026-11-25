package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// CreateToolParams configures a new HTTP, webhook, async or draft tool.
type CreateToolParams struct {
	Name          string                    `json:"name"`
	Description   string                    `json:"description"`
	Parameters    []ToolParameter           `json:"parameters"`
	ParameterSets []ToolParameterSet        `json:"parameter_sets,omitempty"`
	HTTP          *HTTPDefinition           `json:"http,omitempty"`
	Webhook       *ToolWebhookConfiguration `json:"webhook,omitempty"`
	Async         *AsyncDefinition          `json:"async,omitempty"`
	Draft         bool                      `json:"draft,omitempty"`
	BranchID      string                    `json:"branch_id,omitempty"`
	BranchName    string                    `json:"branch_name,omitempty"`
}

// CreateTool creates a new tool.
//
// Note: requires a `Management` API key.
func (c *Client) CreateTool(ctx context.Context, p *CreateToolParams) (*Tool, error) {
	if p != nil && p.Parameters == nil {
		withEmptyParameters := *p
		withEmptyParameters.Parameters = []ToolParameter{}
		p = &withEmptyParameters
	}
	rsp, err := c.makeRequest(ctx, http.MethodPost, "tools", p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var tool Tool
	if err := json.NewDecoder(rsp.Body).Decode(&tool); err != nil {
		return nil, err
	}
	return &tool, nil
}
