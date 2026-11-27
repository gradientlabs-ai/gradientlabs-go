package client

import "time"

// ParameterType determines the data type of a parameter.
type ParameterType string

const (
	// ParameterTypeString indicates the parameter accepts string/text values.
	ParameterTypeString ParameterType = "string"

	// ParameterTypeStringArray indicates the parameter accepts a list of
	// strings. The API rejects it when you create or update a tool, but it can
	// appear on tools that already exist.
	ParameterTypeStringArray ParameterType = "string_array"

	// ParameterTypeJSON indicates the parameter accepts a JSON value. The API
	// rejects it when you create or update a tool, but it can appear on tools
	// that already exist.
	ParameterTypeJSON ParameterType = "json"

	// ParameterTypeInteger indicates the parameter accepts whole numbers.
	ParameterTypeInteger ParameterType = "integer"

	// ParameterTypeFloat indicates the parameter accepts decimal numbers.
	ParameterTypeFloat ParameterType = "float"

	// ParameterTypeBoolean indicates the parameter accepts true or false.
	ParameterTypeBoolean ParameterType = "boolean"

	// ParameterTypeDate indicates the parameter accepts dates.
	ParameterTypeDate ParameterType = "date"

	// ParameterTypeTimestamp indicates the parameter accepts timestamps.
	ParameterTypeTimestamp ParameterType = "timestamp"

	// ParameterTypeDuration indicates the parameter accepts durations.
	ParameterTypeDuration ParameterType = "duration"

	// ParameterTypeAttachment indicates the parameter accepts a file. Set
	// TransferMode to say how the file reaches the tool.
	ParameterTypeAttachment ParameterType = "attachment"
)

// ParameterSource says where a parameter's value can come from.
type ParameterSource string

const (
	// ParameterSourceLLM means the AI agent chooses the value from the
	// conversation.
	ParameterSourceLLM ParameterSource = "llm"

	// ParameterSourceLiteral means the value is a fixed value set in a
	// procedure.
	ParameterSourceLiteral ParameterSource = "literal"

	// ParameterSourceResource means the value comes from a resource attached to
	// the conversation.
	ParameterSourceResource ParameterSource = "resource"
)

// TransferMode says how an attachment parameter's file reaches the tool.
type TransferMode string

const (
	// TransferModeBase64 delivers the file as a base64-encoded string.
	TransferModeBase64 TransferMode = "base64"

	// TransferModeTemporaryURL delivers the file as a temporary URL.
	TransferModeTemporaryURL TransferMode = "temporary_url"
)

// BodyEncoding determines how the HTTP body will be encoded.
type BodyEncoding string

const (
	// BodyEncodingForm indicates the body will be encoded using URL encoding.
	BodyEncodingForm BodyEncoding = "application/x-www-form-urlencoded"

	// BodyEncodingJSON indicates the body will be encoded as JSON.
	BodyEncodingJSON BodyEncoding = "application/json"
)

type ToolList struct {
	Tools []*Tool `json:"tools"`
}

type Tool struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters"`

	// ParameterSets are conditional groups of parameters, chosen by the value
	// of a discriminator parameter.
	ParameterSets []ToolParameterSet `json:"parameter_sets,omitempty"`

	// Webhook, HTTP and Async say how the tool runs. When creating a tool, send
	// exactly one of them, or set Draft and send none.
	Webhook *ToolWebhookConfiguration `json:"webhook,omitempty"`
	HTTP    *HTTPDefinition           `json:"http,omitempty"`
	Async   *AsyncDefinition          `json:"async,omitempty"`

	// MCP says the tool runs on a Model Context Protocol server. It is only
	// returned when reading tools: creating a tool with it fails.
	MCP *MCPConfiguration `json:"mcp,omitempty"`

	// Draft marks the tool as still being built and not ready for the AI agent
	// to use. When true, send none of Webhook, HTTP or Async.
	Draft bool `json:"draft,omitempty"`

	// BranchID is only read when creating a tool: the ID of the branch to
	// create the tool on. Send this or BranchName. The tool goes live when the
	// branch is merged, and ID is empty until the branch has taken in the tool.
	BranchID string `json:"branch_id,omitempty"`

	// BranchName is only read when creating a tool: the git branch name of the
	// branch to create the tool on, as an alternative to BranchID.
	BranchName string `json:"branch_name,omitempty"`
}

type ToolParameter struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        ParameterType     `json:"type"`
	Required    *bool             `json:"required,omitempty"`
	Options     []ParameterOption `json:"options,omitempty"`

	// AllowedSources says where the parameter's value can come from. At least
	// one is required.
	AllowedSources []ParameterSource `json:"allowed_sources"`

	// TransferMode says how an attachment parameter's file reaches the tool.
	// It is required when Type is ParameterTypeAttachment, and not allowed for
	// any other type.
	TransferMode TransferMode `json:"transfer_mode,omitempty"`
}

// ToolParameterSet is a group of parameters that become active when a
// discriminator parameter has a given value.
type ToolParameterSet struct {
	// DiscriminatorParameterName is the name of the top-level parameter whose
	// value decides whether this set is active.
	DiscriminatorParameterName string `json:"discriminator_parameter_name"`

	// DiscriminatorValue is the value of the discriminator parameter that
	// activates this set.
	DiscriminatorValue string `json:"discriminator_value"`

	// Parameters are the extra parameters that become available when this set
	// is active.
	Parameters []ToolParameter `json:"parameters"`
}

// MCPConfiguration says a tool runs on a Model Context Protocol server.
type MCPConfiguration struct {
	// ServerID is the ID of the MCP server that runs the tool.
	ServerID string `json:"server_id"`

	// ExternalToolName is the tool's name on the MCP server.
	ExternalToolName string `json:"external_tool_name"`
}

type ParameterOption struct {
	Value string `json:"value"`
	Text  string `json:"text"`
}

type ToolWebhookConfiguration struct {
	Name string `json:"name"`
}

type HTTPDefinition struct {
	Method          string              `json:"method"`
	URLTemplate     string              `json:"url_template"`
	HeaderTemplates map[string]string   `json:"header_templates,omitempty"`
	Body            *HTTPBodyDefinition `json:"body,omitempty"`

	// Retry changes how the request is retried. Leave it out, or any of its
	// fields, to keep the default retries.
	Retry *HTTPRetryDefinition `json:"retry,omitempty"`
}

// HTTPRetryDefinition overrides how an HTTP request is retried. All attempts
// together, including the waits between them, are limited to 4 minutes.
type HTTPRetryDefinition struct {
	// MaxAttempts is the most times the request is attempted, from 1 to 10.
	// A request other than GET is never retried after a connection error.
	MaxAttempts *int `json:"max_attempts,omitempty"`

	// RequestTimeoutSeconds is how many seconds each attempt waits for a
	// response, from 1 to 90. The default is 10.
	RequestTimeoutSeconds *int `json:"request_timeout_seconds,omitempty"`
}

type HTTPBodyDefinition struct {
	Encoding           BodyEncoding      `json:"encoding"`
	JSONTemplate       string            `json:"json_template,omitempty"`
	FormFieldTemplates map[string]string `json:"form_field_templates,omitempty"`
}

type AsyncDefinition struct {
	// StartExecutionTool is the tool that will be executed to start the async operation. It should return a result
	// fairly quickly that optionally contains information about the work that has been kicked off. It is executed like
	// a normal HTTP/Webhook tool.
	StartExecutionTool ChildTool `json:"start_execution_tool"`

	// CancelExecutionTool is an optional tool that cancels the async operation.
	// It runs when the AI agent cancels the operation or the conversation is
	// resolved. The AI agent never sees what it returns.
	CancelExecutionTool *ChildTool `json:"cancel_execution_tool,omitempty"`

	// Timeout is the maximum time the async operation is allowed to run. If it exceeds this time, we will stop waiting
	// for the response and the agent will be notified that the async operation has timed out.
	Timeout time.Duration `json:"timeout"`
}

// ChildTool represents a tool that is part of a parent tool's configuration. It does not have its own ID, company ID,
// version, dates, authors, etc. as these are all inherited from the parent tool. In its current form it effectively
// maps to an action definition.
//
// Note that the child tool does not currently support parameters, these are decided by the parent tool and passed to
// the child tool when it is executed.
type ChildTool struct {
	Webhook *ToolWebhookConfiguration `json:"webhook,omitempty"`
	HTTP    *HTTPDefinition           `json:"http,omitempty"`
}
