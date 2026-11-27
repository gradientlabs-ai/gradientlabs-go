package client

import (
	"encoding/json"
	"time"
)

// BackOfficeTaskStatus describes the current state of a back-office task.
type BackOfficeTaskStatus string

const (
	// BackOfficeTaskStatusPending means the task has not started yet.
	BackOfficeTaskStatusPending BackOfficeTaskStatus = "pending"

	// BackOfficeTaskStatusInProgress means the AI agent is working on the task.
	BackOfficeTaskStatusInProgress BackOfficeTaskStatus = "in-progress"

	// BackOfficeTaskStatusCompleted means the AI agent completed the task.
	BackOfficeTaskStatusCompleted BackOfficeTaskStatus = "completed"

	// BackOfficeTaskStatusFailed means the task failed.
	BackOfficeTaskStatusFailed BackOfficeTaskStatus = "failed"

	// BackOfficeTaskStatusHandedOff means the AI agent handed the task off to a
	// person.
	BackOfficeTaskStatusHandedOff BackOfficeTaskStatus = "handed-off"
)

// BackOfficeTaskHandOffReason describes why the AI agent handed a task off.
type BackOfficeTaskHandOffReason string

const (
	// BackOfficeTaskHandOffReasonProcedureEscalation means the procedure told
	// the AI agent to escalate the task.
	BackOfficeTaskHandOffReasonProcedureEscalation BackOfficeTaskHandOffReason = "procedure-escalation"

	// BackOfficeTaskHandOffReasonSubProcedureNotLive means a procedure the task
	// needed is not live.
	BackOfficeTaskHandOffReasonSubProcedureNotLive BackOfficeTaskHandOffReason = "sub-procedure-not-live"

	// BackOfficeTaskHandOffReasonOperatorHandoff means a person asked for the
	// task to be handed off.
	BackOfficeTaskHandOffReasonOperatorHandoff BackOfficeTaskHandOffReason = "operator-handoff"

	// BackOfficeTaskHandOffReasonGuardrailViolation means a guardrail stopped
	// the AI agent from continuing.
	BackOfficeTaskHandOffReasonGuardrailViolation BackOfficeTaskHandOffReason = "guardrail-violation"
)

// BackOfficeTask represents a back-office task managed by the AI agent.
type BackOfficeTask struct {
	ID             string                             `json:"id"`
	AgentID        string                             `json:"agent_id"`
	Status         BackOfficeTaskStatus               `json:"status,omitempty"`
	Input          json.RawMessage                    `json:"input"`
	Metadata       map[string]string                  `json:"metadata,omitempty"`
	Attachments    []BackOfficeTaskResponseAttachment `json:"attachments,omitempty"`
	Created        time.Time                          `json:"created"`
	Updated        time.Time                          `json:"updated,omitempty"`
	Completed      time.Time                          `json:"completed,omitempty"`
	Failed         time.Time                          `json:"failed,omitempty"`
	FailureReasons []string                           `json:"failure_reasons,omitempty"`
	HandedOff      time.Time                          `json:"handed_off,omitempty"`
	HandOffReason  BackOfficeTaskHandOffReason        `json:"hand_off_reason,omitempty"`
	Result         *BackOfficeTaskResult              `json:"result,omitempty"`
}

// BackOfficeTaskResponseAttachment is an attachment returned in a BackOfficeTask response.
type BackOfficeTaskResponseAttachment struct {
	IdempotencyKey string `json:"idempotency_key"`
	FileName       string `json:"file_name"`
	ExternalURL    string `json:"external_url,omitempty"`
}

// BackOfficeTaskResult holds the output of a completed back-office task.
type BackOfficeTaskResult struct {
	ResultType string           `json:"result_type"`
	Custom     *json.RawMessage `json:"custom,omitempty"`
}
