package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// filesListPageSize mirrors the backend PAGE_SIZE for GET /files/.
const filesListPageSize = 30

// FileMeta captures metadata returned with files.
type FileMeta struct {
	Name        *string `json:"name,omitempty"`
	ContentType *string `json:"content_type,omitempty"`
	Size        *int64  `json:"size,omitempty"`
}

// FileModelResponse captures file details returned by list/search/upload endpoints.
type FileModelResponse struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Hash      *string        `json:"hash,omitempty"`
	Filename  string         `json:"filename"`
	Data      map[string]any `json:"data,omitempty"`
	Meta      FileMeta       `json:"meta"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

// ContentForm updates file content.
type ContentForm struct {
	Content string `json:"content"`
}

// UploadFile uploads a file and returns metadata.
func (c *Client) UploadFile(ctx context.Context, filePath string, metadata string, process bool, processInBackground bool) (*FileModelResponse, error) {
	query := url.Values{}
	query.Set("process", fmt.Sprintf("%t", process))
	query.Set("process_in_background", fmt.Sprintf("%t", processInBackground))

	fields := map[string]string{}
	if metadata != "" {
		fields["metadata"] = metadata
	}
	files := map[string]string{"file": filePath}

	var resp FileModelResponse
	if err := c.doMultipart(ctx, http.MethodPost, "files/", query, fields, files, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// fileListResponse is the paginated envelope returned by GET /files/.
type fileListResponse struct {
	Items []FileModelResponse `json:"items"`
	Total int                 `json:"total"`
}

// ListFiles lists files available to the user, paging through the
// server-paginated GET /files/ endpoint until exhausted.
func (c *Client) ListFiles(ctx context.Context, includeContent bool) ([]FileModelResponse, error) {
	var all []FileModelResponse
	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("content", fmt.Sprintf("%t", includeContent))
		query.Set("page", fmt.Sprintf("%d", page))

		var resp fileListResponse
		if err := c.do(ctx, http.MethodGet, "files/", query, nil, &resp); err != nil {
			return nil, err
		}

		all = append(all, resp.Items...)
		if len(resp.Items) < filesListPageSize || len(all) >= resp.Total {
			break
		}
	}

	return all, nil
}

// SearchFiles searches for files by filename pattern.
func (c *Client) SearchFiles(ctx context.Context, filename string, includeContent bool, skip int, limit int) ([]FileModelResponse, error) {
	query := url.Values{}
	query.Set("filename", filename)
	query.Set("content", fmt.Sprintf("%t", includeContent))
	if skip > 0 {
		query.Set("skip", fmt.Sprintf("%d", skip))
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}

	var resp []FileModelResponse
	if err := c.do(ctx, http.MethodGet, "files/search", query, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// GetFile retrieves a file by ID.
func (c *Client) GetFile(ctx context.Context, id string) (*FileModel, error) {
	var resp FileModel
	path := "files/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteFile removes a file by ID.
func (c *Client) DeleteFile(ctx context.Context, id string) error {
	path := "files/" + url.PathEscape(id)
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

// UpdateFileContent updates the text content for a file.
func (c *Client) UpdateFileContent(ctx context.Context, id string, content string) error {
	path := "files/" + url.PathEscape(id) + "/data/content/update"
	form := ContentForm{Content: content}
	return c.do(ctx, http.MethodPost, path, nil, form, nil)
}
