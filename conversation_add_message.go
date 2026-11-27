package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AttachmentType identifies the type of file that has been attached to a
// message. Currently, our AI agent does not support processing attachments,
// and will hand the conversation off to a human agent if it encounters one.
type AttachmentType string

const (
	// AttachmentTypeImage indicates that the attachment is an image.
	AttachmentTypeImage AttachmentType = "image"

	// AttachmentTypeFile indicates that the attachment is a generic file such as
	// a document.
	AttachmentTypeFile AttachmentType = "file"
)

// Attachment represents a file that was uploaded with a message.
type Attachment struct {
	// Type of file that was uploaded.
	Type AttachmentType `json:"type"`

	// FileName is the name of the file that was uploaded, or an adequate
	// placeholder for that name, which can be shown to reviewers.
	//
	// If you cannot add the actual file name, a placeholder value is acceptable.
	FileName string `json:"file_name"`

	// URL is the publicly accessible URL where the attachment can be downloaded
	// from. This should be a fully qualified URL. If not given, the AI agent will
	// only know that an attachment exists, but will be unable to process it.
	URL string `json:"url,omitempty"`

	// Summary is an optional short summary of the attachment.  This is only intended
	// to be used if you cannot give us access to the raw attachment, but can
	// run your own LLM completions on the attachment and send us a description instead.
	//
	// If a summary or description is provided, then there must be no URL.
	Summary string `json:"summary,omitempty"`

	// Description is an optional description of the attachment. This is only intended
	// to be used if you cannot give us access to the raw attachment, but can
	// run your own LLM completions on the attachment and send us a description instead.
	//
	// Chat to us first before using it!
	Description string `json:"description,omitempty"`
}

// AddMessageParams are the parameters to Client.AddMessage.
type AddMessageParams struct {
	// ID uniquely identifies this message within the conversation.
	//
	// Can be anything consisting of letters, numbers, or any of the following
	// characters: _ - + =.
	//
	// Tip: use something meaningful to your business.
	ID string `json:"id"`

	// Body contains the message text.
	Body string `json:"body"`

	// Subject is the email subject line. Only applicable for email-channel conversations.
	Subject string `json:"subject,omitempty"`

	// ParticipantID identifies the message sender.
	ParticipantID string `json:"participant_id,omitempty"`

	// ParticipantType identifies the type of participant who sent this message.
	// AI agent messages can only be replayed.
	ParticipantType ParticipantType `json:"participant_type"`

	// Created is the time at which the message was sent.
	Created time.Time `json:"created"`

	// ConversationToken is sent with subsequent tool and webhook calls.
	ConversationToken string `json:"conversation_token,omitempty"`

	// Attachments contains any files that were uploaded with this message.
	Attachments []*Attachment `json:"attachments,omitempty"`
}

// Message represents a message sent by a customer, human support agent, or the
// API agent.
type Message struct {
	// ID uniquely identifies this message within the conversation.
	//
	// Can be anything consisting of letters, numbers, or any of the following
	// characters: _ - + =.
	//
	// Tip: use something meaningful to your business.
	ID string `json:"id"`

	// Body contains the message text.
	Body string `json:"body"`

	// ParticipantID identifies the message sender.
	ParticipantID string `json:"participant_id"`

	// ParticipantType identifies the type of participant who sent this message.
	ParticipantType ParticipantType `json:"participant_type"`

	// Created is the time at which the message was sent.
	Created *time.Time `json:"created"`

	// Attachments contains any files that were uploaded with this message.
	Attachment []*Attachment `json:"attachments,omitempty"`
}

// AddMessage records a message sent by the customer or a human agent.
func (c *Client) AddMessage(ctx context.Context, conversationID string, p AddMessageParams) (*Message, error) {
	// Preserve the timestamp type callers use while letting the API set an omitted time.
	body := struct {
		AddMessageParams
		Created *time.Time `json:"created,omitempty"`
	}{AddMessageParams: p}
	if !p.Created.IsZero() {
		body.Created = &p.Created
	}
	rsp, err := c.makeRequest(ctx, http.MethodPost, fmt.Sprintf("conversations/%s/messages", conversationID), body)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var msg Message
	if err := json.NewDecoder(rsp.Body).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
