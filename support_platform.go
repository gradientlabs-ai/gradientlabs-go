package client

// SupportPlatform identifies the support platform a conversation lives on.
type SupportPlatform string

const (
	SupportPlatformFreshchat  SupportPlatform = "freshchat"
	SupportPlatformFreshdesk  SupportPlatform = "freshdesk"
	SupportPlatformIntercom   SupportPlatform = "intercom"
	SupportPlatformPublicAPI  SupportPlatform = "public-api"
	SupportPlatformSalesforce SupportPlatform = "salesforce"
	SupportPlatformZendesk    SupportPlatform = "zendesk"
)
