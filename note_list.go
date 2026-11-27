package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// ListNotesParams contains the optional parameters for listing notes.
type ListNotesParams struct {
	// Cursor is the cursor for the page to fetch, from Pagination.Next in a
	// previous response. Leave it empty to get the first page.
	Cursor string
}

// ListNotesResponse contains a page of notes.
type ListNotesResponse struct {
	// Notes is the list of notes on this page.
	Notes []*Note `json:"notes"`

	// Pagination contains the cursors for the neighbouring pages.
	Pagination *ForwardPaginationInfo `json:"pagination"`
}

// ListNotes lists the notes created by your company, a page at a time.
func (c *Client) ListNotes(ctx context.Context, p *ListNotesParams) (*ListNotesResponse, error) {
	path := "notes"
	if p != nil && p.Cursor != "" {
		path += "?" + url.Values{"cursor": {p.Cursor}}.Encode()
	}

	rsp, err := c.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var notes ListNotesResponse
	if err := json.NewDecoder(rsp.Body).Decode(&notes); err != nil {
		return nil, err
	}
	return &notes, nil
}
