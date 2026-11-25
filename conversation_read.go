package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type ReadParams struct {
	// SupportPlatform is the name of the support platform where the
	// conversation was started (e.g. Intercom).
	//
	// Leave empty if the conversation was started via the Gradient
	// Labs API.
	SupportPlatform string `json:"support_platform,omitempty"`
}

func (c *Client) ReadConversation(ctx context.Context, conversationID string, p *ReadParams) (*Conversation, error) {
	path := fmt.Sprintf("conversations/%s/read", url.PathEscape(conversationID))
	if p != nil && p.SupportPlatform != "" {
		path = fmt.Sprintf("%s?support_platform=%s", path, url.QueryEscape(p.SupportPlatform))
	}

	rsp, err := c.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var conv Conversation
	if err := json.NewDecoder(rsp.Body).Decode(&conv); err != nil {
		return nil, err
	}
	return &conv, nil
}
