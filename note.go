package client

import "time"

// NoteStatus describes whether a note is draft, live, or deleted.
type NoteStatus string

const (
	// NoteStatusDraft means the note is being written or edited and is not published.
	NoteStatusDraft NoteStatus = "draft"

	// NoteStatusLive means the note is published and available.
	NoteStatusLive NoteStatus = "live"

	// NoteStatusDeleted means the note has been deleted. Use the DeleteNote() method
	// to delete a note.
	NoteStatusDeleted NoteStatus = "deleted"
)

// Note represents a note in the Gradient Labs system.
type Note struct {
	// ID is the Gradient Labs ID for this note. This is
	// created by Gradient Labs.
	ID string `json:"gradient_labs_id"`

	// ExternalID is your identifier for this note.
	ExternalID string `json:"id"`

	// Title is the note's title.
	Title string `json:"title"`

	// Body is the main contents of the note.
	Body string `json:"body,omitempty"`

	// WebpageURL optionally points to a webpage to use as the note body.
	WebpageURL *string `json:"url,omitempty"`

	// StartTime is when the note becomes relevant.
	StartTime *time.Time `json:"valid_from,omitempty"`

	// EndTime is when the note is no longer relevant.
	EndTime *time.Time `json:"valid_to,omitempty"`

	// LastModifiedBy identifies who last modified the note.
	LastModifiedBy string `json:"last_modified_by"`

	// Created is when the note was created.
	Created time.Time `json:"created"`

	// Updated is when the note was last updated.
	Updated time.Time `json:"updated"`

	// Status describes whether the note is draft, live, or deleted.
	Status NoteStatus `json:"status"`

	// Attributes says which customers the note applies to, as a map from
	// attribute key to the attribute values it applies to. It is empty for a
	// note that applies to every conversation.
	Attributes map[string][]string `json:"attributes,omitempty"`
}
