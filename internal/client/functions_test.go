package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newFunctionsTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestCreateFunction(t *testing.T) {
	var gotPath, gotMethod, gotAuth string
	var gotForm FunctionForm
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod, gotAuth = r.URL.Path, r.Method, r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotForm)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"my_filter","user_id":"u1","type":"filter","name":"My Filter","is_active":false,"is_global":false,"created_at":1,"updated_at":2}`))
	})

	desc := "demo"
	out, err := c.CreateFunction(context.Background(), FunctionForm{
		ID: "my_filter", Name: "My Filter", Content: "class Filter:\n    pass\n",
		Meta: FunctionMeta{Description: &desc},
	})
	if err != nil {
		t.Fatalf("CreateFunction: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/functions/create" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("missing auth header: %q", gotAuth)
	}
	if gotForm.ID != "my_filter" || gotForm.Meta.Description == nil || *gotForm.Meta.Description != "demo" {
		t.Fatalf("unexpected form: %+v", gotForm)
	}
	if out.ID != "my_filter" || out.Type != "filter" || out.IsActive {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestGetFunction(t *testing.T) {
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/functions/id/my_filter" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"my_filter","user_id":"u1","type":"filter","name":"My Filter","content":"class Filter:\n    pass\n","meta":{"description":"demo","manifest":{"title":"My Filter"}},"is_active":true,"is_global":false,"created_at":1,"updated_at":2}`))
	})

	out, err := c.GetFunction(context.Background(), "my_filter")
	if err != nil {
		t.Fatalf("GetFunction: %v", err)
	}
	if out.Content == "" || !out.IsActive || out.Meta.Manifest["title"] != "My Filter" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestGetFunctionNotFound(t *testing.T) {
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	if _, err := c.GetFunction(context.Background(), "missing"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteFunction(t *testing.T) {
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/functions/id/my_filter/delete" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`true`))
	})
	if err := c.DeleteFunction(context.Background(), "my_filter"); err != nil {
		t.Fatalf("DeleteFunction: %v", err)
	}
}

func TestUpdateFunction(t *testing.T) {
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/functions/id/my_filter/update" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"my_filter","user_id":"u1","type":"filter","name":"Renamed","content":"class Filter:\n    pass\n","is_active":true,"is_global":false,"created_at":1,"updated_at":3}`))
	})
	out, err := c.UpdateFunction(context.Background(), "my_filter", FunctionForm{ID: "my_filter", Name: "Renamed", Content: "class Filter:\n    pass\n"})
	if err != nil {
		t.Fatalf("UpdateFunction: %v", err)
	}
	if out.Name != "Renamed" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestToggleFunction(t *testing.T) {
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/functions/id/my_filter/toggle" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"my_filter","user_id":"u1","type":"filter","is_active":true,"is_global":false,"created_at":1,"updated_at":4}`))
	})
	out, err := c.ToggleFunction(context.Background(), "my_filter")
	if err != nil {
		t.Fatalf("ToggleFunction: %v", err)
	}
	if !out.IsActive {
		t.Fatalf("expected is_active true, got %+v", out)
	}
}

func TestToggleFunctionGlobal(t *testing.T) {
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/functions/id/my_filter/toggle/global" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"my_filter","user_id":"u1","type":"filter","is_active":true,"is_global":true,"created_at":1,"updated_at":5}`))
	})
	out, err := c.ToggleFunctionGlobal(context.Background(), "my_filter")
	if err != nil {
		t.Fatalf("ToggleFunctionGlobal: %v", err)
	}
	if !out.IsGlobal {
		t.Fatalf("expected is_global true, got %+v", out)
	}
}

func TestGetFunctionValves(t *testing.T) {
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/functions/id/my_filter/valves" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"priority":5}`))
	})
	out, err := c.GetFunctionValves(context.Background(), "my_filter")
	if err != nil {
		t.Fatalf("GetFunctionValves: %v", err)
	}
	if out["priority"].(float64) != 5 {
		t.Fatalf("unexpected valves: %+v", out)
	}
}

func TestGetFunctionValvesSpec(t *testing.T) {
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/functions/id/my_filter/valves/spec" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"type":"object","properties":{"priority":{"type":"integer"}}}`))
	})
	out, err := c.GetFunctionValvesSpec(context.Background(), "my_filter")
	if err != nil {
		t.Fatalf("GetFunctionValvesSpec: %v", err)
	}
	if out["type"] != "object" {
		t.Fatalf("unexpected spec: %+v", out)
	}
}

func TestUpdateFunctionValves(t *testing.T) {
	var gotBody map[string]any
	c := newFunctionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/functions/id/my_filter/valves/update" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		_, _ = w.Write([]byte(`{"priority":9}`))
	})
	out, err := c.UpdateFunctionValves(context.Background(), "my_filter", map[string]any{"priority": 9})
	if err != nil {
		t.Fatalf("UpdateFunctionValves: %v", err)
	}
	if gotBody["priority"].(float64) != 9 || out["priority"].(float64) != 9 {
		t.Fatalf("unexpected valves: sent %+v got %+v", gotBody, out)
	}
}
