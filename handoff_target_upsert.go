package client

import (
	"context"
	"net/http"
)

type UpsertHandOffTargetParams struct {
	// ID is your identifier of choice for this hand-off target. Can be anything consisting
	// of letters, numbers, or any of the following characters: `_` `-` `+` `=`.
	ID string `json:"id"`

	// Name is the hand-off target’s name. This cannot be nil.
	Name string `json:"name"`

	// Ref is a short reference for this hand-off target that never changes,
	// made of letters, digits, `_` and `-`. It starts with a letter or digit,
	// is at most 100 characters and is stored in lowercase. It is unique
	// within your company. Leave it empty to have one made from Name.
	Ref string `json:"ref,omitempty"`
}

// UpsertHandOffTarget inserts or updates a hand-off target.
//
// Note: requires a `Management` API key.
func (c *Client) UpsertHandOffTarget(ctx context.Context, p *UpsertHandOffTargetParams) error {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "hand-off-targets", p)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return err
	}
	return nil
}
