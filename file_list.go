package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// ListFilesParams contains the optional parameters for listing files.
type ListFilesParams struct {
	// Cursor is the cursor for the page to fetch, from Pagination.Next in a
	// previous response. Leave it empty to get the first page.
	Cursor string
}

// ListFilesResponse contains a page of files.
type ListFilesResponse struct {
	// Files is your company's files on this page, newest first. A deleted file
	// isn't listed.
	Files []*File `json:"files"`

	// Pagination contains the cursors for the neighbouring pages.
	Pagination *ForwardPaginationInfo `json:"pagination"`
}

// ListFiles lists the files uploaded to your workspace, a page at a time.
func (c *Client) ListFiles(ctx context.Context, p *ListFilesParams) (*ListFilesResponse, error) {
	path := "files"
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

	var files ListFilesResponse
	if err := json.NewDecoder(rsp.Body).Decode(&files); err != nil {
		return nil, err
	}
	return &files, nil
}
