package client

import "time"

// File is a file your company uploaded once. A back-office test case lists it
// as an attachment by its Ref. A file never changes: new content is a new
// upload with a new ID.
type File struct {
	// ID is the file's unique identifier.
	ID string `json:"id"`

	// Ref is the name a back-office test case uses to list the file as an
	// attachment. It never changes.
	Ref string `json:"ref"`

	// FileName is the name the file was uploaded with, which the AI agent sees.
	FileName string `json:"file_name"`

	// ContentType is the file's type, read from its contents.
	ContentType string `json:"content_type"`

	// SizeBytes is the file's size in bytes.
	SizeBytes int64 `json:"size_bytes"`

	// Created is when the file was uploaded.
	Created time.Time `json:"created"`
}
