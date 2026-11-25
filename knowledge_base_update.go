package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// UpdateKnowledgeBaseParams are the parameters to Client.UpdateKnowledgeBase.
type UpdateKnowledgeBaseParams struct {
	// Name is the new name of the knowledge base.
	Name string `json:"name"`
}

// UpdateKnowledgeBase renames a knowledge base, given the ID you gave it when
// you created it. The reserved knowledge bases "public-api" and "csv" cannot be
// renamed.
func (c *Client) UpdateKnowledgeBase(ctx context.Context, knowledgeBaseID string, p *UpdateKnowledgeBaseParams) (*KnowledgeBase, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, fmt.Sprintf("knowledge-bases/%s", url.PathEscape(knowledgeBaseID)), p)
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
