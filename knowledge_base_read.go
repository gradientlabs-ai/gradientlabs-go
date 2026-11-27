package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ReadKnowledgeBase reads a knowledge base by the ID you gave it when you
// created it.
func (c *Client) ReadKnowledgeBase(ctx context.Context, knowledgeBaseID string) (*KnowledgeBase, error) {
	rsp, err := c.makeRequest(ctx, http.MethodGet, fmt.Sprintf("knowledge-bases/%s", url.PathEscape(knowledgeBaseID)), nil)
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
