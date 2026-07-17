package client

import (
	"context"
	"fmt"
	"net/http"
)

// DeleteConversation deletes a conversation by its ID. The conversation must
// not be in an active state, and on-demand deletion must be enabled.
//
// Note: requires a `Management` API key.
func (c *Client) DeleteConversation(ctx context.Context, conversationID string) error {
	rsp, err := c.makeRequest(ctx, http.MethodDelete, fmt.Sprintf("conversations/%s", conversationID), nil)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return err
	}
	return nil
}
