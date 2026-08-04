package client

// StartOutboundConversationResponse is returned when an outbound conversation
// has been started.
type StartOutboundConversationResponse struct {
	// ConversationID is the internal identifier for the created conversation.
	// You can use this ID with other conversation APIs to check status, send
	// messages, etc.
	ConversationID string `json:"conversation_id"`
}
