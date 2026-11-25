package client

// SupportPlatform identifies the support platform a conversation lives on.
type SupportPlatform string

const (
	SupportPlatformChatSDK       SupportPlatform = "chat-sdk"
	SupportPlatformDixa          SupportPlatform = "dixa"
	SupportPlatformFreshchat     SupportPlatform = "freshchat"
	SupportPlatformFreshdesk     SupportPlatform = "freshdesk"
	SupportPlatformGmail         SupportPlatform = "gmail"
	SupportPlatformIntercom      SupportPlatform = "intercom"
	SupportPlatformIntercomVoice SupportPlatform = "intercom-voice"
	SupportPlatformKustomer      SupportPlatform = "kustomer"
	SupportPlatformLivechat      SupportPlatform = "livechat"
	SupportPlatformLiveKit       SupportPlatform = "livekit"
	SupportPlatformPublicAPI     SupportPlatform = "public-api"
	SupportPlatformSalesforce    SupportPlatform = "salesforce"
	SupportPlatformTalkdesk      SupportPlatform = "talkdesk"
	SupportPlatformTwilio        SupportPlatform = "twilio"
	SupportPlatformZendesk       SupportPlatform = "zendesk"
)
