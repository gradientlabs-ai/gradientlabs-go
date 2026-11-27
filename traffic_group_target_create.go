package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// CreateTrafficGroupTargetParams are the parameters to Client.CreateTrafficGroupTarget.
type CreateTrafficGroupTargetParams struct {
	// GroupID is the unique identifier of the traffic group to add the target to.
	GroupID string `json:"-"`

	// TargetType is the type of target to add (possible values: "procedure").
	TargetType string `json:"target_type"`

	// TargetID is the unique identifier of the target to add to the group.
	TargetID string `json:"target_id"`
}

// CreateTrafficGroupTarget adds a target to a traffic group.
//
// Note: requires a `Management` API key.
func (c *Client) CreateTrafficGroupTarget(ctx context.Context, p *CreateTrafficGroupTargetParams) (*TrafficGroupTarget, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, fmt.Sprintf("traffic-groups/%s/targets", url.PathEscape(p.GroupID)), p)
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
