package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// MemoryV2 is a single memory to store for a customer, with a described
// category.
type MemoryV2 struct {
	// ExternalID is the caller's own identifier for the memory. It must be
	// unique within your company, and retrying the same memory does not create
	// a duplicate. IDs starting with "support_conversation/" are reserved.
	ExternalID string `json:"external_id"`

	// CustomType is the customer-defined category for the memory.
	CustomType MemoryCustomType `json:"custom_type"`

	// CreatedAt is the time the memory's underlying event occurred.
	CreatedAt time.Time `json:"created_at"`

	// Data is the memory's payload. It is stored verbatim and may be any JSON value.
	Data json.RawMessage `json:"data"`
}

// BatchCreateMemoriesV2Params are the parameters to Client.BatchCreateCustomerMemoriesV2.
type BatchCreateMemoriesV2Params struct {
	// Memories is the non-empty list of memories to store for the customer.
	Memories []MemoryV2 `json:"memories"`
}

// BatchCreateCustomerMemoriesV2 submits a batch of memories scoped to a
// customer, for the AI agent to search over on demand. Unlike
// BatchCreateCustomerMemories, each memory's category can carry a description.
//
// The batch is processed asynchronously: the call returns as soon as the job is
// accepted and does not wait for the memories to be stored. This operation is experimental. It returns a 409
// error when a batch is already being created for the same customer.
func (c *Client) BatchCreateCustomerMemoriesV2(ctx context.Context, customerID string, p BatchCreateMemoriesV2Params) error {
	rsp, err := c.makeRequest(ctx, http.MethodPost, fmt.Sprintf("customers/%s/memories/v2", url.PathEscape(customerID)), p)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return err
	}
	return nil
}
