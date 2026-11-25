package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type UpdateToolParams struct {
	// ID identifies the tool to update. It is sent in the URL, not the body.
	ID          string          `json:"-"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters,omitempty"`

	// ParameterSets replaces the tool's conditional groups of parameters. An
	// empty list removes them all, and leaving it out keeps them.
	ParameterSets *[]ToolParameterSet `json:"parameter_sets,omitempty"`

	// Webhook, HTTP and Async hold the new configuration for a tool of that
	// type. Leave them out to keep the current configuration, and send at most
	// one.
	Webhook *ToolWebhookConfiguration `json:"webhook,omitempty"`
	HTTP    *HTTPDefinition           `json:"http,omitempty"`
	Async   *AsyncDefinition          `json:"async,omitempty"`
}

// UpdateTool creates a new revision of a tool. The name and the type of the
// tool cannot be changed.
//
// Note: requires a `Management` API key.
func (c *Client) UpdateTool(ctx context.Context, p *UpdateToolParams) (*Tool, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPut, fmt.Sprintf("tools/%s", p.ID), p)
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
