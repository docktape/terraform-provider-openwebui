package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// PromptForm represents the payload for creating or updating prompt definitions.
type PromptForm struct {
	Command       string         `json:"command"`
	Name          string         `json:"name"`
	Content       string         `json:"content"`
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

// DeletePrompt removes a prompt by its server identifier (UUID).
func (c *Client) DeletePrompt(ctx context.Context, id string) error {
	path := fmt.Sprintf("prompts/id/%s/delete", url.PathEscape(id))
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}
