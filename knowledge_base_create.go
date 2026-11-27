package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// CreateKnowledgeBaseParams are the parameters to Client.CreateKnowledgeBase.
type CreateKnowledgeBaseParams struct {
	// ID is your own identifier for this knowledge base. You name the knowledge
	// base by it in KnowledgeBaseIDs on articles and notes. It can't contain
	// `/`, and "public-api" and "csv" are reserved.
	ID string `json:"id"`

	// Name is the name of the knowledge base.
	Name string `json:"name"`
}

// CreateKnowledgeBase creates a knowledge base that articles and notes can be
// attached to. It returns a 409 error when a knowledge base with the ID already
// exists.
func (c *Client) CreateKnowledgeBase(ctx context.Context, p *CreateKnowledgeBaseParams) (*KnowledgeBase, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "knowledge-bases", p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var kb KnowledgeBase
	if err := json.NewDecoder(rsp.Body).Decode(&kb); err != nil {
		return nil, err
	}
	return &kb, nil
}
