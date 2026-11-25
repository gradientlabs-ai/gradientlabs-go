package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ListTopicsParams struct {
	// SupportPlatform optionally filters topics by support platform.
	// Valid values include "public-api" and "intercom". If not provided,
	// topics from every platform are listed.
	SupportPlatform string `json:"support_platform,omitempty"`
}

type Topic struct {
	// Source identifies the CRM or support platform that the topic comes from
	Source string `json:"Source"`

	// ExternalID identifies this topic in the Source
	ExternalID string `json:"ExternalID"`

	// Name is the human-readable name for this topic
	Name string `json:"Name"`

	// Description is the optional subtext for the topic
	Description string `json:"Description,omitempty"`

	// Visibility describes who can see the topic
	Visibility Visibility `json:"Visibility"`

	// ParentExternalID identifies the topic that this topic is nested under
	ParentExternalID string `json:"ParentExternalID,omitempty"`

	// Created is when the topic was created in the source
	Created time.Time `json:"Created"`

	// LastEdited is when the topic was last changed in the source
	LastEdited time.Time `json:"LastEdited"`

	// LastSeen is the last time we saw this topic when crawling
	LastSeen time.Time `json:"LastSeen"`

	// Data is a raw representation of the topic from the support platform.
	// The API sends it base64-encoded, and it is decoded here.
	Data []byte `json:"Data"`

	// PublicURL optionally points to the public resource for this topic
	PublicURL string `json:"PublicURL,omitempty"`
}

type ListTopicsResponse struct {
	Topics []*Topic `json:"Topics"`
}

// ListTopics lists a company's topics, optionally filtered by support platform.
func (c *Client) ListTopics(ctx context.Context, p *ListTopicsParams) (*ListTopicsResponse, error) {
	path := "topics"
	if p != nil && p.SupportPlatform != "" {
		path = fmt.Sprintf("topics?support_platform=%s", p.SupportPlatform)
	}

	rsp, err := c.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var result ListTopicsResponse
	if err := json.NewDecoder(rsp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
