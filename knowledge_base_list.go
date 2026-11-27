package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// ListKnowledgeBasesResponse contains the knowledge bases created through the
// API.
type ListKnowledgeBasesResponse struct {
	KnowledgeBases []*KnowledgeBase `json:"knowledge_bases"`
}

// ListKnowledgeBases lists the knowledge bases created through the API.
func (c *Client) ListKnowledgeBases(ctx context.Context) (*ListKnowledgeBasesResponse, error) {
	rsp, err := c.makeRequest(ctx, http.MethodGet, "knowledge-bases", nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var kbs ListKnowledgeBasesResponse
	if err := json.NewDecoder(rsp.Body).Decode(&kbs); err != nil {
		return nil, err
	}
	return &kbs, nil
}
