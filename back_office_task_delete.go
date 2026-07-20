package client

import (
	"context"
	"fmt"
	"net/http"
)

// DeleteBackOfficeTask deletes a back-office task by its ID. The task must not
// be in an in-progress state, and on-demand deletion must be enabled.
//
// Note: requires a `Management` API key.
func (c *Client) DeleteBackOfficeTask(ctx context.Context, taskID string) error {
	rsp, err := c.makeRequest(ctx, http.MethodDelete, fmt.Sprintf("back-office-tasks/%s", taskID), nil)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return err
	}
	return nil
}
