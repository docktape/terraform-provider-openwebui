package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func terminalHCL(attrs string) string {
	return `
resource "openwebui_terminal_servers_config" "test" {
` + attrs + `
}
`
}

func TestTerminalServersConfigPlanShapes(t *testing.T) {
	planOnlyValid(t, terminalHCL(`  connections = []`))
	planOnlyValid(t, terminalHCL(`  connections = [{ url = "http://t:8000" }]`))
	planOnlyValid(t, terminalHCL(`  connections = [{
    id          = "a"
    name        = "A"
    url         = "http://t:8000"
    enabled     = false
    path        = "/spec.json"
    key         = "k"
    auth_type   = "session"
    server_type = "orchestrator"
    policy_id   = "p"
    config_json = jsonencode({ access_grants = [] })
  }]`))
	planOnlyRejected(t, terminalHCL(`  connections = [{ url = "" }]`), `(?s)url.*(at least 1|length)`)
	planOnlyRejected(t, terminalHCL(`  connections = [{ url = "http://t", auth_type = "basic" }]`), `(?s)auth_type.*(bearer|system_oauth)`)
	for _, bad := range []string{``, `null`, `[]`, `{`} {
		planOnlyRejected(t, terminalHCL(`  connections = [{ url = "http://t", config_json = "`+bad+`" }]`), `(?s)config_json.*JSON\s+object`)
	}
	planOnlyRejected(t, terminalHCL(``), `(?s)connections.*(required|Missing)`)
}

func TestTerminalServersConfigSchemaShape(t *testing.T) {
	var resp resource.SchemaResponse
	NewTerminalServersConfigResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if len(resp.Schema.Attributes) != 2 {
		t.Fatalf("expected id and connections, got %d attributes", len(resp.Schema.Attributes))
	}
	for name, a := range resp.Schema.Attributes {
		if a.GetDescription() == "" || a.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q needs both Description and MarkdownDescription", name)
		}
	}
	list, ok := resp.Schema.Attributes["connections"].(schema.ListNestedAttribute)
	if !ok || !list.IsRequired() {
		t.Fatalf("connections must be a Required ListNestedAttribute")
	}
	if len(list.NestedObject.Attributes) != 10 {
		t.Errorf("expected 10 connection attributes, got %d", len(list.NestedObject.Attributes))
	}
	for name, a := range list.NestedObject.Attributes {
		if a.GetDescription() == "" || a.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q needs both Description and MarkdownDescription", name)
		}
		if a.IsSensitive() != (name == "key") {
			t.Errorf("attribute %q: unexpected Sensitive=%v", name, a.IsSensitive())
		}
	}
}

func TestFlattenTerminalServerConnections(t *testing.T) {
	// Empty and nil server lists are state [], never null.
	if got := flattenTerminalServerConnections(nil); got == nil || len(got) != 0 {
		t.Errorf("nil list must flatten to an empty non-nil slice, got %#v", got)
	}

	s := func(v string) *string { return &v }
	on := true
	got := flattenTerminalServerConnections([]client.TerminalServerConnection{
		{ID: s("a"), Name: s("A"), Enabled: &on, URL: "http://a", Path: s("/p"), Key: s("k"), AuthType: s("none"),
			Config: json.RawMessage(`{"x": 1.0}`), ServerType: s("terminal"), PolicyID: s("")},
		{URL: "http://b"},
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	a, b := got[0], got[1]
	if a.ID.ValueString() != "a" || !a.Enabled.ValueBool() || a.Key.ValueString() != "k" || a.ConfigJSON.ValueString() != `{"x":1.0}` ||
		a.ServerType.ValueString() != "terminal" || a.PolicyID.IsNull() || a.PolicyID.ValueString() != "" {
		t.Errorf("unexpected first entry %+v", a)
	}
	// Null strings with a server default map to "", nullable ones to null,
	// and a stored null enabled is disabled.
	for field, v := range map[string]types.String{"id": b.ID, "name": b.Name, "path": b.Path, "key": b.Key, "auth_type": b.AuthType} {
		if v.IsNull() || v.IsUnknown() {
			t.Errorf("%s must flatten to a known empty string, got %s", field, v)
		}
	}
	if b.ID.ValueString() != "" || b.Name.ValueString() != "" || b.Path.ValueString() != "" || b.Key.ValueString() != "" ||
		b.AuthType.ValueString() != "" || b.Enabled.ValueBool() || !b.ConfigJSON.IsNull() || !b.ServerType.IsNull() || !b.PolicyID.IsNull() {
		t.Errorf("unexpected second entry %+v", b)
	}
}

func TestPreserveTerminalConnectionJSON(t *testing.T) {
	preferred := []terminalServerConnectionModel{{ConfigJSON: types.StringValue(`{ "b": 1,  "a": 2 }`)}, {ConfigJSON: types.StringValue(`{"x":1}`)}}
	fresh := []terminalServerConnectionModel{{ConfigJSON: types.StringValue(`{"a":2,"b":1}`)}, {ConfigJSON: types.StringValue(`{"x":2}`)}, {ConfigJSON: types.StringNull()}}
	got := preserveTerminalConnectionJSON(preferred, fresh)
	if got[0].ConfigJSON.ValueString() != `{ "b": 1,  "a": 2 }` {
		t.Errorf("equivalent JSON must keep the user's text, got %s", got[0].ConfigJSON)
	}
	if got[1].ConfigJSON.ValueString() != `{"x":2}` {
		t.Errorf("changed JSON must take the server value, got %s", got[1].ConfigJSON)
	}
	if !got[2].ConfigJSON.IsNull() {
		t.Errorf("entry beyond the preferred list must stay as read")
	}
}

// fakeTerminalServers is a stateful stand-in for GET/POST
// /api/v1/configs/terminal_servers that echoes what it stored, like Open WebUI.
type fakeTerminalServers struct {
	mu    sync.Mutex
	store []any
	posts []map[string]any
}

func (f *fakeTerminalServers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path != "/api/v1/configs/terminal_servers" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			Connections []any `json:"TERMINAL_SERVER_CONNECTIONS"`
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&body); err != nil || body.Connections == nil {
			http.Error(w, `{"detail":"invalid"}`, http.StatusUnprocessableEntity)
			return
		}
		for i, c := range body.Connections {
			body.Connections[i] = pythonNumbers(c)
		}
		f.store = body.Connections
		for _, c := range body.Connections {
			if m, ok := c.(map[string]any); ok {
				f.posts = append(f.posts, m)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"TERMINAL_SERVER_CONNECTIONS": f.store})
}

// pythonNumbers mimics how Open WebUI re-serialises JSON numbers: integers
// keep full precision (Python int), anything with a fraction or exponent
// becomes a float (so `1.50` comes back as `1.5`).
func pythonNumbers(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			x[k] = pythonNumbers(e)
		}
	case []any:
		for i, e := range x {
			x[i] = pythonNumbers(e)
		}
	case json.Number:
		if strings.ContainsAny(string(x), ".eE") {
			f, _ := x.Float64()
			return f
		}
	}
	return v
}

func TestTerminalServersConfigAgainstFakeServer(t *testing.T) {
	fake := &fakeTerminalServers{}
	server := httptest.NewServer(fake)
	defer server.Close()
	t.Setenv("OPENWEBUI_ENDPOINT", server.URL)
	t.Setenv("OPENWEBUI_TOKEN", "test-token")

	const name = "openwebui_terminal_servers_config.test"
	empty := terminalHCL(`  connections = []`)
	minimal := terminalHCL(`  connections = [{ url = "http://t1:8000" }]`)
	full := terminalHCL(`  connections = [
    {
      id          = "b"
      name        = "Second"
      url         = "http://t2:8000"
      enabled     = false
      key         = "secret"
      auth_type   = "none"
      server_type = "orchestrator"
      policy_id   = "pol"
      config_json = "{ \"z\": 1,  \"access_grants\": [] }"
    },
    {
      id   = ""
      name = ""
      url  = "http://t3:8000"
      path = "/custom.json"
    },
  ]`)
	numbers := terminalHCL(`  connections = [{
    url         = "http://t4:8000"
    config_json = "{\"big\": 9007199254740993, \"speed\": 1.50, \"nested\": [1e2]}"
  }]`)
	reordered := terminalHCL(`  connections = [
    { id = "c", url = "http://t3:8000" },
    { id = "b", url = "http://t2:8000" },
  ]`)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config: empty,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "id", "terminal_servers"),
					tfresource.TestCheckResourceAttr(name, "connections.#", "0"),
				),
			},
			{Config: empty, PlanOnly: true},
			{
				Config: minimal,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "connections.#", "1"),
					tfresource.TestCheckResourceAttr(name, "connections.0.id", ""),
					tfresource.TestCheckResourceAttr(name, "connections.0.enabled", "true"),
					tfresource.TestCheckResourceAttr(name, "connections.0.path", "/openapi.json"),
					tfresource.TestCheckResourceAttr(name, "connections.0.auth_type", "bearer"),
					tfresource.TestCheckResourceAttr(name, "connections.0.key", ""),
					tfresource.TestCheckNoResourceAttr(name, "connections.0.config_json"),
					tfresource.TestCheckNoResourceAttr(name, "connections.0.server_type"),
					tfresource.TestCheckNoResourceAttr(name, "connections.0.policy_id"),
				),
			},
			{Config: minimal, PlanOnly: true},
			{
				Config: full,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "connections.#", "2"),
					tfresource.TestCheckResourceAttr(name, "connections.0.id", "b"),
					tfresource.TestCheckResourceAttr(name, "connections.0.enabled", "false"),
					tfresource.TestCheckResourceAttr(name, "connections.0.key", "secret"),
					tfresource.TestCheckResourceAttr(name, "connections.0.server_type", "orchestrator"),
					tfresource.TestCheckResourceAttr(name, "connections.0.config_json", `{ "z": 1,  "access_grants": [] }`),
					tfresource.TestCheckResourceAttr(name, "connections.1.path", "/custom.json"),
					tfresource.TestCheckResourceAttr(name, "connections.1.id", ""),
				),
			},
			{Config: full, PlanOnly: true},
			{
				Config: reordered,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "connections.0.id", "c"),
					tfresource.TestCheckResourceAttr(name, "connections.1.id", "b"),
					tfresource.TestCheckResourceAttr(name, "connections.1.key", ""),
				),
			},
			{Config: reordered, PlanOnly: true},
			{
				Config: numbers,
				Check: tfresource.TestCheckResourceAttr(name, "connections.0.config_json",
					`{"big": 9007199254740993, "speed": 1.50, "nested": [1e2]}`),
			},
			{Config: numbers, PlanOnly: true},
			{Config: empty, Check: tfresource.TestCheckResourceAttr(name, "connections.#", "0")},
			{Config: empty, PlanOnly: true},
		},
	})

	// The minimal entry reached the wire with every default made explicit.
	var sawMinimal bool
	for _, p := range fake.posts {
		if p["url"] == "http://t1:8000" {
			sawMinimal = true
			if p["path"] != "/openapi.json" || p["auth_type"] != "bearer" || p["enabled"] != true || p["id"] != "" || p["config"] != nil || p["server_type"] != nil {
				t.Errorf("unexpected wire form of minimal entry: %v", p)
			}
		}
	}
	var sawBig bool
	for _, p := range fake.posts {
		if p["url"] == "http://t4:8000" {
			sawBig = true
			config, _ := p["config"].(map[string]any)
			if config["big"] != json.Number("9007199254740993") {
				t.Errorf("large integer lost precision on the wire: %v", p)
			}
		}
	}
	if !sawBig {
		t.Errorf("number entry never posted")
	}
	if !sawMinimal {
		t.Errorf("minimal entry never posted")
	}
}
