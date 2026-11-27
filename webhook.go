package client

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLeeway   = 5 * time.Minute
	signatureHeader = "X-GradientLabs-Signature"
	tokenHeader     = "X-GradientLabs-Token"
)

var (
	// ErrInvalidWebhookSignature is returned when the authenticity of the webhook
	// could not be verified using its signature. You should respond with an HTTP
	// 401 status code.
	ErrInvalidWebhookSignature = fmt.Errorf("%s header is invalid", signatureHeader)

	// ErrUnknownWebhookType is returned when the webhook contained an event of an
	// unknwon type. You should generally log these and return an HTTP 200 status
	// code.
	ErrUnknownWebhookType = errors.New("unknown webhook type")
)

// WebhookType indicates the type of webhook event.
type WebhookType string

const (
	// WebhookTypeAgentMessage indicates the agent wants to send the customer a
	// message.
	WebhookTypeAgentMessage WebhookType = "agent.message"

	// WebhookTypeConversationHandOff indicates the agent is escalating or handing
	// the conversation off to a human agent.
	WebhookTypeConversationHandOff WebhookType = "conversation.hand_off"

	// WebhookTypeConversationFinished indicates the agent has concluded the
	// conversation with the customer and you should close any corresponding
	// ticket, etc.
	WebhookTypeConversationFinished WebhookType = "conversation.finished"

	// WebhookTypeAction indicates that the agent needs an action to
	// be executed (e.g., while following a procedure)
	WebhookTypeActionExecute WebhookType = "action.execute"

	// Deprecated: resource requests now arrive as action.execute events.
	WebhookTypeResourcePull WebhookType = "resource.pull"

	// WebhookTypeConversationStart indicates the agent wants you to open an
	// outbound conversation. Reply with a ConversationStartResponse.
	WebhookTypeConversationStart WebhookType = "conversation.start"

	// WebhookTypeBackOfficeTaskComplete indicates the agent completed a
	// back-office task.
	WebhookTypeBackOfficeTaskComplete WebhookType = "back-office-task.complete"

	// WebhookTypeBackOfficeTaskHandOff indicates the agent handed a back-office
	// task off to a person.
	WebhookTypeBackOfficeTaskHandOff WebhookType = "back-office-task.hand-off"

	// WebhookTypeBackOfficeTaskFail indicates a back-office task failed.
	WebhookTypeBackOfficeTaskFail WebhookType = "back-office-task.fail"
)

// Webhook is an event delivered to your webhook endpoint.
type Webhook struct {
	// ID uniquely identifies this event.
	ID string `json:"id"`

	// Type indicates the type of event.
	Type WebhookType `json:"type"`

	// SequenceNumber can be used to establish an order of webhook events.
	// For more information, see: https://api-docs.gradient-labs.ai/#sequence-numbers
	SequenceNumber int `json:"sequence_number"`

	// Timestamp is the time at which this event was generated.
	Timestamp time.Time `json:"timestamp"`

	// Data contains the event data. Use the helper methods (e.g.
	// Webhook.AgentMessage) to access it.
	Data any `json:"-"`
}

// AgentMessage returns the data for an `agent.message` event.
func (w Webhook) AgentMessage() (*AgentMessageEvent, bool) {
	e, ok := w.Data.(*AgentMessageEvent)
	return e, ok
}

// ConversationHandOff returns the data for an `conversation.hand_off` event.
func (w Webhook) ConversationHandOff() (*ConversationHandOffEvent, bool) {
	e, ok := w.Data.(*ConversationHandOffEvent)
	return e, ok
}

// ConversationFinished returns the data for an `conversation.finished` event.
func (w Webhook) ConversationFinished() (*ConversationFinishedEvent, bool) {
	e, ok := w.Data.(*ConversationFinishedEvent)
	return e, ok
}

// ActionExecute returns the data for an `action.execute` event.
func (w Webhook) ActionExecute() (*ActionExecuteEvent, bool) {
	e, ok := w.Data.(*ActionExecuteEvent)
	return e, ok
}

// ResourcePull reads a legacy resource request.
// Deprecated: resource requests now arrive as action.execute events.
func (w Webhook) ResourcePull() (*ResourcePullEvent, bool) {
	e, ok := w.Data.(*ResourcePullEvent)
	return e, ok
}

// ConversationStart returns the data for a `conversation.start` event.
func (w Webhook) ConversationStart() (*ConversationStartEvent, bool) {
	e, ok := w.Data.(*ConversationStartEvent)
	return e, ok
}

// BackOfficeTaskComplete returns the data for a `back-office-task.complete`
// event.
func (w Webhook) BackOfficeTaskComplete() (*BackOfficeTaskCompleteEvent, bool) {
	e, ok := w.Data.(*BackOfficeTaskCompleteEvent)
	return e, ok
}

// BackOfficeTaskHandOff returns the data for a `back-office-task.hand-off`
// event.
func (w Webhook) BackOfficeTaskHandOff() (*BackOfficeTaskHandOffEvent, bool) {
	e, ok := w.Data.(*BackOfficeTaskHandOffEvent)
	return e, ok
}

// BackOfficeTaskFail returns the data for a `back-office-task.fail` event.
func (w Webhook) BackOfficeTaskFail() (*BackOfficeTaskFailEvent, bool) {
	e, ok := w.Data.(*BackOfficeTaskFailEvent)
	return e, ok
}

// AgentMessageEvent contains the data for an `agent.message` webhook event.
type AgentMessageEvent struct {
	// Conversation contains the details of the conversation the event relates to.
	Conversation WebhookConversation `json:"conversation"`

	// Body contains the text of the message the agent wants to send.
	Body string `json:"body"`

	// Total identifies the number of agent messages that have been
	// produced in the current turn.
	Total int `json:"total,omitempty"`

	// Sequence identifies which agent message this is in the current turn.
	Sequence int `json:"sequence,omitempty"`

	// Intent is the most recent intent that was classified from the customer's
	// conversation, if any.
	Intent string `json:"intent,omitempty"`

	// IsHolding identifies whether the agent message is a holding-response or not.
	// Holding-responses are sent to let the customer know that the agent is working on a full response.
	// For example: "I'm sorry to hear you want to close your account. Let me take a look at the next steps for you..."
	IsHolding bool `json:"is_holding,omitempty"`

	// LastCustomerMessageID is the ID of the customer's latest message that
	// this message answers, if any.
	LastCustomerMessageID string `json:"last_customer_message_id,omitempty"`
}

// ConversationHandOffEvent contains the data for a `conversation.hand_off` event.
type ConversationHandOffEvent struct {
	// Conversation contains the details of the conversation the event relates to.
	Conversation WebhookConversation `json:"conversation"`

	// Target defines where the agent wants to hand this conversation to.
	Target string `json:"target,omitempty"`

	// Reason is the code that describes why the agent wants to hand off this
	// conversation.
	Reason string `json:"reason_code"`

	// Description is a human-legible description of the Reason code.
	Description string `json:"reason"`

	// Note is a human-legible summary of the conversation so far, for a smooth hand-off.
	Note string `json:"note,omitempty"`

	// Intent is the most recent intent that was classified from the customer's
	// conversation, if any.
	Intent string `json:"intent,omitempty"`
}

// ConversationFinishedEvent contains the data for a `conversation.finished` event.
type ConversationFinishedEvent struct {
	// Conversation contains the details of the conversation the event relates to.
	Conversation WebhookConversation `json:"conversation"`

	// Reason is the code that describes why the agent wants to finish this
	// conversation.
	Reason string `json:"reason_code,omitempty"`

	// Intent is the most recent intent that was classified from the customer's
	// conversation, if any.
	Intent string `json:"intent,omitempty"`
}

// ActionExecuteEvent contains the data for a `action.execute` event.
type ActionExecuteEvent struct {
	// Action is the name of the action to execute
	Action string `json:"action"`

	// Params are the arguments to execute the action with.
	Params json.RawMessage `json:"params"`

	// Conversation contains the details of the conversation the event relates to.
	// It is empty when the action runs for a back-office task.
	Conversation WebhookConversation `json:"conversation"`

	// BackOfficeTask is the back-office task the action runs for. It is nil when
	// the action runs in a conversation.
	BackOfficeTask *WebhookBackOfficeTask `json:"back_office_task,omitempty"`
}

// ResourcePullEvent contains the data for a `resource.pull` event.
type ResourcePullEvent struct {
	// ResourceType is the name of the resource type the agent wants to pull.
	ResourceType string `json:"resource_type"`

	// Conversation contains the details of the conversation the event relates to.
	// It is empty when the resource is read for a back-office task.
	Conversation WebhookConversation `json:"conversation"`

	// BackOfficeTask is the back-office task the resource is read for. It is nil
	// when the resource is read for a conversation.
	BackOfficeTask *WebhookBackOfficeTask `json:"back_office_task,omitempty"`
}

// ConversationStartEvent contains the data for a `conversation.start` event,
// which asks you to open an outbound conversation. Reply with a
// ConversationStartResponse.
type ConversationStartEvent struct {
	// CustomerID is your identifier for the customer to contact.
	CustomerID string `json:"customer_id"`

	// Channel is how to contact the customer.
	Channel Channel `json:"channel"`

	// Body is the first message to send to the customer.
	Body string `json:"body"`

	// Subject is the subject line, for email conversations. It is empty on other
	// channels.
	Subject string `json:"subject"`

	// InitiatorID says who started the outbound conversation: "public-api" when
	// you started it through the API. When the AI agent started it from another
	// conversation or a back-office task, this is the Gradient Labs ID of that
	// conversation or task, not your ID for it.
	InitiatorID string `json:"initiator_id"`
}

// ConversationStartResponse is the body your endpoint returns to a
// `conversation.start` event.
type ConversationStartResponse struct {
	// ConversationID is your identifier for the conversation you opened. Use it
	// in later API calls about this conversation.
	ConversationID string `json:"conversation_id"`
}

// BackOfficeTaskCompleteEvent contains the data for a
// `back-office-task.complete` event.
type BackOfficeTaskCompleteEvent struct {
	// BackOfficeTask is the task that was completed.
	BackOfficeTask BackOfficeTask `json:"back_office_task"`
}

// BackOfficeTaskHandOffEvent contains the data for a
// `back-office-task.hand-off` event.
type BackOfficeTaskHandOffEvent struct {
	// BackOfficeTask is the task that was handed off.
	BackOfficeTask BackOfficeTask `json:"back_office_task"`
}

// BackOfficeTaskFailEvent contains the data for a `back-office-task.fail`
// event.
type BackOfficeTaskFailEvent struct {
	// BackOfficeTask is the task that failed.
	BackOfficeTask BackOfficeTask `json:"back_office_task"`
}

// WebhookBackOfficeTask names the back-office task a tool or resource request
// comes from.
type WebhookBackOfficeTask struct {
	// ID is your identifier for the back-office task.
	ID string `json:"id"`
}

// WebhookConversation contains the details of the conversation the webhook
// relates to.
type WebhookConversation struct {
	// ID is chosen unique identifier for this conversation.
	ID string `json:"id"`

	// CustomerID is your chosen identifier for the customer.
	CustomerID string `json:"customer_id"`

	// CustomerSource says where the customer's identifier came from. Only set on
	// `action.execute` and `resource.pull` events.
	CustomerSource string `json:"customer_source,omitempty"`

	// Metadata is the metadata you attached to the conversation. Only set on
	// `action.execute` and `resource.pull` events.
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// ParseWebhook parses the request, verifies its signature, and returns the
// webhook data.
func (c *Client) ParseWebhook(req *http.Request) (webhook *Webhook, token string, err error) {
	if err := c.VerifyWebhookRequest(req); err != nil {
		return nil, "", err
	}

	var payload struct {
		Webhook

		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		return nil, "", err
	}

	switch payload.Type {
	case WebhookTypeAgentMessage:
		var am AgentMessageEvent
		if err := json.Unmarshal(payload.Data, &am); err != nil {
			return nil, "", err
		}
		payload.Webhook.Data = &am
	case WebhookTypeConversationHandOff:
		var ho ConversationHandOffEvent
		if err := json.Unmarshal(payload.Data, &ho); err != nil {
			return nil, "", err
		}
		payload.Webhook.Data = &ho
	case WebhookTypeConversationFinished:
		var fin ConversationFinishedEvent
		if err := json.Unmarshal(payload.Data, &fin); err != nil {
			return nil, "", err
		}
		payload.Webhook.Data = &fin
	case WebhookTypeActionExecute:
		var act ActionExecuteEvent
		if err := json.Unmarshal(payload.Data, &act); err != nil {
			return nil, "", err
		}
		payload.Webhook.Data = &act
	case WebhookTypeResourcePull:
		var pull ResourcePullEvent
		if err := json.Unmarshal(payload.Data, &pull); err != nil {
			return nil, "", err
		}
		payload.Webhook.Data = &pull
	case WebhookTypeConversationStart:
		var start ConversationStartEvent
		if err := json.Unmarshal(payload.Data, &start); err != nil {
			return nil, "", err
		}
		payload.Webhook.Data = &start
	case WebhookTypeBackOfficeTaskComplete:
		var done BackOfficeTaskCompleteEvent
		if err := json.Unmarshal(payload.Data, &done); err != nil {
			return nil, "", err
		}
		payload.Webhook.Data = &done
	case WebhookTypeBackOfficeTaskHandOff:
		var ho BackOfficeTaskHandOffEvent
		if err := json.Unmarshal(payload.Data, &ho); err != nil {
			return nil, "", err
		}
		payload.Webhook.Data = &ho
	case WebhookTypeBackOfficeTaskFail:
		var fail BackOfficeTaskFailEvent
		if err := json.Unmarshal(payload.Data, &fail); err != nil {
			return nil, "", err
		}
		payload.Webhook.Data = &fail
	default:
		return nil, "", fmt.Errorf("%w: %q", ErrUnknownWebhookType, payload.Type)
	}

	// Extract the optional sensitive token from the request header.
	token = req.Header.Get(tokenHeader)

	return &payload.Webhook, token, nil
}

// VerifyWebhookRequest verifies the authenticity of the given request using
// its signature header. You do not need to call it if you're already using
// Client.ParseWebhook.
func (c *Client) VerifyWebhookRequest(req *http.Request) error {
	return c.webhookVerifier.VerifyRequest(req)
}

// WebhookVerifier verifies the authenticity of requests to your webhook
// endpoint using the X-GradientLabs-Signature header.
type WebhookVerifier struct {
	secret []byte
	leeway time.Duration
}

// VerifyRequest verifies the authenticity of the given request using its
// signature header.
func (v WebhookVerifier) VerifyRequest(req *http.Request) error {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if err := req.Body.Close(); err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return v.VerifySignature(body, req.Header.Get(signatureHeader))
}

// VerifySignature is a lower level variant of VerifyRequest
func (v WebhookVerifier) VerifySignature(body []byte, sig string) error {
	ts, sigs, err := v.parseHeader(sig)
	if err != nil {
		return ErrInvalidWebhookSignature
	}

	if time.Since(ts).Abs() > v.leeway {
		return ErrInvalidWebhookSignature
	}

	expected, err := v.computeSignature(ts, body)
	if err != nil {
		return err
	}

	for _, sig := range sigs {
		if hmac.Equal(expected, sig) {
			return nil
		}
	}

	return ErrInvalidWebhookSignature
}

func (v WebhookVerifier) computeSignature(ts time.Time, body []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, v.secret)

	if _, err := io.WriteString(mac, strconv.Itoa(int(ts.Unix()))); err != nil {
		return nil, err
	}

	if _, err := io.WriteString(mac, "."); err != nil {
		return nil, err
	}

	if _, err := mac.Write(body); err != nil {
		return nil, err
	}

	return mac.Sum(nil), nil
}

func (v WebhookVerifier) parseHeader(header string) (time.Time, [][]byte, error) {
	var (
		ts   time.Time
		sigs [][]byte
	)
	for _, pair := range strings.Split(header, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return ts, nil, fmt.Errorf("invalid %s header", signatureHeader)
		}

		switch parts[0] {
		case "t":
			unix, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return ts, nil, fmt.Errorf("invalid timestamp component %s", parts[1])
			}
			ts = time.Unix(unix, 0)
		case "v1":
			sig, err := hex.DecodeString(parts[1])
			if err != nil {
				return ts, nil, errors.New("invalid signature")
			}
			sigs = append(sigs, sig)
		}
	}

	if ts.IsZero() {
		return ts, nil, fmt.Errorf("%s header contains no timestamp component", signatureHeader)
	}

	return ts, sigs, nil
}
