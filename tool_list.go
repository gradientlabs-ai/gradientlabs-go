package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// ListTools lists company tools and enabled MCP tools. Interactive tools are omitted.
//
// Note: requires a `Management` API key.
func (c *Client) ListTools(ctx context.Context) ([]*Tool, error) {
	rsp, err := c.makeRequest(ctx, http.MethodGet, "tools", nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var toolList ToolList
	if err := json.NewDecoder(rsp.Body).Decode(&toolList); err != nil {
		return nil, err
	}

	return toolList.Tools, nil
}
