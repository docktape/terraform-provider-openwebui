package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The error steps come first because the post-test destroy re-reads the last
// configuration, which must therefore be the successful one.
// The verify data source is unit-tested against a fake endpoint only: no
// reachable terminal server exists for an acceptance test.
func TestTerminalServerVerifyDataSource(t *testing.T) {
	var (
		mu   sync.Mutex
		last map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/configs/terminal_servers/verify" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		last = body
		mu.Unlock()
		if body["url"] == "http://down" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":"Failed to connect to the terminal server"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status": true, "type": "orchestrator"}`))
	}))
	defer server.Close()
	t.Setenv("OPENWEBUI_ENDPOINT", server.URL)
	t.Setenv("OPENWEBUI_TOKEN", "test-token")

	const name = "data.openwebui_terminal_server_verify.test"
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config:      `data "openwebui_terminal_server_verify" "test" { url = "http://down" }`,
				ExpectError: regexp.MustCompile(`Verify terminal server failed`),
			},
			{
				Config: `data "openwebui_terminal_server_verify" "test" {
  url       = "http://x"
  auth_type = "basic"
}`,
				ExpectError: regexp.MustCompile(`(?s)auth_type.*(bearer|system_oauth)`),
			},
			{
				Config: `data "openwebui_terminal_server_verify" "test" {
  url = "http://term:8000"
  key = "k"
}`,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(name, "verified", "true"),
					tfresource.TestCheckResourceAttr(name, "server_type", "orchestrator"),
					func(_ *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if last["auth_type"] != "bearer" || last["key"] != "k" || last["url"] != "http://term:8000" {
							t.Errorf("unexpected verify request %v", last)
						}
						return nil
					},
				),
			},
		},
	})
}
