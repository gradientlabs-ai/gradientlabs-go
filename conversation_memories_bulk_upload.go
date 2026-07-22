package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// BulkUploadMemoriesParams are the parameters to Client.BulkUploadConversationMemories.
type BulkUploadMemoriesParams struct {
	// IdempotencyKey de-duplicates retries of the same upload. Re-uploading
	// with the same key returns the original upload instead of inserting again.
	IdempotencyKey string `json:"idempotency_key"`

	// Memories is the list of individual memories to store. Each element is
	// stored verbatim as the memory's raw payload and may be any JSON object.
	Memories []json.RawMessage `json:"memories"`

	// CreatedAtKeys optionally lists JSON keys tried in order to read each
	// memory's timestamp from its payload. When none match, the upload time
	// is used.
	CreatedAtKeys []string `json:"created_at_keys,omitempty"`
}

// MemoriesUploadResult is the result of Client.BulkUploadConversationMemories.
type MemoriesUploadResult struct {
	// UploadID uniquely identifies the upload.
	UploadID string `json:"upload_id"`

	// MemoriesInserted is the number of memories inserted by the upload.
	MemoriesInserted int `json:"memories_inserted"`
}

// BulkUploadConversationMemories uploads a batch of memories scoped to a
// conversation, for the AI agent to search over on demand.
func (c *Client) BulkUploadConversationMemories(ctx context.Context, conversationID string, p BulkUploadMemoriesParams) (*MemoriesUploadResult, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, fmt.Sprintf("conversations/%s/memories", conversationID), p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var result MemoriesUploadResult
	if err := json.NewDecoder(rsp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
