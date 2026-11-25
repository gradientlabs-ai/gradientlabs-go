package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// CreateFileParams are the parameters to Client.CreateFile.
type CreateFileParams struct {
	// UploadID is the upload ID that BeginFileUpload returned.
	UploadID string `json:"upload_id"`

	// FileName is the file's name, which the AI agent sees: 1 to 200 characters,
	// with no slashes and no space at either end. Keep the extension and leave
	// out personal data.
	FileName string `json:"file_name"`

	// Ref is the name a back-office test case uses to list this file as an
	// attachment. It is made of letters, digits, `_` and `-`, starts with a
	// letter or digit, and is at most 100 characters. It must be different from
	// the ref of every other file in your company, and it never changes. Leave
	// it empty to have one made from FileName.
	Ref string `json:"ref,omitempty"`
}

// CreateFile turns an uploaded file into a workspace file. The file is checked
// here: png, jpeg, webp and pdf are accepted, as are plain text and CSV if
// your workspace takes them. Calling this again with the same UploadID returns
// the same file, and a Ref that another file already holds is refused.
func (c *Client) CreateFile(ctx context.Context, p *CreateFileParams) (*File, error) {
	rsp, err := c.makeRequest(ctx, http.MethodPost, "files", p)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var file File
	if err := json.NewDecoder(rsp.Body).Decode(&file); err != nil {
		return nil, err
	}
	return &file, nil
}
