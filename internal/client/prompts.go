package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// PromptForm represents the payload for creating or updating prompt definitions.
// Open WebUI's form has no is_active: create always yields an active prompt and
// update leaves the flag untouched. Use TogglePrompt to change it.
type PromptForm struct {
	Command       string         `json:"command"`
	Name          string         `json:"name"`
	Content       string         `json:"content"`
	Tags          []string       `json:"tags,omitempty"`
	Data          map[string]any `json:"data,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
	AccessControl map[string]any `json:"-"`
}

// MarshalJSON serialises the form using the API's access_grants list, derived
// from the provider's access_control representation.
func (f PromptForm) MarshalJSON() ([]byte, error) {
	type alias PromptForm
	return json.Marshal(struct {
		alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{
		alias:        alias(f),
		AccessGrants: accessControlToGrants(f.AccessControl),
	})
}

// PromptModel is returned by the prompt endpoints.
type PromptModel struct {
	ID            string         `json:"id"`
	Command       string         `json:"command"`
	Name          string         `json:"name"`
	Content       string         `json:"content"`
	IsActive      *bool          `json:"is_active,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
	Data          map[string]any `json:"data,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
	UserID        string         `json:"user_id"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
	AccessControl map[string]any `json:"-"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (r *PromptModel) UnmarshalJSON(data []byte) error {
	type alias PromptModel
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

// CreatePrompt registers a new prompt.
func (c *Client) CreatePrompt(ctx context.Context, form PromptForm) (*PromptModel, error) {
	var resp PromptModel
	if err := c.do(ctx, http.MethodPost, "prompts/create", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListPrompts returns all prompts visible to the caller.
func (c *Client) ListPrompts(ctx context.Context) ([]PromptModel, error) {
	var resp []PromptModel
	if err := c.do(ctx, http.MethodGet, "prompts/", nil, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// FindPromptByCommand returns the prompt with exactly the given command,
// whether or not it is active, or ErrNotFound. GET /prompts/ only lists active
// prompts, so this searches GET /prompts/list, which has no is_active filter
// (v0.11.0 and v0.12.0) but applies the same read-access rules. The server
// matches the query as a substring of name, command, content and owner and
// pages by 30, so the exact match is made here and further pages are fetched
// until the result is exhausted.
func (c *Client) FindPromptByCommand(ctx context.Context, command string) (*PromptModel, error) {
	seen := 0
	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("query", command)
		query.Set("page", strconv.Itoa(page))

		var resp struct {
			Items []PromptModel `json:"items"`
			Total int           `json:"total"`
		}
		if err := c.do(ctx, http.MethodGet, "prompts/list", query, nil, &resp); err != nil {
			return nil, err
		}

		for i := range resp.Items {
			if resp.Items[i].Command == command {
				return &resp.Items[i], nil
			}
		}

		seen += len(resp.Items)
		if len(resp.Items) == 0 || seen >= resp.Total {
			return nil, ErrNotFound
		}
	}
}

// GetPrompt fetches a prompt by its server identifier (UUID).
func (c *Client) GetPrompt(ctx context.Context, id string) (*PromptModel, error) {
	var resp PromptModel
	path := fmt.Sprintf("prompts/id/%s", url.PathEscape(id))
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdatePrompt updates an existing prompt by its server identifier (UUID).
func (c *Client) UpdatePrompt(ctx context.Context, id string, form PromptForm) (*PromptModel, error) {
	var resp PromptModel
	path := fmt.Sprintf("prompts/id/%s/update", url.PathEscape(id))
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// TogglePrompt flips a prompt's is_active flag and returns the updated prompt.
// The endpoint takes no value, so callers must compare the result with the
// state they want.
func (c *Client) TogglePrompt(ctx context.Context, id string) (*PromptModel, error) {
	var resp PromptModel
	path := fmt.Sprintf("prompts/id/%s/toggle", url.PathEscape(id))
	if err := c.do(ctx, http.MethodPost, path, nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeletePrompt removes a prompt by its server identifier (UUID).
func (c *Client) DeletePrompt(ctx context.Context, id string) error {
	path := fmt.Sprintf("prompts/id/%s/delete", url.PathEscape(id))
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}
