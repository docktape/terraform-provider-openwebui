package client

import (
	"context"
	"net/http"
)

// GetVersion returns the Open WebUI server version (for example "0.12.0").
// The endpoint lives at /api/version, outside the /api/v1 prefix.
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	var out struct {
		Version string `json:"version"`
	}
	if err := c.doRaw(ctx, http.MethodGet, c.rootURL+"/api/version", nil, &out); err != nil {
		return "", err
	}
	return out.Version, nil
}
