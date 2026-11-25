package client

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testSigningKey = "whsec_test"

func signedWebhookRequest(t *testing.T, body string) *http.Request {
	t.Helper()

	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(testSigningKey))
	fmt.Fprintf(mac, "%d.%s", ts, body)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set(signatureHeader, fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil))))
	req.Header.Set(tokenHeader, "conversation-token")
	return req
}

func parseTestWebhook(t *testing.T, eventType, data string) *Webhook {
	t.Helper()

	c, err := NewClient(WithWebhookSigningKey(testSigningKey))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	body := fmt.Sprintf(`{"id":"evt_1","type":%q,"sequence_number":4,"timestamp":"2024-01-15T10:30:00Z","data":%s}`, eventType, data)
	wh, token, err := c.ParseWebhook(signedWebhookRequest(t, body))
	if err != nil {
		t.Fatalf("ParseWebhook(%s): %v", eventType, err)
	}
	if token != "conversation-token" {
		t.Errorf("token = %q", token)
	}
	if wh.Type != WebhookType(eventType) || wh.SequenceNumber != 4 {
		t.Errorf("envelope = %+v", wh)
	}
	return wh
}

const backOfficeTaskJSON = `{"back_office_task":{"id":"task-1","agent_id":"agent_1","status":"handed-off","input":{"a":1},"created":"2024-01-15T10:30:00Z","handed_off":"2024-01-15T10:40:00Z","hand_off_reason":"operator-handoff","failure_reasons":["why"],"result":{"result_type":"refund"}}}`

func TestParseWebhookBackOfficeTaskEvents(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		wh := parseTestWebhook(t, "back-office-task.complete", backOfficeTaskJSON)
		e, ok := wh.BackOfficeTaskComplete()
		if !ok || e.BackOfficeTask.ID != "task-1" || e.BackOfficeTask.Result == nil || e.BackOfficeTask.Result.ResultType != "refund" {
			t.Errorf("event = %+v, %v", e, ok)
		}
	})

	t.Run("hand-off", func(t *testing.T) {
		wh := parseTestWebhook(t, "back-office-task.hand-off", backOfficeTaskJSON)
		e, ok := wh.BackOfficeTaskHandOff()
		if !ok || e.BackOfficeTask.HandOffReason != BackOfficeTaskHandOffReasonOperatorHandoff || e.BackOfficeTask.Status != BackOfficeTaskStatusHandedOff {
			t.Errorf("event = %+v, %v", e, ok)
		}
	})

	t.Run("fail", func(t *testing.T) {
		wh := parseTestWebhook(t, "back-office-task.fail", backOfficeTaskJSON)
		e, ok := wh.BackOfficeTaskFail()
		if !ok || len(e.BackOfficeTask.FailureReasons) != 1 {
			t.Errorf("event = %+v, %v", e, ok)
		}
	})
}

func TestParseWebhookConversationStart(t *testing.T) {
	wh := parseTestWebhook(t, "conversation.start", `{"customer_id":"user-1","channel":"email","body":"Hello","subject":"Your payment","initiator_id":"public-api"}`)

	e, ok := wh.ConversationStart()
	if !ok {
		t.Fatalf("ConversationStart() = %v", ok)
	}
	if e.CustomerID != "user-1" || e.Channel != ChannelEmail || e.Body != "Hello" || e.Subject != "Your payment" || e.InitiatorID != "public-api" {
		t.Errorf("event = %+v", e)
	}
}

func TestParseWebhookAgentMessageLastCustomerMessageID(t *testing.T) {
	wh := parseTestWebhook(t, "agent.message", `{"conversation":{"id":"c1","customer_id":"u1"},"body":"Hi","last_customer_message_id":"m1","sequence":1,"total":2}`)

	e, ok := wh.AgentMessage()
	if !ok || e.LastCustomerMessageID != "m1" {
		t.Errorf("event = %+v, %v", e, ok)
	}
}

func TestParseWebhookActionExecuteForAConversationAndATask(t *testing.T) {
	wh := parseTestWebhook(t, "action.execute", `{"action":"lookup-order","params":{"order_id":"1"},"conversation":{"id":"c1","customer_id":"u1","customer_source":"web-app","metadata":{"plan":"pro"}}}`)
	e, ok := wh.ActionExecute()
	if !ok || e.Conversation.CustomerSource != "web-app" || string(e.Conversation.Metadata) != `{"plan":"pro"}` || e.BackOfficeTask != nil {
		t.Errorf("event = %+v, %v", e, ok)
	}

	wh = parseTestWebhook(t, "action.execute", `{"action":"lookup-order","params":{},"back_office_task":{"id":"task-1"}}`)
	e, ok = wh.ActionExecute()
	if !ok || e.BackOfficeTask == nil || e.BackOfficeTask.ID != "task-1" || e.Conversation.ID != "" {
		t.Errorf("event = %+v, %v", e, ok)
	}
}

func TestParseWebhookResourcePullForATask(t *testing.T) {
	wh := parseTestWebhook(t, "resource.pull", `{"resource_type":"order-history","back_office_task":{"id":"task-1"}}`)
	e, ok := wh.ResourcePull()
	if !ok || e.ResourceType != "order-history" || e.BackOfficeTask == nil || e.BackOfficeTask.ID != "task-1" {
		t.Errorf("event = %+v, %v", e, ok)
	}
}

func TestParseWebhookExistingEvents(t *testing.T) {
	wh := parseTestWebhook(t, "conversation.hand_off", `{"conversation":{"id":"c1","customer_id":"u1"},"reason":"The customer asked","reason_code":"customer-request","target":"billing","note":"n","intent":"refund"}`)
	if e, ok := wh.ConversationHandOff(); !ok || e.Reason != "customer-request" || e.Description != "The customer asked" || e.Target != "billing" {
		t.Errorf("event = %+v, %v", e, ok)
	}

	wh = parseTestWebhook(t, "conversation.finished", `{"conversation":{"id":"c1","customer_id":"u1"},"reason_code":"spam","intent":"refund"}`)
	if e, ok := wh.ConversationFinished(); !ok || e.Reason != "spam" {
		t.Errorf("event = %+v, %v", e, ok)
	}
}

func TestParseWebhookUnknownType(t *testing.T) {
	c, err := NewClient(WithWebhookSigningKey(testSigningKey))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	body := `{"id":"evt_1","type":"something.new","sequence_number":1,"timestamp":"2024-01-15T10:30:00Z","data":{}}`

	_, _, err = c.ParseWebhook(signedWebhookRequest(t, body))
	if !errors.Is(err, ErrUnknownWebhookType) {
		t.Errorf("err = %v, want ErrUnknownWebhookType", err)
	}
}

func TestParseWebhookRejectsABadSignature(t *testing.T) {
	c, err := NewClient(WithWebhookSigningKey("another-key"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	body := `{"id":"evt_1","type":"agent.message","data":{}}`

	_, _, err = c.ParseWebhook(signedWebhookRequest(t, body))
	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Errorf("err = %v, want ErrInvalidWebhookSignature", err)
	}
}
