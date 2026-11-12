package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func TestPromptResponseToModel_NewFields(t *testing.T) {
	ctx := context.Background()
	active := false
	resp := &client.PromptModel{
		ID:       "p1",
		Command:  "/greet",
		Name:     "Greet",
		Content:  "Hello!",
		IsActive: &active,
		Tags:     []string{"util", "test"},
	}

	state, diags := promptResponseToModel(ctx, nil, resp)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if state.IsActive.IsNull() || state.IsActive.ValueBool() != false {
		t.Fatalf("expected is_active=false, got %v", state.IsActive)
	}

	var tags []string
	if err := state.Tags.ElementsAs(ctx, &tags, false); err != nil {
		t.Fatalf("ElementsAs tags: %v", err)
	}
	if len(tags) != 2 || tags[0] != "util" || tags[1] != "test" {
		t.Fatalf("unexpected tags: %v", tags)
	}

	if !state.DataJSON.IsNull() {
		t.Fatalf("expected data_json to be null when Data is nil, got %v", state.DataJSON)
	}
	if !state.MetaJSON.IsNull() {
		t.Fatalf("expected meta_json to be null when Meta is nil, got %v", state.MetaJSON)
	}
}

func TestPromptResponseToModel_NilIsActive(t *testing.T) {
	ctx := context.Background()
	resp := &client.PromptModel{
		ID:       "p2",
		Command:  "/test",
		Name:     "Test",
		Content:  "content",
		IsActive: nil,
		Tags:     nil,
	}

	state, diags := promptResponseToModel(ctx, nil, resp)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	// nil IsActive should default to true
	if state.IsActive.IsNull() || !state.IsActive.ValueBool() {
		t.Fatalf("expected is_active=true (default) when nil, got %v", state.IsActive)
	}

	// nil Tags should produce null list
	if !state.Tags.IsNull() {
		t.Fatalf("expected nil tags to produce null list, got %v", state.Tags)
	}
}

// promptTogglePartialFailureBackend answers create/update with a prompt that has
// a group grant (so flattening it needs GET /groups/), and fails both the
// toggle and the group lookup.
func promptTogglePartialFailureBackend(t *testing.T) *client.Client {
	t.Helper()
	const prompt = `{"id":"p1","command":"/greet","name":"Greet","content":"hi","is_active":true,"user_id":"u1","created_at":1,"updated_at":2,
		"access_grants":[{"principal_type":"group","principal_id":"g1","permission":"read"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && (r.URL.Path == "/api/v1/prompts/create" || r.URL.Path == "/api/v1/prompts/id/p1/update"):
			_, _ = w.Write([]byte(prompt))
		case r.URL.Path == "/api/v1/prompts/id/p1/toggle":
			http.Error(w, `{"detail":"toggle failed"}`, http.StatusInternalServerError)
		case r.URL.Path == "/api/v1/groups/":
			http.Error(w, `{"detail":"groups failed"}`, http.StatusInternalServerError)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(srv.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func promptTestSchemaAndPlan(t *testing.T, id types.String) (tfsdk.Plan, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	var sresp resource.SchemaResponse
	NewPromptResource().Schema(ctx, resource.SchemaRequest{}, &sresp)
	nullList := types.ListNull(types.StringType)
	model := promptResourceModel{
		ID:          id,
		Command:     types.StringValue("/greet"),
		Name:        types.StringValue("Greet"),
		Content:     types.StringValue("hi"),
		IsActive:    types.BoolValue(false),
		Tags:        nullList,
		DataJSON:    types.StringNull(),
		MetaJSON:    types.StringNull(),
		ReadGroups:  nullList,
		WriteGroups: nullList,
		CreatedAt:   types.StringUnknown(),
		UpdatedAt:   types.StringUnknown(),
		UserID:      types.StringUnknown(),
	}
	empty := tftypes.NewValue(sresp.Schema.Type().TerraformType(ctx), nil)
	plan := tfsdk.Plan{Schema: sresp.Schema, Raw: empty}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatalf("encode plan: %v", diags)
	}
	state := tfsdk.State{Schema: sresp.Schema, Raw: plan.Raw}
	return plan, state
}

func requireDiagContaining(t *testing.T, summaries []string, want string) {
	t.Helper()
	for _, s := range summaries {
		if strings.Contains(s, want) {
			return
		}
	}
	t.Errorf("expected a diagnostic containing %q, got %v", want, summaries)
}

// TestPromptCreateReportsToggleFailureAndKeepsState: when flattening the
// created prompt fails (group lookup) the toggle error must still be reported,
// and the prompt must still be saved to state instead of leaking.
func TestPromptCreateReportsToggleFailureAndKeepsState(t *testing.T) {
	ctx := context.Background()
	plan, state := promptTestSchemaAndPlan(t, types.StringUnknown())
	empty := tfsdk.State{Schema: state.Schema, Raw: tftypes.NewValue(state.Schema.Type().TerraformType(ctx), nil)}

	r := &promptResource{client: promptTogglePartialFailureBackend(t)}
	resp := resource.CreateResponse{State: empty}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)

	var summaries []string
	for _, d := range resp.Diagnostics {
		summaries = append(summaries, d.Summary())
	}
	requireDiagContaining(t, summaries, "Set prompt is_active failed")
	requireDiagContaining(t, summaries, "Unable to list groups")

	if resp.State.Raw.IsNull() {
		t.Fatal("the created prompt must be saved to state even though diagnostics carry errors")
	}
	var saved promptResourceModel
	if diags := resp.State.Get(ctx, &saved); diags.HasError() {
		t.Fatalf("read state: %v", diags)
	}
	if saved.ID.ValueString() != "p1" {
		t.Errorf("state id = %q, want p1", saved.ID.ValueString())
	}
}

// TestPromptUpdateReportsToggleFailureWhenFlatteningFails: the toggle error
// must not be swallowed by the early return on a flattening error.
func TestPromptUpdateReportsToggleFailureWhenFlatteningFails(t *testing.T) {
	ctx := context.Background()
	plan, state := promptTestSchemaAndPlan(t, types.StringValue("p1"))

	r := &promptResource{client: promptTogglePartialFailureBackend(t)}
	resp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &resp)

	var summaries []string
	for _, d := range resp.Diagnostics {
		summaries = append(summaries, d.Summary())
	}
	requireDiagContaining(t, summaries, "Set prompt is_active failed")
	requireDiagContaining(t, summaries, "Unable to list groups")
}
