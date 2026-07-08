package client

// CustomerSupportPlatformIdentifierType identifies the kind of record an
// identifier points to, for support platforms that have more than one kind
// of user identifier.
type CustomerSupportPlatformIdentifierType string

const (
	// CustomerSupportPlatformIdentifierTypeIntercomLead indicates the
	// identifier is for an Intercom lead.
	CustomerSupportPlatformIdentifierTypeIntercomLead CustomerSupportPlatformIdentifierType = "intercom_lead"

	// CustomerSupportPlatformIdentifierTypeIntercomUser indicates the
	// identifier is for an Intercom user.
	CustomerSupportPlatformIdentifierTypeIntercomUser CustomerSupportPlatformIdentifierType = "intercom_user"

	// CustomerSupportPlatformIdentifierTypeZendeskConversationUser indicates
	// the identifier is for a Zendesk conversation (end) user.
	CustomerSupportPlatformIdentifierTypeZendeskConversationUser CustomerSupportPlatformIdentifierType = "zendesk_conversation_user"

	// CustomerSupportPlatformIdentifierTypeZendeskSupportUser indicates the
	// identifier is for a Zendesk support user.
	CustomerSupportPlatformIdentifierTypeZendeskSupportUser CustomerSupportPlatformIdentifierType = "zendesk_support_user"

	// CustomerSupportPlatformIdentifierTypeSalesforceContactID indicates the
	// identifier is a Salesforce contact ID.
	CustomerSupportPlatformIdentifierTypeSalesforceContactID CustomerSupportPlatformIdentifierType = "salesforce_contact_id"

	// CustomerSupportPlatformIdentifierTypeSalesforceAccountID indicates the
	// identifier is a Salesforce account ID.
	CustomerSupportPlatformIdentifierTypeSalesforceAccountID CustomerSupportPlatformIdentifierType = "salesforce_account_id"
)

// CustomerSupportPlatformIdentifier links the customer being created to their
// record in a third-party support platform, alongside CustomerID.
type CustomerSupportPlatformIdentifier struct {
	// SupportPlatform is the support platform this identifier belongs to.
	SupportPlatform SupportPlatform `json:"support_platform"`

	// Type identifies the kind of identifier this is. Only required for
	// platforms that have more than one kind of identifier: intercom,
	// zendesk, and salesforce. Leave this empty for freshchat and freshdesk,
	// which don't have subtypes.
	Type CustomerSupportPlatformIdentifierType `json:"type,omitempty"`

	// Value is the external ID of the customer in the support platform.
	Value string `json:"value"`
}
