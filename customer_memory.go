package client

import (
	"encoding/json"
	"time"
)

// MemoryCustomType is the customer-defined category of a memory.
type MemoryCustomType struct {
	// Name is the category's name. "support_conversation" is reserved for
	// Gradient Labs.
	Name string `json:"name"`

	// Description says what the category holds.
	Description string `json:"description,omitempty"`
}

// CustomerMemory is a memory stored for a customer, which the AI agent can
// search over on demand.
type CustomerMemory struct {
	// ExternalID is your own identifier for the memory.
	ExternalID string `json:"external_id"`

	// CustomType is the customer-defined category for the memory.
	CustomType MemoryCustomType `json:"custom_type"`

	// CreatedAt is the time the memory's underlying event occurred.
	CreatedAt time.Time `json:"created_at"`

	// Data is the memory's payload, which is stored verbatim.
	Data json.RawMessage `json:"data"`
}
