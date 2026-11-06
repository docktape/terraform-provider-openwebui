package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func skillStrList(t *testing.T, values ...string) types.List {
	t.Helper()
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

func skillBasePlan() skillResourceModel {
	return skillResourceModel{
		SkillID:     types.StringValue("playbook"),
		Name:        types.StringValue("Playbook"),
		Content:     types.StringValue("do the thing"),
		Description: types.StringNull(),
		IsActive:    types.BoolValue(true),
		Tags:        types.ListNull(types.StringType),
		ReadGroups:  types.ListNull(types.StringType),
		WriteGroups: types.ListNull(types.StringType),
	}
}

func skillListElems(t *testing.T, l types.List) []string {
	t.Helper()
	var out []string
	if diags := l.ElementsAs(context.Background(), &out, false); diags.HasError() {
		t.Fatalf("ElementsAs: %s", diags)
	}
	return out
}

func TestSkillFormFromPlan_TagsNullUnknownEmptyPopulated(t *testing.T) {
	ctx := context.Background()
	cases := map[string]struct {
		tags types.List
		want []string
	}{
		"null":      {types.ListNull(types.StringType), []string{}},
		"unknown":   {types.ListUnknown(types.StringType), []string{}},
		"empty":     {skillStrList(t), []string{}},
		"populated": {skillStrList(t, "support", "playbook"), []string{"support", "playbook"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			plan := skillBasePlan()
			plan.Tags = tc.tags
			form, diags := skillFormFromPlan(ctx, nil, plan)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}
			// Tags must never be nil: nil would marshal as null; the wire value is always a list.
			if form.Meta.Tags == nil {
				t.Fatalf("expected non-nil tags slice")
			}
			if len(form.Meta.Tags) != len(tc.want) {
				t.Fatalf("expected tags %v, got %v", tc.want, form.Meta.Tags)
			}
			for i := range tc.want {
				if form.Meta.Tags[i] != tc.want[i] {
					t.Fatalf("expected tags %v, got %v", tc.want, form.Meta.Tags)
				}
			}
		})
	}
}

func TestSkillFormFromPlan_IsActiveDefaultsTrueWhenUnknownOrNull(t *testing.T) {
	ctx := context.Background()
	for name, v := range map[string]types.Bool{
		"unknown": types.BoolUnknown(),
		"null":    types.BoolNull(),
	} {
		t.Run(name, func(t *testing.T) {
			plan := skillBasePlan()
			plan.IsActive = v
			form, diags := skillFormFromPlan(ctx, nil, plan)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}
			if !form.IsActive {
				t.Fatalf("expected is_active to default to true (server default) when %s", name)
			}
		})
	}

	plan := skillBasePlan()
	plan.IsActive = types.BoolValue(false)
	form, _ := skillFormFromPlan(ctx, nil, plan)
	if form.IsActive {
		t.Fatalf("explicit is_active=false must be preserved")
	}
}

func TestSkillFormFromPlan_Description(t *testing.T) {
	ctx := context.Background()

	plan := skillBasePlan()
	form, _ := skillFormFromPlan(ctx, nil, plan)
	if form.Description != nil {
		t.Fatalf("null description must map to nil, got %q", *form.Description)
	}

	plan.Description = types.StringUnknown()
	form, _ = skillFormFromPlan(ctx, nil, plan)
	if form.Description != nil {
		t.Fatalf("unknown description must map to nil")
	}

	plan.Description = types.StringValue("")
	form, _ = skillFormFromPlan(ctx, nil, plan)
	if form.Description == nil || *form.Description != "" {
		t.Fatalf("empty-string description must be preserved, got %v", form.Description)
	}

	plan.Description = types.StringValue("d")
	form, _ = skillFormFromPlan(ctx, nil, plan)
	if form.Description == nil || *form.Description != "d" {
		t.Fatalf("expected description d, got %v", form.Description)
	}
	if form.ID != "playbook" || form.Name != "Playbook" || form.Content != "do the thing" {
		t.Fatalf("unexpected form: %+v", form)
	}
}

func TestSkillResponseToModel_TagsRoundTrip(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		in   []string
		want []string
	}{
		"nil from server null": {nil, []string{}},
		"empty":                {[]string{}, []string{}},
		"populated":            {[]string{"a", "b"}, []string{"a", "b"}},
	} {
		t.Run(name, func(t *testing.T) {
			access := &client.SkillAccessResponse{ID: "s1", Name: "S", IsActive: true, Meta: client.SkillMeta{Tags: tc.in}}
			state, diags := skillResponseToModel(ctx, nil, access, nil, types.StringValue("fallback"))
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}
			// The schema default for tags is [], so state must never be null.
			if state.Tags.IsNull() || state.Tags.IsUnknown() {
				t.Fatalf("tags must be a known, non-null list, got %v", state.Tags)
			}
			got := skillListElems(t, state.Tags)
			if len(got) != len(tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("expected %v, got %v", tc.want, got)
				}
			}
		})
	}
}

func TestSkillResponseToModel_FieldsAndContentPrecedence(t *testing.T) {
	ctx := context.Background()
	desc := "d"
	write := true
	access := &client.SkillAccessResponse{
		ID: "s1", UserID: "u1", Name: "S", Description: &desc, IsActive: false,
		CreatedAt: 11, UpdatedAt: 22, WriteAccess: &write,
	}

	// Content supplied by the API wins over the fallback.
	content := "from api"
	state, diags := skillResponseToModel(ctx, nil, access, &content, types.StringValue("fallback"))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if state.Content.ValueString() != "from api" {
		t.Fatalf("expected API content, got %v", state.Content)
	}
	if state.ID.ValueString() != "s1" || state.SkillID.ValueString() != "s1" || state.UserID.ValueString() != "u1" {
		t.Fatalf("unexpected ids: %+v", state)
	}
	if state.Description.ValueString() != "d" || state.IsActive.ValueBool() || state.CreatedAt.ValueInt64() != 11 || state.UpdatedAt.ValueInt64() != 22 {
		t.Fatalf("unexpected scalar fields: %+v", state)
	}
	if state.WriteAccess.IsNull() || !state.WriteAccess.ValueBool() {
		t.Fatalf("expected write_access=true, got %v", state.WriteAccess)
	}

	// Empty-string content from the API is a real value, not a miss.
	empty := ""
	state, _ = skillResponseToModel(ctx, nil, access, &empty, types.StringValue("fallback"))
	if state.Content.IsNull() || state.Content.ValueString() != "" {
		t.Fatalf("expected empty content preserved, got %v", state.Content)
	}

	// Without API content the prior value is kept; with neither it is null.
	state, _ = skillResponseToModel(ctx, nil, access, nil, types.StringValue("fallback"))
	if state.Content.ValueString() != "fallback" {
		t.Fatalf("expected fallback content, got %v", state.Content)
	}
	state, _ = skillResponseToModel(ctx, nil, access, nil, types.StringNull())
	if !state.Content.IsNull() {
		t.Fatalf("expected null content, got %v", state.Content)
	}

	// Null description and absent write_access.
	state, _ = skillResponseToModel(ctx, nil, &client.SkillAccessResponse{ID: "s2", Name: "S2"}, nil, types.StringNull())
	if !state.Description.IsNull() {
		t.Fatalf("expected null description, got %v", state.Description)
	}
	if !state.WriteAccess.IsNull() {
		t.Fatalf("expected null write_access, got %v", state.WriteAccess)
	}
}

func TestSkillResponseToModel_NilAccessIsError(t *testing.T) {
	_, diags := skillResponseToModel(context.Background(), nil, nil, nil, types.StringNull())
	if !diags.HasError() {
		t.Fatalf("expected an error diagnostic for a nil response")
	}
}

// TestSkill_AccessGroupsRoundTrip proves the shared group helpers are wired
// correctly for skills: names resolve to IDs on the way out and back to names
// on the way in.
func TestSkill_AccessGroupsRoundTrip(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/groups/" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`[{"id":"g1","user_id":"u","name":"Support","description":"","created_at":1,"updated_at":1,"user_ids":[]},{"id":"g2","user_id":"u","name":"Editors","description":"","created_at":1,"updated_at":1,"user_ids":[]}]`))
	}))
	t.Cleanup(server.Close)
	apiClient, err := client.NewClient(server.URL, "t", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	plan := skillBasePlan()
	// Open WebUI v0.11.0 does not treat write as read, so every write group must
	// also be listed under read_groups (enforced by the shared access-group ValidateConfig/ModifyPlan).
	plan.ReadGroups = skillStrList(t, "Support", "Editors")
	plan.WriteGroups = skillStrList(t, "Editors")
	form, diags := skillFormFromPlan(ctx, apiClient, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	read, _ := form.AccessControl["read"].(map[string]any)
	write, _ := form.AccessControl["write"].(map[string]any)
	if ids, _ := read["group_ids"].([]string); len(ids) != 2 || ids[0] != "g1" || ids[1] != "g2" {
		t.Fatalf("expected read.group_ids=[g1 g2], got %+v", read)
	}
	if ids, _ := write["group_ids"].([]string); len(ids) != 1 || ids[0] != "g2" {
		t.Fatalf("expected write.group_ids=[g2], got %+v", write)
	}

	access := &client.SkillAccessResponse{
		ID: "s1", Name: "S",
		AccessControl: map[string]any{
			// What the real server returns for the plan above: read=[g1 g2], write=[g2].
			"read":  map[string]any{"group_ids": []string{"g1", "g2"}, "user_ids": []string{}},
			"write": map[string]any{"group_ids": []string{"g2"}, "user_ids": []string{}},
		},
	}
	state, diags := skillResponseToModel(ctx, apiClient, access, nil, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if got := skillListElems(t, state.ReadGroups); len(got) != 2 || got[0] != "Support" || got[1] != "Editors" {
		t.Fatalf("expected read_groups=[Support Editors], got %v", got)
	}
	if got := skillListElems(t, state.WriteGroups); len(got) != 1 || got[0] != "Editors" {
		t.Fatalf("expected write_groups=[Editors], got %v", got)
	}
}

func TestSkillContentFor(t *testing.T) {
	ctx := context.Background()

	// Content already on the GET response: no export call is made.
	inline := "inline"
	got, diags := skillContentFor(ctx, nil, &client.SkillAccessResponse{ID: "s1", Content: &inline})
	if diags.HasError() || got == nil || *got != "inline" {
		t.Fatalf("expected inline content without calling the API, got %v diags=%s", got, diags)
	}

	// Content absent: falls back to the export listing.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/skills/export" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`[{"id":"other","user_id":"u","name":"O","content":"nope","meta":{"tags":[]},"is_active":true,"access_grants":[],"updated_at":1,"created_at":1},{"id":"s1","user_id":"u","name":"S","content":"exported","meta":{"tags":[]},"is_active":true,"access_grants":[],"updated_at":1,"created_at":1}]`))
	}))
	t.Cleanup(server.Close)
	apiClient, err := client.NewClient(server.URL, "t", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got, diags = skillContentFor(ctx, apiClient, &client.SkillAccessResponse{ID: "s1"})
	if diags.HasError() || got == nil || *got != "exported" {
		t.Fatalf("expected exported content, got %v diags=%s", got, diags)
	}

	// Skill absent from the export: nil, so callers keep their prior value.
	got, diags = skillContentFor(ctx, apiClient, &client.SkillAccessResponse{ID: "missing"})
	if diags.HasError() || got != nil {
		t.Fatalf("expected nil content for an id missing from the export, got %v diags=%s", got, diags)
	}

	// Export forbidden (non-admin without workspace.skills_export): warning only.
	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Unauthorized"}`))
	}))
	t.Cleanup(denied.Close)
	deniedClient, _ := client.NewClient(denied.URL, "t", false)
	got, diags = skillContentFor(ctx, deniedClient, &client.SkillAccessResponse{ID: "s1"})
	if diags.HasError() || got != nil || diags.WarningsCount() != 1 {
		t.Fatalf("expected one warning and nil content, got %v diags=%s", got, diags)
	}
}

func TestSkillIDValidator(t *testing.T) {
	v := skillIDValidator{}
	cases := map[string]bool{ // value -> valid
		"customer-support-playbook": true,
		"web_scraper.v2":            true,
		"abc123":                    true,
		"":                          false,
		"Has Upper":                 false,
		"UPPER":                     false,
		"has space":                 false,
		"tab\tchar":                 false,
		"a/b":                       false,
		".":                         false,
		"..":                        false,
		"a.b":                       true,
		".hidden":                   true,
	}
	for value, valid := range cases {
		t.Run(value, func(t *testing.T) {
			var resp validator.StringResponse
			v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(value)}, &resp)
			if valid && resp.Diagnostics.HasError() {
				t.Fatalf("expected %q to be accepted, got %s", value, resp.Diagnostics)
			}
			if !valid && !resp.Diagnostics.HasError() {
				t.Fatalf("expected %q to be rejected", value)
			}
		})
	}

	// Unknown and null values are not validated.
	for _, val := range []types.String{types.StringUnknown(), types.StringNull()} {
		var resp validator.StringResponse
		v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: val}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error for %v: %s", val, resp.Diagnostics)
		}
	}
}

func TestSkillResourceSchemaIsValid(t *testing.T) {
	r := NewSkillResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid schema: %s", diags)
	}
	for name, a := range resp.Schema.Attributes {
		if a.GetDescription() == "" || a.GetMarkdownDescription() == "" {
			t.Errorf("attribute %q must have Description and MarkdownDescription", name)
		}
	}
}
