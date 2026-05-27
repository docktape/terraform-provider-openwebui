package client

import (
	"context"
	"net/http"
	"net/url"
)

// FunctionMeta captures descriptive metadata for a function.
type FunctionMeta struct {
	Description *string        `json:"description,omitempty"`
	Manifest    map[string]any `json:"manifest,omitempty"`
}

// FunctionForm is the create/update payload for a function.
type FunctionForm struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Content string       `json:"content"`
	Meta    FunctionMeta `json:"meta"`
}

// FunctionModel is the full function record returned by read/update/toggle.
// The create response omits Content; callers that need it must call GetFunction.
type FunctionModel struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	Content   string       `json:"content"`
	Meta      FunctionMeta `json:"meta"`
	IsActive  bool         `json:"is_active"`
	IsGlobal  bool         `json:"is_global"`
	UpdatedAt int64        `json:"updated_at"`
	CreatedAt int64        `json:"created_at"`
}

// CreateFunction provisions a new function. The response omits content.
func (c *Client) CreateFunction(ctx context.Context, form FunctionForm) (*FunctionModel, error) {
	var resp FunctionModel
	if err := c.do(ctx, http.MethodPost, "functions/create", nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetFunction retrieves a function by ID, including its content.
func (c *Client) GetFunction(ctx context.Context, id string) (*FunctionModel, error) {
	var resp FunctionModel
	path := "functions/id/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteFunction removes a function by ID.
func (c *Client) DeleteFunction(ctx context.Context, id string) error {
	path := "functions/id/" + url.PathEscape(id) + "/delete"
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

// UpdateFunction updates an existing function by ID.
func (c *Client) UpdateFunction(ctx context.Context, id string, form FunctionForm) (*FunctionModel, error) {
	var resp FunctionModel
	path := "functions/id/" + url.PathEscape(id) + "/update"
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ToggleFunction flips a function's is_active flag.
func (c *Client) ToggleFunction(ctx context.Context, id string) (*FunctionModel, error) {
	var resp FunctionModel
	path := "functions/id/" + url.PathEscape(id) + "/toggle"
	if err := c.do(ctx, http.MethodPost, path, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ToggleFunctionGlobal flips a function's is_global flag.
func (c *Client) ToggleFunctionGlobal(ctx context.Context, id string) (*FunctionModel, error) {
	var resp FunctionModel
	path := "functions/id/" + url.PathEscape(id) + "/toggle/global"
	if err := c.do(ctx, http.MethodPost, path, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
