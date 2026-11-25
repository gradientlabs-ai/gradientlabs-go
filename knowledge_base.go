package client

import "time"

// KnowledgeBase groups articles and notes, and controls which AI agents can use
// them.
type KnowledgeBase struct {
	// ID is the Gradient Labs identifier for this knowledge base.
	ID string `json:"gradient_labs_id"`

	// ExternalID is your own identifier for this knowledge base. You name the
	// knowledge base by it in KnowledgeBaseIDs on articles and notes.
	ExternalID string `json:"id"`

	// Name is the name of the knowledge base.
	Name string `json:"name"`

	// Created is when the knowledge base was created.
	Created time.Time `json:"created"`

	// Updated is when the knowledge base was last updated.
	Updated time.Time `json:"updated"`
}
