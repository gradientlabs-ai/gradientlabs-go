package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ListCustomerMemoriesResponse contains the memories stored for a customer.
type ListCustomerMemoriesResponse struct {
	// Memories is every memory currently retained for the customer.
	Memories []CustomerMemory `json:"memories"`
}

// ListCustomerMemories lists all the memories stored for a customer.
// This operation is experimental.
func (c *Client) ListCustomerMemories(ctx context.Context, customerID string) (*ListCustomerMemoriesResponse, error) {
	rsp, err := c.makeRequest(ctx, http.MethodGet, fmt.Sprintf("customers/%s/memories", url.PathEscape(customerID)), nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var memories ListCustomerMemoriesResponse
	if err := json.NewDecoder(rsp.Body).Decode(&memories); err != nil {
		return nil, err
	}
	return &memories, nil
}
