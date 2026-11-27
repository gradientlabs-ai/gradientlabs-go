package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ReadNote reads a note by the ID you gave it when you created it.
func (c *Client) ReadNote(ctx context.Context, noteID string) (*Note, error) {
	rsp, err := c.makeRequest(ctx, http.MethodGet, fmt.Sprintf("notes/%s", url.PathEscape(noteID)), nil)
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
