package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ReadConversationDeprecated reads a conversation with the older
// GET /conversations/{id} endpoint, which cannot read a conversation from
// another support platform.
//
// Deprecated: use ReadConversation.
func (c *Client) ReadConversationDeprecated(ctx context.Context, conversationID string) (*Conversation, error) {
	rsp, err := c.makeRequest(ctx, http.MethodGet, fmt.Sprintf("conversations/%s", url.PathEscape(conversationID)), nil)
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
