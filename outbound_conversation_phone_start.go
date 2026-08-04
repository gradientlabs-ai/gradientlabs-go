package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// StartOutboundPhoneConversationParams are the parameters to
// Client.StartOutboundPhoneConversation.
type StartOutboundPhoneConversationParams struct {
	// CustomerID is your own identifier for the customer, as used in your
	// systems. It is stored as the customer's company customer ID, and is the
	// identifier echoed back to you in tool and webhook payloads.
	CustomerID string `json:"customer_id"`

	// ProcedureID is the ID of the outbound procedure that defines what the AI
	// agent should accomplish on the call. The procedure must be of type
	// "outbound", must be live (deployed), and must be enabled for the "voice"
	// channel.
	ProcedureID string `json:"procedure_id"`

	// ToPhoneNumber is the customer's phone number to dial (E.164 format, e.g.
	// "+14155551234").
	ToPhoneNumber string `json:"to_phone_number"`

	// FromPhoneNumber is the caller ID to place the call from (E.164 format).
	// It must be a phone number already provisioned for your company.
	FromPhoneNumber string `json:"from_phone_number"`

	// CustomerSupportPlatformIdentifiers optionally links the customer to their
	// record(s) in third-party support platforms (e.g. Intercom, Zendesk,
	// Salesforce). These are added to the customer alongside CustomerID, and
	// can be used to match against customers created via those platforms'
	// native integrations, and to pull that platform's customer data into the
	// call as context.
	CustomerSupportPlatformIdentifiers []*CustomerSupportPlatformIdentifier `json:"customer_support_platform_identifiers,omitempty"`

	// Resources is an arbitrary object attached to the conversation and
	// available to the AI agent during the conversation. You can also use
	// resources as parameters for your tools.
	Resources map[string]any `json:"resources,omitempty"`
}

// StartOutboundPhoneConversation places an outbound phone call, in which the AI
// agent proactively contacts a customer, following the instructions defined in
// the specified outbound procedure.
//
// FromPhoneNumber must be a phone number already provisioned for your company.
//
// The customer is created, or matched to an existing record, from CustomerID
// and any CustomerSupportPlatformIdentifiers you supply. The dialled number is
// recorded against that same customer.
func (c *Client) StartOutboundPhoneConversation(ctx context.Context, p StartOutboundPhoneConversationParams) (*StartOutboundConversationResponse, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "outbound/conversations/phone", p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var result StartOutboundConversationResponse
	if err := json.NewDecoder(rsp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
