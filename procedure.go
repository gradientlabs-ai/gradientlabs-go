package client

import "time"

type ProcedureStatus string

const (
	// ProcedureStatusDraft indicates the procedure has been saved as a draft, but
	// won't be used in real conversations until it is promoted to live.
	ProcedureStatusDraft ProcedureStatus = "draft"

	// ProcedureStatusLive indicates the procedure is live and will be used in real
	// conversations.
	ProcedureStatusLive ProcedureStatus = "live"

	// ProcedureStatusUnsaved indicates the procedure has changes that have not
	// been saved yet.
	ProcedureStatusUnsaved ProcedureStatus = "unsaved"

	// ProcedureStatusArchived indicates the procedure has been archived.
	ProcedureStatusArchived ProcedureStatus = "archived"
)

type UserDetails struct {
	// Email identifies the user.
	Email string `json:"email"`
}

type Procedure struct {
	// ID uniquely identifies the procedure.
	ID string `json:"id"`

	// Name is the user-given name of the procedure.
	Name string `json:"name"`

	// Description is the user-given description for the procedure
	Description string `json:"description"`

	// Status is the overall status of the procedure.
	Status ProcedureStatus `json:"status"`

	// Created is the time at which the procedure was originally created.
	Created time.Time `json:"created"`

	// Updated is the time at which the procedure's status, metadata, or current
	// revision was last changed. It does *not* reflect revisions created as part
	// of testing unsaved changes.
	Updated time.Time `json:"updated"`

	// IsDailyLimited is true if this procedure can only be executed for a maximum
	// number of conversations in a given day (defined below).
	IsDailyLimited bool `json:"has_daily_limit"`

	// MaxDailyConversations is the maximum number of conversations that a procedure
	// can be used in on a given day, when it is rate limited.
	MaxDailyConversations int `json:"max_daily_conversations,omitempty"`
}
