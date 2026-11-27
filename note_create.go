package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// CreateNoteParams contains the parameters for creating a note.
type CreateNoteParams struct {
	// ID is your identifier for this note.
	ID string `json:"id"`

	// Title is the note's title.
	Title string `json:"title"`

	// Body is the main contents of a note. This is mutually exclusive with WebpageURL.
	Body string `json:"body,omitempty"`

	// WebpageURL optionally points to a webpage to use as the note body.
	// This is mutually exclusive with Body.
	WebpageURL *string `json:"webpage_url,omitempty"`

	// StartTime is when the note becomes relevant.
	StartTime *time.Time `json:"start_time,omitempty"`

	// EndTime is when the note is no longer relevant.
	EndTime *time.Time `json:"end_time,omitempty"`

	// Attributes says which customers the note applies to, as a map from
	// attribute key to the attribute values it applies to. Keys and values must
	// match the attributes defined for your company. Leave it out, or send an
	// empty map, for a note that applies to every conversation.
	Attributes map[string][]string `json:"attributes,omitempty"`

	// KnowledgeBaseIDs lists the knowledge bases this note belongs to, by the
	// IDs you gave them (see CreateKnowledgeBase). The list can't be empty. If
	// you leave it out, the note goes into the default "public-api" knowledge
	// base. Note responses don't include knowledge bases.
	KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`
}

// CreateNote creates a new note.
func (c *Client) CreateNote(ctx context.Context, p *CreateNoteParams) (*Note, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "notes", p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var note Note
	if err := json.NewDecoder(rsp.Body).Decode(&note); err != nil {
		return nil, err
	}
	return &note, nil
}
