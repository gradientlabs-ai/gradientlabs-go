package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// StartConversationParams are the parameters to Client.StartConversation.
type StartConversationParams struct {
	// ID uniquely identifies the conversation.
	//
	// Can be anything consisting of letters, numbers, or any of the following
	// characters: _ - + =.
	//
	// Tip: use something meaningful to your business (e.g. a ticket number).
	ID string `json:"id"`

	// CustomerID uniquely identifies the customer. Used to build historical
	// context of conversations the agent has had with this customer.
	CustomerID string `json:"customer_id"`

	// CustomerSupportPlatformIdentifiers optionally links the customer to
	// their record(s) in third-party support platforms (e.g. Intercom,
	// Zendesk), alongside CustomerID.
	CustomerSupportPlatformIdentifiers []*CustomerSupportPlatformIdentifier `json:"customer_support_platform_identifiers,omitempty"`

	// AssigneeID optionally identifies who the conversation is assigned to.
	AssigneeID string `json:"assignee_id,omitempty"`

	// AssigneeType optionally identifies which type of participant is currently
	// assigned to respond. Set this to ParticipantTypeAIAgent to assign the conversation
	// to the Gradient Labs AI when starting it.
	AssigneeType ParticipantType `json:"assignee_type,omitempty"`

	// Channel represents the way a customer is getting in touch. It will be used
	// to determine how the agent formats responses, etc.
	Channel Channel `json:"channel"`

	// Created optionally defines the time when the conversation started.
	// If not given, this will default to the current time.
	Created *time.Time `json:"created,omitempty"`

	// Resources is an arbitrary object attached to the conversation and available to the AI agent
	// during the conversation. You can also use resources as parameters for your tools.
	Resources map[string]any `json:"resources,omitempty"`

	// ConversationToken is the raw sensitive token that can be optionally provided when starting a conversation.
	// The latest token of the conversation will be echoed back in future Webhooks, under the header `X-GradientLabs-Token`,
	// as well as in HTTP Tools using templates.
	ConversationToken string `json:"conversation_token,omitempty"`

	// TrafficGroupID optionally restricts the conversation to only access procedures
	// assigned to the specified traffic group, plus any procedures not assigned to any group.
	TrafficGroupID string `json:"traffic_group_id,omitempty"`
}

// StartConversation begins a conversation.
func (c *Client) StartConversation(ctx context.Context, p StartConversationParams) (*Conversation, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "conversations", p)
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
