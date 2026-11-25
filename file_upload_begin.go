package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// BeginFileUploadParams are the parameters to Client.BeginFileUpload.
type BeginFileUploadParams struct {
	// ContentType is the Content-Type header the upload sends. Leave it empty
	// to send "application/octet-stream": the file's type is read from its
	// contents either way.
	ContentType string `json:"content_type,omitempty"`
}

// FileUpload says where and how to upload a file.
type FileUpload struct {
	// UploadID is what you pass to CreateFile once the file has been uploaded.
	UploadID string `json:"upload_id"`

	// UploadURL is the URL to send the file to as an HTTP PUT, with every header
	// in Headers.
	UploadURL string `json:"upload_url"`

	// ContentType is the Content-Type header the PUT must send.
	ContentType string `json:"content_type"`

	// Headers are the headers the PUT must send, Content-Type included.
	// Storage refuses a PUT without them, or a file over MaxBytes.
	Headers map[string]string `json:"headers"`

	// ExpiresAt is when UploadURL stops accepting the file.
	ExpiresAt time.Time `json:"expires_at"`

	// MaxBytes is the largest file that can be uploaded, in bytes.
	MaxBytes int64 `json:"max_bytes"`
}

// BeginFileUpload starts uploading a file to your workspace. PUT the file to
// the returned UploadURL, then call CreateFile with the UploadID. The file
// never passes through this API.
func (c *Client) BeginFileUpload(ctx context.Context, p *BeginFileUploadParams) (*FileUpload, error) {
	if p == nil {
		p = &BeginFileUploadParams{}
	}

	rsp, err := c.makeRequest(ctx, http.MethodPost, "file-uploads", p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var upload FileUpload
	if err := json.NewDecoder(rsp.Body).Decode(&upload); err != nil {
		return nil, err
	}
	return &upload, nil
}
