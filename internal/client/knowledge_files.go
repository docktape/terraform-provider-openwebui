package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// KnowledgeFileIDForm associates a file with a knowledge base.
type KnowledgeFileIDForm struct {
	FileID string `json:"file_id"`
}

// KnowledgeFileListResponse captures paginated file results for a knowledge base.
type KnowledgeFileListResponse struct {
	Items []FileUserResponse `json:"items"`
	Total int                `json:"total"`
}

// FileUserResponse captures file details including user metadata.
type FileUserResponse struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Hash      *string        `json:"hash,omitempty"`
	Filename  string         `json:"filename"`
	Data      map[string]any `json:"data,omitempty"`
	Meta      FileMeta       `json:"meta"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
	User      *User          `json:"user,omitempty"`
}

// KnowledgeFilesLookupLimit is a large page size used when looking up a
// specific file within a knowledge base by ID, to avoid the server's
// default 30-item page size (PAGE_ITEM_COUNT) hiding files attached beyond
// the first page. The backend's limit override on GET /knowledge/{id}/files
// is admin-only ("Page size (admin only). Defaults to 30.") — safe here
// because this provider always authenticates with an admin token.
const KnowledgeFilesLookupLimit = 10000

// ListKnowledgeFiles retrieves file attachments for a knowledge base.
// Pass limit=0 to use the server's default page size.
func (c *Client) ListKnowledgeFiles(ctx context.Context, knowledgeID string, queryValue string, viewOption string, orderBy string, direction string, page int, limit int) (*KnowledgeFileListResponse, error) {
	query := url.Values{}
	if queryValue != "" {
		query.Set("query", queryValue)
	}
	if viewOption != "" {
		query.Set("view_option", viewOption)
	}
	if orderBy != "" {
		query.Set("order_by", orderBy)
	}
	if direction != "" {
		query.Set("direction", direction)
	}
	if page > 0 {
		query.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}

	var resp KnowledgeFileListResponse
	path := fmt.Sprintf("knowledge/%s/files", url.PathEscape(knowledgeID))
	if err := c.do(ctx, http.MethodGet, path, query, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// AddKnowledgeFile associates a file with a knowledge base.
func (c *Client) AddKnowledgeFile(ctx context.Context, knowledgeID string, fileID string) (*KnowledgeFilesResponse, error) {
	var resp KnowledgeFilesResponse
	form := KnowledgeFileIDForm{FileID: fileID}
	path := fmt.Sprintf("knowledge/%s/file/add", url.PathEscape(knowledgeID))
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateKnowledgeFile updates a file association within a knowledge base.
func (c *Client) UpdateKnowledgeFile(ctx context.Context, knowledgeID string, fileID string) (*KnowledgeFilesResponse, error) {
	var resp KnowledgeFilesResponse
	form := KnowledgeFileIDForm{FileID: fileID}
	path := fmt.Sprintf("knowledge/%s/file/update", url.PathEscape(knowledgeID))
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RemoveKnowledgeFile detaches a file from a knowledge base.
func (c *Client) RemoveKnowledgeFile(ctx context.Context, knowledgeID string, fileID string, deleteFile bool) (*KnowledgeFilesResponse, error) {
	var resp KnowledgeFilesResponse
	form := KnowledgeFileIDForm{FileID: fileID}
	query := url.Values{"delete_file": []string{fmt.Sprintf("%t", deleteFile)}}
	path := fmt.Sprintf("knowledge/%s/file/remove", url.PathEscape(knowledgeID))
	if err := c.do(ctx, http.MethodPost, path, query, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
