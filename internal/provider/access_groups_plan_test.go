package provider

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

// accessGroupResources lists every resource that exposes read_groups and
// write_groups through buildAccessControl.
var accessGroupResources = map[string]func() resource.Resource{
	"tool":      NewToolResource,
	"knowledge": NewKnowledgeResource,
	"model":     NewModelResource,
	"prompt":    NewPromptResource,
	"skill":     NewSkillResource,
}

func agList(values ...string) types.List {
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

var (
	agNull    = types.ListNull(types.StringType)
	agUnknown = types.ListUnknown(types.StringType)
)

func agListWithUnknown(values ...string) types.List {
	elems := make([]attr.Value, 0, len(values)+1)
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	elems = append(elems, types.StringUnknown())
	return types.ListValueMust(types.StringType, elems)
}

func agSchema(t *testing.T, newResource func() resource.Resource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	newResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %s", resp.Diagnostics)
	}
	return resp.Schema
}

// agObject builds a value of the resource's schema type with every attribute
// null except read_groups and write_groups.
func agObject(t *testing.T, s schema.Schema, read, write types.List) tftypes.Value {
	t.Helper()
	ctx := context.Background()
	objType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("schema type is not an object")
	}
	readVal, err := read.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	writeVal, err := write.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(typ, nil)
	}
	vals["read_groups"] = readVal
	vals["write_groups"] = writeVal
	return tftypes.NewValue(objType, vals)
}

func TestAccessGroupKey(t *testing.T) {
	if got := accessGroupKey("  Support Team "); got != "support team" {
		t.Fatalf("got %q", got)
	}
}

func TestKnownStringElements(t *testing.T) {
	cases := map[string]struct {
		in   types.List
		want []string
		ok   bool
	}{
		"null":            {agNull, nil, false},
		"unknown":         {agUnknown, nil, false},
		"unknown element": {agListWithUnknown("A"), nil, false},
		"null element":    {types.ListValueMust(types.StringType, []attr.Value{types.StringNull()}), nil, false},
		"empty":           {agList(), []string{}, true},
		"values":          {agList("A", "B"), []string{"A", "B"}, true},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := knownStringElements(c.in)
			if ok != c.ok || (ok && !reflect.DeepEqual(got, c.want)) {
				t.Fatalf("got %v %v, want %v %v", got, ok, c.want, c.ok)
			}
		})
	}
}

func TestMissingWriteGroups(t *testing.T) {
	cases := map[string]struct {
		read, write, want []string
	}{
		"subset":           {[]string{"A", "B"}, []string{"B"}, nil},
		"equal":            {[]string{"A"}, []string{"A"}, nil},
		"case insensitive": {[]string{"support"}, []string{"SUPPORT"}, nil},
		"whitespace":       {[]string{"Support"}, []string{" Support "}, nil},
		"empty write":      {[]string{"A"}, nil, nil},
		"both empty":       {nil, nil, nil},
		"missing":          {[]string{"A"}, []string{"B"}, []string{"B"}},
		"read empty":       {nil, []string{"B", "C"}, []string{"B", "C"}},
		"partial":          {[]string{"A"}, []string{"A", "C", "B"}, []string{"C", "B"}},
		"dedupe":           {[]string{"A"}, []string{"B", "b", " B"}, []string{"B"}},
		"blank skipped":    {[]string{"A"}, []string{"", "  "}, nil},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := missingWriteGroups(c.read, c.write); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func agValidate(t *testing.T, newResource func() resource.Resource, read, write types.List) diag.Diagnostics {
	t.Helper()
	s := agSchema(t, newResource)
	r, ok := newResource().(resource.ResourceWithValidateConfig)
	if !ok {
		t.Fatalf("resource does not implement ValidateConfig")
	}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: s, Raw: agObject(t, s, read, write)},
	}, &resp)
	return resp.Diagnostics
}

func TestAccessGroupsValidateConfig(t *testing.T) {
	valid := map[string][2]types.List{
		"superset":              {agList("Support", "Editors"), agList("Editors")},
		"equal":                 {agList("Support"), agList("Support")},
		"case insensitive":      {agList("support"), agList("SUPPORT")},
		"whitespace":            {agList("Support"), agList(" Support ")},
		"empty write":           {agList("Support"), agList()},
		"both empty":            {agList(), agList()},
		"both null":             {agNull, agNull},
		"read null skipped":     {agNull, agList("Editors")},
		"write null skipped":    {agList("Support"), agNull},
		"read unknown skipped":  {agUnknown, agList("Editors")},
		"write unknown skipped": {agList("Support"), agUnknown},
		"unknown element read":  {agListWithUnknown("A"), agList("B")},
		"unknown element write": {agList("A"), agListWithUnknown("B")},
	}
	invalid := map[string][2]types.List{
		"write not in read": {agList("Support"), agList("Editors")},
		"read empty":        {agList(), agList("Editors")},
		"partially in read": {agList("Support"), agList("Support", "Editors")},
	}

	for resName, newResource := range accessGroupResources {
		for name, c := range valid {
			t.Run(resName+"/valid/"+name, func(t *testing.T) {
				if diags := agValidate(t, newResource, c[0], c[1]); diags.HasError() {
					t.Fatalf("expected no error, got %s", diags)
				}
			})
		}
		for name, c := range invalid {
			t.Run(resName+"/invalid/"+name, func(t *testing.T) {
				diags := agValidate(t, newResource, c[0], c[1])
				if !diags.HasError() {
					t.Fatalf("expected an error")
				}
				if diags.Errors()[0].Summary() != "write_groups must also be listed in read_groups" {
					t.Fatalf("unexpected summary: %s", diags)
				}
				if !strings.Contains(diags.Errors()[0].Detail(), `"Editors"`) || strings.Contains(diags.Errors()[0].Detail(), `"Support"`) {
					t.Fatalf("expected only the offending group in the message, got %s", diags)
				}
			})
		}
	}
}

type agPlanCase struct {
	configRead, configWrite   types.List
	plannedRead, plannedWrite types.List
	destroy                   bool

	wantRead  types.List
	wantError string // substring of the error detail; empty means no error
}

func agModifyPlan(t *testing.T, newResource func() resource.Resource, c agPlanCase) (types.List, diag.Diagnostics) {
	t.Helper()
	ctx := context.Background()
	s := agSchema(t, newResource)
	r, ok := newResource().(resource.ResourceWithModifyPlan)
	if !ok {
		t.Fatalf("resource does not implement ModifyPlan")
	}

	config := tfsdk.Config{Schema: s, Raw: agObject(t, s, c.configRead, c.configWrite)}
	planRaw := agObject(t, s, c.plannedRead, c.plannedWrite)
	if c.destroy {
		planRaw = tftypes.NewValue(s.Type().TerraformType(ctx), nil)
		config.Raw = tftypes.NewValue(s.Type().TerraformType(ctx), nil)
	}
	plan := tfsdk.Plan{Schema: s, Raw: planRaw}
	state := tfsdk.State{Schema: s, Raw: agObject(t, s, c.plannedRead, c.plannedWrite)}

	resp := resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: state}, &resp)
	if c.destroy || resp.Diagnostics.HasError() {
		return types.List{}, resp.Diagnostics
	}

	var read types.List
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, accessReadGroupsPath, &read)...)
	var write types.List
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, accessWriteGroupsPath, &write)...)
	if !write.Equal(c.plannedWrite) {
		t.Fatalf("planned write_groups changed: %s", write)
	}
	return read, resp.Diagnostics
}

func TestAccessGroupsModifyPlan(t *testing.T) {
	cases := map[string]agPlanCase{
		"read-only config keeps plan": {
			configRead: agList("A"), configWrite: agNull,
			plannedRead: agList("A"), plannedWrite: agList(),
			wantRead: agList("A"),
		},
		"read and write consistent": {
			configRead: agList("A", "B"), configWrite: agList("B"),
			plannedRead: agList("A", "B"), plannedWrite: agList("B"),
			wantRead: agList("A", "B"),
		},
		"case and space insensitive": {
			configRead: agList("a"), configWrite: agList(" A "),
			plannedRead: agList("a"), plannedWrite: agList(" A "),
			wantRead: agList("a"),
		},
		"read unset, write set: read extended": {
			configRead: agNull, configWrite: agList("C", "B"),
			plannedRead: agList("A", "B"), plannedWrite: agList("C", "B"),
			wantRead: agList("A", "B", "C"),
		},
		"read unset, write set, empty state read": {
			configRead: agNull, configWrite: agList("B", "b", ""),
			plannedRead: agList(), plannedWrite: agList("B", "b", ""),
			wantRead: agList("B"),
		},
		"create: read unknown stays unknown": {
			configRead: agNull, configWrite: agList("B"),
			plannedRead: agUnknown, plannedWrite: agList("B"),
			wantRead: agUnknown,
		},
		"create: both unset": {
			configRead: agNull, configWrite: agNull,
			plannedRead: agUnknown, plannedWrite: agUnknown,
			wantRead: agUnknown,
		},
		"update: both unset, consistent state": {
			configRead: agNull, configWrite: agNull,
			plannedRead: agList("A", "B"), plannedWrite: agList("B"),
			wantRead: agList("A", "B"),
		},
		"update: both unset, state write outside read": {
			configRead: agNull, configWrite: agNull,
			plannedRead: agList("A"), plannedWrite: agList("B"),
			wantRead: agList("A", "B"),
		},
		"write unknown skipped": {
			configRead: agList("A"), configWrite: agUnknown,
			plannedRead: agList("A"), plannedWrite: agUnknown,
			wantRead: agList("A"),
		},
		"write unknown element skipped": {
			configRead: agList("A"), configWrite: agListWithUnknown("B"),
			plannedRead: agList("A"), plannedWrite: agListWithUnknown("B"),
			wantRead: agList("A"),
		},
		"read unknown element skipped": {
			configRead: agListWithUnknown("A"), configWrite: agList("B"),
			plannedRead: agListWithUnknown("A"), plannedWrite: agList("B"),
			wantRead: agListWithUnknown("A"),
		},
		"read configured, state-derived write missing: error": {
			configRead: agList("A"), configWrite: agNull,
			plannedRead: agList("A"), plannedWrite: agList("A", "B"),
			wantError: `"B", kept from the current state`,
		},
		"read configured empty, state-derived write: error": {
			configRead: agList(), configWrite: agNull,
			plannedRead: agList(), plannedWrite: agList("B"),
			wantError: `"B", kept from the current state`,
		},
		"read and write configured, mismatch known only at plan: error": {
			configRead: agList("A"), configWrite: agList("B"),
			plannedRead: agList("A"), plannedWrite: agList("B"),
			wantError: `The write group(s) "B" are not in the configured read_groups`,
		},
		"destroy is skipped": {
			destroy: true,
		},
	}

	for resName, newResource := range accessGroupResources {
		for name, c := range cases {
			t.Run(resName+"/"+name, func(t *testing.T) {
				if c.destroy {
					c.configRead, c.configWrite, c.plannedRead, c.plannedWrite = agNull, agNull, agList("A"), agList("B")
				}
				read, diags := agModifyPlan(t, newResource, c)
				if c.wantError != "" {
					if !diags.HasError() {
						t.Fatalf("expected an error")
					}
					if !strings.Contains(diags.Errors()[0].Detail(), c.wantError) {
						t.Fatalf("expected %q in the error, got %s", c.wantError, diags)
					}
					return
				}
				if diags.HasError() {
					t.Fatalf("unexpected error: %s", diags)
				}
				if c.destroy {
					return
				}
				if !read.Equal(c.wantRead) {
					t.Fatalf("planned read_groups = %s, want %s", read, c.wantRead)
				}
			})
		}
	}
}

func TestAccessGroupsDescriptions(t *testing.T) {
	for resName, newResource := range accessGroupResources {
		s := agSchema(t, newResource)
		for _, name := range []string{"read_groups", "write_groups"} {
			a := s.Attributes[name]
			if a.GetDescription() == "" || a.GetDescription() != a.GetMarkdownDescription() {
				t.Errorf("%s.%s: Description and MarkdownDescription must be set and equal", resName, name)
			}
			if !strings.Contains(a.GetDescription(), "read group") {
				t.Errorf("%s.%s: description must explain that write groups are stored as read groups", resName, name)
			}
		}
	}
}

// Prompt and knowledge used to flatten empty grants to null, which made
// `read_groups = []` / `write_groups = []` fail with an inconsistent result.
func TestAccessGroupsEmptyGrantsFlattenToEmptyList(t *testing.T) {
	ctx := context.Background()
	empty := agList()

	prompt, diags := promptResponseToModel(ctx, nil, &client.PromptModel{ID: "p1", Command: "/p", Name: "P", Content: "c"})
	if diags.HasError() {
		t.Fatalf("prompt: %s", diags)
	}
	knowledge, diags := knowledgeResponseToModel(ctx, nil, client.KnowledgeFilesResponse{ID: "k1", Name: "K"})
	if diags.HasError() {
		t.Fatalf("knowledge: %s", diags)
	}

	for name, l := range map[string]types.List{
		"prompt.read_groups":     prompt.ReadGroups,
		"prompt.write_groups":    prompt.WriteGroups,
		"knowledge.read_groups":  knowledge.ReadGroups,
		"knowledge.write_groups": knowledge.WriteGroups,
	} {
		if !l.Equal(empty) {
			t.Errorf("%s = %s, want []", name, l)
		}
	}
}
