package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type UpsertArticleParams struct {
	// AuthorID optionally identifies the user who last edited the article
	AuthorID string `json:"author_id"`

	// ID is your identifier of choice for this article.
	ID string `json:"id"`

	// Title is the article's title. It may be empty if the article is a draft.
	Title string `json:"title"`

	// Description is an article's tagline. It may be empty.
	Description string `json:"description"`

	// Body is the main contents of an article. It may be empty if the article is a draft.
	Body string `json:"body"`

	// Visibility describes who can access this article, ranging from the
	// whole world (public) through to employees only (internal).
	Visibility Visibility `json:"visibility"`

	// TopicID optionally identifies the topic that this
	// article is associated with. If given, you must have created
	// the topic first (see: UpsertArticleTopic)
	TopicID string `json:"topic_id"`

	// PublicationStatus describes whether this article is published or not.
	PublicationStatus PublicationStatus `json:"status"`

	// Data optionally gives additional meta-data about the article.
	Data json.RawMessage `json:"data"`

	// Created is when the article was first authored.
	Created time.Time `json:"created"`

	// LastEdited is when the article was last changed.
	LastEdited time.Time `json:"last_edited"`

	// PublicURL optionally links to where this article can be read online.
	PublicURL string `json:"public_url"`

	// KnowledgeBaseIDs lists the knowledge bases this article belongs to, by
	// the IDs you gave them (see CreateKnowledgeBase). Knowledge bases control
	// which AI agents can use the article. The list can't be empty. If you
	// leave it out, the article goes into the default "public-api" knowledge
	// base, and an update that leaves it out moves the article there.
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`
}

// UpsertArticle inserts or updates a help article
func (c *Client) UpsertArticle(ctx context.Context, p *UpsertArticleParams) error {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "articles", p)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return err
	}
	return nil
}
