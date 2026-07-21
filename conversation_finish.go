package client

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type FinishParams struct {
	// Timestamp optionally defines the time when the conversation ended.
	// If not given, this will default to the current time.
	Timestamp *time.Time `json:"timestamp,omitempty"`

	// Reason optionally allows you to describe why this conversation is finishing.
	Reason string `json:"reason,omitempty"`

	// ReasonCode optionally categorises why this conversation is finishing.
	// Valid values are "customer-ended-chat" and "customer-unresponsive".
	ReasonCode string `json:"reason_code,omitempty"`
}

// FinishConversation finishes a conversation.
//
// A conversation finishes when it has come to its natural conclusion. This could be because
// the customer's query has been resolved, a human agent or other automation has closed the chat,
// or because the chat is being closed due to inactivity.
func (c *Client) FinishConversation(ctx context.Context, conversationID string, p FinishParams) error {
	rsp, err := c.makeRequest(ctx, http.MethodPut, fmt.Sprintf("conversations/%s/finish", conversationID), p)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return err
	}
	return nil
}
