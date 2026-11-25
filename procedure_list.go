package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

type ProcedureListParams struct {
	// Cursor is used to retrieve the next/previous page of the list.
	Cursor string `json:"cursor,omitempty"`

	// Status is used to filter the list of procedures by status.
	Status ProcedureStatus `json:"status,omitempty"`
}

type ProcedureListResponse struct {
	// Procedures contains the list of procedures.
	Procedures []*Procedure `json:"procedures"`

	// Pagination contains the pagination-related information.
	Pagination *PaginationInfo `json:"pagination"`
}

// ListProcedures lists procedures.
//
// Note: requires a `Management` API key.
func (c *Client) ListProcedures(ctx context.Context, p *ProcedureListParams) (*ProcedureListResponse, error) {
	path := "procedures"
	query := url.Values{}
	if p != nil && p.Cursor != "" {
		query.Set("cursor", p.Cursor)
	}
	if p != nil && p.Status != "" {
		query.Set("status", string(p.Status))
	}
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	rsp, err := c.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var procs ProcedureListResponse
	if err := json.NewDecoder(rsp.Body).Decode(&procs); err != nil {
		return nil, err
	}
	return &procs, nil
}
