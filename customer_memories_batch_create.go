package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Memory is a single memory to store for a customer, which the AI agent can
// then search over on demand.
type Memory struct {
	// ExternalID is the caller's own identifier for the memory.
	ExternalID string `json:"external_id"`

	// CustomType is an optional free-form label categorising the memory.
	CustomType string `json:"custom_type,omitempty"`

	// CreatedAt is the time at which the memory was created.
	CreatedAt time.Time `json:"created_at"`

	// Data is the memory's payload. It is stored verbatim and may be any JSON
	// object.
	Data json.RawMessage `json:"data"`
}

// BatchCreateMemoriesParams are the parameters to Client.BatchCreateCustomerMemories.
type BatchCreateMemoriesParams struct {
	// Memories is the non-empty list of memories to store for the customer.
	Memories []Memory `json:"memories"`
}

// BatchCreateCustomerMemories submits a batch of memories scoped to a customer,
// for the AI agent to search over on demand.
//
// The batch is processed asynchronously: the call returns as soon as the job is
// accepted and does not wait for the memories to be stored. It returns a 409
// error when a batch is already being created for the same customer.
func (c *Client) BatchCreateCustomerMemories(ctx context.Context, customerID string, p BatchCreateMemoriesParams) error {
	rsp, err := c.makeRequest(ctx, http.MethodPost, fmt.Sprintf("customers/%s/memories", customerID), p)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return err
	}
	return nil
}
