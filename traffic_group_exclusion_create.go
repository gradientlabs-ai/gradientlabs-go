package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// CreateTrafficGroupExclusionParams are the parameters to Client.CreateTrafficGroupExclusion.
type CreateTrafficGroupExclusionParams struct {
	// GroupID is the unique identifier of the traffic group.
	GroupID string `json:"-"`

	// TargetType is the type of target to exclude (e.g. "procedure").
	TargetType string `json:"target_type"`

	// TargetID is the unique identifier of the target to exclude from the group.
	TargetID string `json:"target_id"`
}

// CreateTrafficGroupExclusion excludes a target from a traffic group.
//
// Note: requires a Management API key.
func (c *Client) CreateTrafficGroupExclusion(ctx context.Context, p *CreateTrafficGroupExclusionParams) (*TrafficGroupTarget, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, fmt.Sprintf("traffic-groups/%s/exclusions", url.PathEscape(p.GroupID)), p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var target TrafficGroupTarget
	if err := json.NewDecoder(rsp.Body).Decode(&target); err != nil {
		return nil, err
	}
	return &target, nil
}
