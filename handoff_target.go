package client

type HandOffTarget struct {
	// ID is your identifier of choice for this hand-off target. Can be anything consisting
	// of letters, numbers, or any of the following characters: `_` `-` `+` `=`.
	ID string `json:"id"`

	// Name is the hand-off target’s name.
	Name string `json:"name"`

	// Ref is a short reference for this hand-off target that never changes.
	// It is unique within your company.
	Ref string `json:"ref"`
}
