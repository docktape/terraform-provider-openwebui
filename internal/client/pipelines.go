package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// AddPipelineForm represents the payload for registering a pipeline by URL.
type AddPipelineForm struct {
	URL    string `json:"url"`
	URLIdx int    `json:"urlIdx"`
}

// DeletePipelineForm represents the payload for deleting a pipeline.
type DeletePipelineForm struct {
	ID     string `json:"id"`
	URLIdx int    `json:"urlIdx"`
}

// AddPipeline registers a new pipeline by URL.
func (c *Client) AddPipeline(ctx context.Context, urlValue string, urlIdx int) (map[string]any, error) {
	var resp map[string]any
	form := AddPipelineForm{URL: urlValue, URLIdx: urlIdx}
	if err := c.do(ctx, http.MethodPost, "pipelines/add", nil, form, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// UploadPipeline uploads a pipeline bundle.
func (c *Client) UploadPipeline(ctx context.Context, filePath string, urlIdx int) (map[string]any, error) {
	var resp map[string]any
	fields := map[string]string{"urlIdx": fmt.Sprintf("%d", urlIdx)}
	files := map[string]string{"file": filePath}
	if err := c.doMultipart(ctx, http.MethodPost, "pipelines/upload", nil, fields, files, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// DeletePipeline removes a pipeline by ID.
func (c *Client) DeletePipeline(ctx context.Context, id string, urlIdx int) error {
	form := DeletePipelineForm{ID: id, URLIdx: urlIdx}
	return c.do(ctx, http.MethodDelete, "pipelines/delete", nil, form, nil)
}

// GetPipelines retrieves pipelines for a given URL index.
func (c *Client) GetPipelines(ctx context.Context, urlIdx *int) ([]map[string]any, error) {
	query := url.Values{}
	if urlIdx != nil {
		query.Set("urlIdx", fmt.Sprintf("%d", *urlIdx))
	}

	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "pipelines/", query, nil, &raw); err != nil {
		return nil, err
	}

	return decodePipelineCollection(raw)
}

// GetPipelineValves retrieves valve settings for a pipeline.
func (c *Client) GetPipelineValves(ctx context.Context, pipelineID string, urlIdx int) (map[string]any, error) {
	var resp map[string]any
	query := url.Values{"urlIdx": []string{fmt.Sprintf("%d", urlIdx)}}
	path := fmt.Sprintf("pipelines/%s/valves", url.PathEscape(pipelineID))
	if err := c.do(ctx, http.MethodGet, path, query, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// GetPipelineValvesSpec retrieves valve specs for a pipeline.
func (c *Client) GetPipelineValvesSpec(ctx context.Context, pipelineID string, urlIdx int) (map[string]any, error) {
	var resp map[string]any
	query := url.Values{"urlIdx": []string{fmt.Sprintf("%d", urlIdx)}}
	path := fmt.Sprintf("pipelines/%s/valves/spec", url.PathEscape(pipelineID))
	if err := c.do(ctx, http.MethodGet, path, query, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// UpdatePipelineValves updates valve settings for a pipeline.
func (c *Client) UpdatePipelineValves(ctx context.Context, pipelineID string, urlIdx int, valves map[string]any) (map[string]any, error) {
	var resp map[string]any
	query := url.Values{"urlIdx": []string{fmt.Sprintf("%d", urlIdx)}}
	path := fmt.Sprintf("pipelines/%s/valves/update", url.PathEscape(pipelineID))
	if err := c.do(ctx, http.MethodPost, path, query, valves, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func decodePipelineCollection(raw json.RawMessage) ([]map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}

	switch v := decoded.(type) {
	case []any:
		return decodePipelineSlice(v), nil
	case map[string]any:
		if list := pipelineListFromMap(v); list != nil {
			return list, nil
		}
		return []map[string]any{v}, nil
	default:
		return nil, nil
	}
}

func pipelineListFromMap(values map[string]any) []map[string]any {
	for _, key := range []string{"pipelines", "items", "data"} {
		if raw, ok := values[key]; ok {
			if list, ok := raw.([]any); ok {
				return decodePipelineSlice(list)
			}
		}
	}

	return nil
}

func decodePipelineSlice(values []any) []map[string]any {
	var result []map[string]any
	for _, item := range values {
		if item == nil {
			continue
		}
		if entry, ok := item.(map[string]any); ok {
			result = append(result, entry)
		}
	}

	return result
}
