package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestKnowledgeResource_UpgradeStateDropsRemovedAttributes feeds the state
// shape provider 1.2.0 wrote (including non-null data_json and meta_json, which
// are no longer in the schema) through UpgradeResourceState. The framework must
// drop the removed attributes without a diagnostic, so no state upgrader is
// needed.
func TestKnowledgeResource_UpgradeStateDropsRemovedAttributes(t *testing.T) {
	ctx := context.Background()
	server, err := testAccProtoV6ProviderFactories()["openwebui"]()
	if err != nil {
		t.Fatalf("provider server: %v", err)
	}

	raw := `{
  "id": "k1",
  "name": "kb",
  "description": "d",
  "data_json": "{\"a\":1}",
  "meta_json": "{\"source\":\"external\"}",
  "read_groups": [],
  "write_groups": [],
  "created_at": "2026-01-01",
  "updated_at": "2026-01-02",
  "user_id": "u1"
}`
	resp, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "openwebui_knowledge",
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: []byte(raw)},
	})
	if err != nil {
		t.Fatalf("UpgradeResourceState: %v", err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("unexpected diagnostic: %s: %s", d.Summary, d.Detail)
	}
	if resp.UpgradedState == nil {
		t.Fatal("no upgraded state returned")
	}

	schemaResp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	typ := schemaResp.ResourceSchemas["openwebui_knowledge"].ValueType()
	val, err := resp.UpgradedState.Unmarshal(typ)
	if err != nil {
		t.Fatalf("unmarshal upgraded state: %v", err)
	}
	var attrs map[string]tftypes.Value
	if err := val.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}
	for _, removed := range []string{"data_json", "meta_json"} {
		if _, ok := attrs[removed]; ok {
			t.Errorf("%s still in upgraded state", removed)
		}
	}
	var name string
	if err := attrs["name"].As(&name); err != nil || name != "kb" {
		t.Errorf("name = %q (%v), want kb", name, err)
	}
}
