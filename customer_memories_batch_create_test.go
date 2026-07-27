package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestBatchCreateCustomerMemories(t *testing.T) {
	var gotReq *http.Request
	var gotBody []byte

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithURL("https://example.test"),
		WithTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotReq = r
			gotBody, _ = io.ReadAll(r.Body)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		})),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.BatchCreateCustomerMemories(context.Background(), "cust-1234", BatchCreateMemoriesParams{
		Memories: []Memory{
			{
				ExternalID: "order_123",
				CustomType: "order",
				CreatedAt:  time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
				Data:       json.RawMessage(`{"status":"shipped"}`),
			},
			{
				ExternalID: "order_456",
				CreatedAt:  time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
				Data:       json.RawMessage(`{"status":"delivered"}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("BatchCreateCustomerMemories: %v", err)
	}

	if gotReq.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", gotReq.Method)
	}
	if want := "/customers/cust-1234/memories"; gotReq.URL.Path != want {
		t.Errorf("path = %q, want %q", gotReq.URL.Path, want)
	}

	var sent BatchCreateMemoriesParams
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if len(sent.Memories) != 2 {
		t.Fatalf("memories length = %d, want 2", len(sent.Memories))
	}
	if sent.Memories[0].ExternalID != "order_123" {
		t.Errorf("memories[0].external_id = %q, want order_123", sent.Memories[0].ExternalID)
	}
	if sent.Memories[0].CustomType != "order" {
		t.Errorf("memories[0].custom_type = %q, want order", sent.Memories[0].CustomType)
	}
	if !sent.Memories[0].CreatedAt.Equal(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("memories[0].created_at = %v, want 2026-07-01T10:00:00Z", sent.Memories[0].CreatedAt)
	}
	if string(sent.Memories[0].Data) != `{"status":"shipped"}` {
		t.Errorf("memories[0].data = %s, want {\"status\":\"shipped\"}", sent.Memories[0].Data)
	}
}

func TestBatchCreateCustomerMemoriesOmitsEmptyCustomType(t *testing.T) {
	var gotBody []byte

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotBody, _ = io.ReadAll(r.Body)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		})),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.BatchCreateCustomerMemories(context.Background(), "cust-1", BatchCreateMemoriesParams{
		Memories: []Memory{
			{
				ExternalID: "order_123",
				CreatedAt:  time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
				Data:       json.RawMessage(`{}`),
			},
		},
	}); err != nil {
		t.Fatalf("BatchCreateCustomerMemories: %v", err)
	}

	if strings.Contains(string(gotBody), "custom_type") {
		t.Errorf("expected custom_type to be omitted, got body: %s", gotBody)
	}
}
