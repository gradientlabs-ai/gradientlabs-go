package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ReadToolParams are the optional parameters to Client.ReadTool.
type ReadToolParams struct {
	// Version optionally specifies which tool revision to retrieve.
	// If zero, the latest version is returned.
	Version int
}

// ReadTool retrieves a tool. Enabled MCP tools can be read by ID. Interactive tools return 404.
//
// Pass a non-nil p to supply optional parameters such as a specific version.
// Pass nil (or a zero-value ReadToolParams) to retrieve the latest version.
//
// Note: requires a `Management` API key.
func (c *Client) ReadTool(ctx context.Context, toolID string, p *ReadToolParams) (*Tool, error) {
	path := fmt.Sprintf("tools/%s", toolID)
	if p != nil && p.Version > 0 {
		path = fmt.Sprintf("%s?version=%d", path, p.Version)
	}

	rsp, err := c.makeRequest(ctx, http.MethodGet, path, nil)
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
