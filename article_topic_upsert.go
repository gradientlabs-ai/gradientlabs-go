package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type UpsertArticleTopicParams struct {
	// ID is your identifier for this topic
	ID string `json:"id"`

	// ParentID is the identifier for this topic's parent topic (if any).
	ParentID string `json:"parent_id"`

	// Name is the topic's name. This cannot be nil.
	Name string `json:"name"`

	// Description is an topic's tagline. It may be empty.
	Description string `json:"description"`

	// Visibility describes who can see this topic, ranging from the
	// whole world (public) through to employees only (internal).
	Visibility Visibility `json:"visibility"`

	// PublicationStatus describes whether this topic is published or not.
	PublicationStatus PublicationStatus `json:"status"`

	// Data optionally gives additional meta-data about the topic.
	Data json.RawMessage `json:"data"`

	// Created is when the topic was first created.
	Created time.Time `json:"created"`

	// LastEdited is when the topic was last changed.
	LastEdited time.Time `json:"last_edited"`

	// PublicURL optionally links to where this topic can be browsed online.
	PublicURL string `json:"public_url"`
}

func (c *Client) UpsertArticleTopic(ctx context.Context, p *UpsertArticleTopicParams) error {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "topics", p)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return err
	}
	return nil
}
