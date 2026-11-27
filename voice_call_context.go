package client

import "time"

// VoiceCallContext contains context information about the most recent voice call
// for a given phone number.
type VoiceCallContext struct {
	// StartedAt is the time the call began.
	StartedAt time.Time `json:"started_at"`

	// Summary is a short summary of the call (truncated unless IncludeLargeFields is set).
	Summary string `json:"summary,omitempty"`

	// Transcript is the full formatted conversation transcript.
	// Only populated when ReadVoiceCallContextParams.IncludeLargeFields is true.
	Transcript string `json:"transcript,omitempty"`

	// HandoffReason is the reason the call was handed off to a human agent, if at all.
	HandoffReason string `json:"handoff_reason,omitempty"`

	// LastExecutedProcedure is the name of the last procedure executed during the call.
	LastExecutedProcedure string `json:"last_executed_procedure,omitempty"`

	// LastExecutedProcedureURL is a link to the procedure in the Gradient Labs UI.
	LastExecutedProcedureURL string `json:"last_executed_procedure_url,omitempty"`

	// GradientLabsURL is a link to the call conversation in the Gradient Labs UI.
	GradientLabsURL string `json:"gradient_labs_url,omitempty"`

	// LiveHandoffURL is a link a human agent opens to take over the call. Only
	// set when the call is being handed off live, or for the test phone number.
	LiveHandoffURL string `json:"live_handoff_url,omitempty"`
}
