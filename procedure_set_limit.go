package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ProcedureLimitParams struct {
	// HasDailyLimit identifies whether the procedure should have a limit
	HasDailyLimit bool `json:"has_daily_limit,omitempty"`

	// MaxDailyConversations is the maximum number of conversations that
	// can use this procedure on a given day.
	MaxDailyConversations int `json:"max_daily_conversations,omitempty"`
}

// SetProcedureLimit updates the daily usage limit of a procedure.
//
// Note: requires a `Management` API key.
func (c *Client) SetProcedureLimit(ctx context.Context, procedureID string, p *ProcedureLimitParams) (*Procedure, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, fmt.Sprintf("procedure/%s/limit", procedureID), p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var response struct {
		Procedure Procedure `json:"procedure"`
	}
	if err := json.NewDecoder(rsp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response.Procedure, nil
}
