package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// StartOutboundEmailConversationParams are the parameters to
// Client.StartOutboundEmailConversation.
type StartOutboundEmailConversationParams struct {
	// CustomerID is your own identifier for the customer, as used in your
	// systems. It is stored as the customer's company customer ID, and is the
	// identifier echoed back to you in tool and webhook payloads.
	CustomerID string `json:"customer_id"`

	// ProcedureID is the ID of the outbound procedure that defines what the AI
	// agent should accomplish in this conversation. The procedure must be of
	// type "outbound", must be live (deployed), and must be enabled for the
	// email channel.
	ProcedureID string `json:"procedure_id"`

	// SupportPlatform is the support platform the email is sent from.
	SupportPlatform SupportPlatform `json:"support_platform"`

	// CustomerSupportPlatformIdentifiers optionally links the customer to their
	// record(s) in third-party support platforms (e.g. Intercom, Zendesk,
	// Salesforce). These are added to the customer alongside CustomerID, and
	// can be used to match against customers created via those platforms'
	// native integrations.
	//
	// The platform named in SupportPlatform needs an identifier here, unless
	// the customer already carries one from an earlier conversation. Zendesk
	// requires type CustomerSupportPlatformIdentifierTypeZendeskSupportUser;
	// Salesforce requires
	// CustomerSupportPlatformIdentifierTypeSalesforceContactID.
	CustomerSupportPlatformIdentifiers []*CustomerSupportPlatformIdentifier `json:"customer_support_platform_identifiers,omitempty"`

	// Subject is the subject line for the initial email. Required if Body is
	// provided, and forbidden otherwise.
	Subject string `json:"subject,omitempty"`

	// Body is the content of the initial email to send to the customer.
	// Required if Subject is provided, and forbidden otherwise. If both are
	// omitted, the AI agent will write the opening email based on the
	// procedure.
	Body string `json:"body,omitempty"`

	// Resources is an arbitrary object attached to the conversation and
	// available to the AI agent during the conversation. You can also use
	// resources as parameters for your tools.
	Resources map[string]any `json:"resources,omitempty"`
}

// StartOutboundEmailConversation creates and starts a new outbound email
// conversation, in which the AI agent proactively initiates contact with a
// customer, following the instructions defined in the specified outbound
// procedure.
//
// If Subject and Body are provided, that email will be sent as the opening
// message. Otherwise, the AI agent will write one based on the procedure.
//
// The customer is created, or matched to an existing record, from CustomerID
// and any CustomerSupportPlatformIdentifiers you supply. The platform the email
// is sent from needs an identifier for that customer, so sending from Zendesk
// needs a Zendesk identifier, and so on.
func (c *Client) StartOutboundEmailConversation(ctx context.Context, p StartOutboundEmailConversationParams) (*StartOutboundConversationResponse, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "outbound/conversations/email", p)
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
