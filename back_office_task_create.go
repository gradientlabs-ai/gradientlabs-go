package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// BackOfficeTaskCreateParams are the parameters to Client.CreateBackOfficeTask.
type BackOfficeTaskCreateParams struct {
	// ID is the external identifier for this task.
	ID string `json:"id"`

	// AgentID is the agent (agent group) that runs this task.
	AgentID string `json:"agent_id"`

	// ProcedureID is the procedure within the agent to start the task from.
	ProcedureID string `json:"procedure_id"`

	// Input is the structured input data for the task.
	Input json.RawMessage `json:"input"`

	// Created optionally sets the task creation timestamp.
	Created *time.Time `json:"created,omitempty"`

	// Metadata is optional free-form key-value metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Attachments lists any files to attach to this task.
	Attachments []BackOfficeTaskAttachment `json:"attachments,omitempty"`
}

// BackOfficeTaskAttachment is a file attachment for a back-office task creation request.
type BackOfficeTaskAttachment struct {
	// FileName is the name of the file.
	FileName string `json:"file_name"`

	// URL is the publicly accessible URL of the file.
	// Either URL or Base64Contents must be provided.
	URL string `json:"url,omitempty"`

	// Base64Contents is the base64-encoded file contents.
	// Either URL or Base64Contents must be provided.
	Base64Contents string `json:"base_64_contents,omitempty"`
}

// CreateBackOfficeTask submits a new back-office task for AI processing.
//
// Note: requires a Public API key.
func (c *Client) CreateBackOfficeTask(ctx context.Context, p BackOfficeTaskCreateParams) (*BackOfficeTask, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "back-office-tasks", p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var task BackOfficeTask
	if err := json.NewDecoder(rsp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}
