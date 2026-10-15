package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// SkillMeta captures descriptive metadata for a skill.
type SkillMeta struct {
	Tags []string `json:"tags"`
}

// MarshalJSON always serialises tags as a list. The backend accepts a null
// tags value but then echoes null back, so a concrete list keeps round-trips
// stable.
func (m SkillMeta) MarshalJSON() ([]byte, error) {
	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}
	return json.Marshal(struct {
		Tags []string `json:"tags"`
	}{Tags: tags})
}

// SkillForm represents the payload for creating or updating skills. The update
// endpoint replaces every field, so unset values must still be sent.
type SkillForm struct {
	ID string `json:"id"`
	// Name is unique across the instance (database constraint).
	Name string `json:"name"`
	// Description is nullable on the backend; nil is sent as an explicit null so
	// that an update clears a previously stored description.
	Description   *string        `json:"description"`
	Content       string         `json:"content"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      bool           `json:"is_active"`
	AccessControl map[string]any `json:"-"`
}

// MarshalJSON serialises the form using the API's access_grants list, derived
// from the provider's access_control representation. An empty list is always
// sent: the backend leaves existing grants untouched when the key is null.
func (f SkillForm) MarshalJSON() ([]byte, error) {
	type alias SkillForm
	return json.Marshal(struct {
		alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{
		alias:        alias(f),
		AccessGrants: accessControlToGrants(f.AccessControl),
	})
}

// SkillResponse captures basic skill details (create response).
type SkillResponse struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   *string        `json:"description"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      bool           `json:"is_active"`
	AccessControl map[string]any `json:"-"`
	UpdatedAt     int64          `json:"updated_at"`
	CreatedAt     int64          `json:"created_at"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (r *SkillResponse) UnmarshalJSON(data []byte) error {
	type alias SkillResponse
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

// SkillModel includes full skill content (update and export responses).
type SkillModel struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   *string        `json:"description"`
	Content       string         `json:"content"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      bool           `json:"is_active"`
	AccessControl map[string]any `json:"-"`
	UpdatedAt     int64          `json:"updated_at"`
	CreatedAt     int64          `json:"created_at"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (r *SkillModel) UnmarshalJSON(data []byte) error {
	type alias SkillModel
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

// SkillAccessResponse captures skill details with access metadata.
//
// Content is not part of the declared response model, but the backend's
// SkillUserResponse allows extra fields and the router builds the response from
// the full skill, so `content` is present on the wire in practice (verified on
// v0.11.0). It is captured when present; callers must fall back to
// ExportSkills when it is nil.
type SkillAccessResponse struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   *string        `json:"description"`
	Content       *string        `json:"content,omitempty"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      bool           `json:"is_active"`
	AccessControl map[string]any `json:"-"`
	UpdatedAt     int64          `json:"updated_at"`
	CreatedAt     int64          `json:"created_at"`
	User          *User          `json:"user,omitempty"`
	WriteAccess   *bool          `json:"write_access,omitempty"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (r *SkillAccessResponse) UnmarshalJSON(data []byte) error {
	type alias SkillAccessResponse
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

// SkillAccessListResponse is the paginated envelope returned by GET /skills/list.
type SkillAccessListResponse struct {
	Items []SkillAccessResponse `json:"items"`
	Total int                   `json:"total"`
}

// SkillUserResponse captures skill details including user metadata (list view).
type SkillUserResponse struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   *string        `json:"description"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      bool           `json:"is_active"`
	AccessControl map[string]any `json:"-"`
	UpdatedAt     int64          `json:"updated_at"`
	CreatedAt     int64          `json:"created_at"`
	User          *User          `json:"user,omitempty"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (r *SkillUserResponse) UnmarshalJSON(data []byte) error {
	type alias SkillUserResponse
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

// CreateSkill provisions a new skill. The backend lowercases the id and
// replaces spaces with hyphens, so callers should supply an already-normalised id.
func (c *Client) CreateSkill(ctx context.Context, form SkillForm) (*SkillResponse, error) {
	var resp SkillResponse
	if err := c.do(ctx, http.MethodPost, "skills/create", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetSkill retrieves a skill by ID. A missing skill yields ErrNotFound (HTTP 404).
func (c *Client) GetSkill(ctx context.Context, id string) (*SkillAccessResponse, error) {
	var resp SkillAccessResponse
	path := "skills/id/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateSkill replaces an existing skill by ID.
func (c *Client) UpdateSkill(ctx context.Context, id string, form SkillForm) (*SkillModel, error) {
	var resp SkillModel
	path := "skills/id/" + url.PathEscape(id) + "/update"
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteSkill removes a skill by ID.
func (c *Client) DeleteSkill(ctx context.Context, id string) error {
	path := "skills/id/" + url.PathEscape(id) + "/delete"
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

// ListSkills returns every skill visible to the caller (GET /skills/).
func (c *Client) ListSkills(ctx context.Context) ([]SkillUserResponse, error) {
	var resp []SkillUserResponse
	if err := c.do(ctx, http.MethodGet, "skills/", nil, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// ListSkillAccess returns one page (30 items) of skill access summaries from
// GET /skills/list. Pages start at 1; a value below 1 requests the first page.
func (c *Client) ListSkillAccess(ctx context.Context, page int) (*SkillAccessListResponse, error) {
	var query url.Values
	if page > 1 {
		query = url.Values{"page": []string{strconv.Itoa(page)}}
	}

	var resp SkillAccessListResponse
	if err := c.do(ctx, http.MethodGet, "skills/list", query, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ExportSkills returns full skill payloads including content.
func (c *Client) ExportSkills(ctx context.Context) ([]SkillModel, error) {
	var resp []SkillModel
	if err := c.do(ctx, http.MethodGet, "skills/export", nil, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}
