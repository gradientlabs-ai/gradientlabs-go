package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestBulkUploadConversationMemories(t *testing.T) {
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
				Body:       io.NopCloser(strings.NewReader(`{"upload_id":"upl_123","memories_inserted":2}`)),
				Header:     make(http.Header),
			}, nil
		})),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.BulkUploadConversationMemories(context.Background(), "conv-1234", BulkUploadMemoriesParams{
		IdempotencyKey: "idem-1",
		Memories: []json.RawMessage{
			json.RawMessage(`{"order_id":"A1"}`),
			json.RawMessage(`{"order_id":"A2"}`),
		},
		CreatedAtKeys: []string{"created_at", "timestamp"},
	})
	if err != nil {
		t.Fatalf("BulkUploadConversationMemories: %v", err)
	}

	if gotReq.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", gotReq.Method)
	}
	if want := "/conversations/conv-1234/memories"; gotReq.URL.Path != want {
		t.Errorf("path = %q, want %q", gotReq.URL.Path, want)
	}

	var sent BulkUploadMemoriesParams
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if sent.IdempotencyKey != "idem-1" {
		t.Errorf("idempotency_key = %q, want idem-1", sent.IdempotencyKey)
	}
	if len(sent.Memories) != 2 {
		t.Errorf("memories length = %d, want 2", len(sent.Memories))
	}
	if len(sent.CreatedAtKeys) != 2 {
		t.Errorf("created_at_keys length = %d, want 2", len(sent.CreatedAtKeys))
	}

	if result.UploadID != "upl_123" {
		t.Errorf("upload_id = %q, want upl_123", result.UploadID)
	}
	if result.MemoriesInserted != 2 {
		t.Errorf("memories_inserted = %d, want 2", result.MemoriesInserted)
	}
}

func TestBulkUploadConversationMemoriesOmitsEmptyCreatedAtKeys(t *testing.T) {
	var gotBody []byte

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotBody, _ = io.ReadAll(r.Body)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"upload_id":"upl_1","memories_inserted":1}`)),
				Header:     make(http.Header),
			}, nil
		})),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.BulkUploadConversationMemories(context.Background(), "conv-1", BulkUploadMemoriesParams{
		IdempotencyKey: "idem-1",
		Memories:       []json.RawMessage{json.RawMessage(`{}`)},
	}); err != nil {
		t.Fatalf("BulkUploadConversationMemories: %v", err)
	}

	if strings.Contains(string(gotBody), "created_at_keys") {
		t.Errorf("expected created_at_keys to be omitted, got body: %s", gotBody)
	}
}
