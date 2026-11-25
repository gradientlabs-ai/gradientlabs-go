package client

type PaginationInfo struct {
	// Next is a cursor to retrieve the next page of results.
	Next *string `json:"next,omitempty"`

	// Prev is a cursor to retrieve the previous page of results.
	Prev *string `json:"prev,omitempty"`
}

// ForwardPaginationInfo holds the cursor for the next page.
type ForwardPaginationInfo struct {
	Next *string `json:"next,omitempty"`
}
