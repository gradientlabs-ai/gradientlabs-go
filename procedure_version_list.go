package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GatedConfig holds configuration for a gated procedure version.
type GatedConfig struct {
	MaxDailyConversations int `json:"MaxDailyConversations"`
}

type ProcedureVersion struct {
	// Name is the user-given name of the procedure at the time of this version.
	Name string `json:"Name"`
	// Description of the procedure at the time of this version.
	Description string `json:"Description"`
	// Version is a numeric identifier for the version. Version is incremented every
	// time a new version of the procedure is saved.
	Version int `json:"Version"`
	// Author is the ID of the user who created this version of the procedure.
	Author string `json:"Author"`
	// Created is the time at which this version of the procedure was created.
	Created time.Time `json:"Created"`
	// Gated indicates whether the procedure version is a gated version that is used before "live",
	// within the daily limit defined in GatedConfig.
	Gated bool `json:"Gated"`
	// GatedConfig defines how the gated version is limited. GatedConfig is relevant only when Gated == true.
	GatedConfig *GatedConfig `json:"GatedConfig"`
	// Live indicates whether the procedure version is the "production" version used by the agent by default,
	// when there are no gated versions or all gated versions have exceeded their daily limit.
	Live bool `json:"Live"`
}

type ListProcedureVersionsResponse struct {
	Versions []*ProcedureVersion `json:"Versions"`
}

func (c *Client) ListProcedureVersions(ctx context.Context, procedureID string) (*ListProcedureVersionsResponse, error) {
	rsp, err := c.makeRequest(ctx, http.MethodGet, fmt.Sprintf("procedures/%s/versions", procedureID), nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var result ListProcedureVersionsResponse
	if err := json.NewDecoder(rsp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
