package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// KnowledgeForm models the payload for creating or updating knowledge records.
type KnowledgeForm struct {
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	AccessControl map[string]any `json:"-"`
	Data          map[string]any `json:"data,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
}

// MarshalJSON serialises the form using the API's access_grants list, derived
// from the provider's access_control representation.
func (f KnowledgeForm) MarshalJSON() ([]byte, error) {
	type alias KnowledgeForm
	return json.Marshal(struct {
		alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{
		alias:        alias(f),
		AccessGrants: accessControlToGrants(f.AccessControl),
	})
}

// FileModel represents a file associated with a knowledge base entry.
type FileModel struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Filename  string         `json:"filename"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
	Hash      *string        `json:"hash,omitempty"`
	Path      *string        `json:"path,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// KnowledgeResponse captures the core knowledge object returned by create and update endpoints.
type KnowledgeResponse struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
	AccessControl map[string]any `json:"-"`
	Data          map[string]any `json:"data,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
	Files         []FileModel    `json:"files,omitempty"`
}

func (r *KnowledgeResponse) UnmarshalJSON(data []byte) error {
	type alias KnowledgeResponse
	aux := struct {
		*alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.AccessControl = grantsToAccessControl(aux.AccessGrants)
	return nil
}

// KnowledgeFilesResponse is returned by the knowledge detail endpoint and includes file metadata.
type KnowledgeFilesResponse struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
	AccessControl map[string]any `json:"-"`
	Data          map[string]any `json:"data,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
	Files         []FileModel    `json:"files"`
}

func (r *KnowledgeFilesResponse) UnmarshalJSON(data []byte) error {
	type alias KnowledgeFilesResponse
	aux := struct {
		*alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.AccessControl = grantsToAccessControl(aux.AccessGrants)
	return nil
}

// CreateKnowledge provisions a new knowledge base entry.
func (c *Client) CreateKnowledge(ctx context.Context, form KnowledgeForm) (*KnowledgeResponse, error) {
	var resp KnowledgeResponse
	if err := c.do(ctx, http.MethodPost, "knowledge/create", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetKnowledge retrieves a knowledge record by identifier.
func (c *Client) GetKnowledge(ctx context.Context, id string) (*KnowledgeFilesResponse, error) {
	var resp KnowledgeFilesResponse
	path := fmt.Sprintf("knowledge/%s", url.PathEscape(id))
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// KnowledgeListItem represents a knowledge entry returned by the list endpoint.
type KnowledgeListItem struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
	AccessControl map[string]any `json:"-"`
	Data          map[string]any `json:"data,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
}

func (r *KnowledgeListItem) UnmarshalJSON(data []byte) error {
	type alias KnowledgeListItem
	aux := struct {
		*alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.AccessControl = grantsToAccessControl(aux.AccessGrants)
	return nil
}

// ListKnowledge retrieves all knowledge entries visible to the caller.
func (c *Client) ListKnowledge(ctx context.Context) ([]KnowledgeListItem, error) {
	var resp []KnowledgeListItem
	if err := c.do(ctx, http.MethodGet, "knowledge/list", nil, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// UpdateKnowledge mutates an existing knowledge record.
func (c *Client) UpdateKnowledge(ctx context.Context, id string, form KnowledgeForm) (*KnowledgeResponse, error) {
	var resp KnowledgeResponse
	path := fmt.Sprintf("knowledge/%s/update", url.PathEscape(id))
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteKnowledge removes a knowledge record.
func (c *Client) DeleteKnowledge(ctx context.Context, id string) error {
	path := fmt.Sprintf("knowledge/%s/delete", url.PathEscape(id))
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}
