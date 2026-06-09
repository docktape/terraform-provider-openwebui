package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &groupResource{}
var _ resource.ResourceWithConfigure = &groupResource{}
var _ resource.ResourceWithImportState = &groupResource{}

// groupResource manages Open WebUI groups.
type groupResource struct {
	client *client.Client
}

// groupResourceModel maps Terraform state.
// Permissions is types.Object (not groupPermissionsModel directly) so the framework
// can represent it as unknown during plan when the attribute is Optional+Computed.
type groupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Users       types.List   `tfsdk:"users"`
	Permissions types.Object `tfsdk:"permissions"`
	UserID      types.String `tfsdk:"user_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type groupPermissionsModel struct {
	Workspace types.Map `tfsdk:"workspace"`
	Sharing   types.Map `tfsdk:"sharing"`
	Chat      types.Map `tfsdk:"chat"`
	Features  types.Map `tfsdk:"features"`
}

// NewGroupResource constructs a new resource instance.
func NewGroupResource() resource.Resource {
	return &groupResource{}
}

// Metadata implements resource.Resource.
func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema defines the resource schema for groups.
func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user group in Open WebUI. Groups control workspace and chat permissions and are used for resource access control.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "UUID assigned by Open WebUI on create.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Unique display name of the group.",
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable description of the group's purpose.",
			},
			"users": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "List of user email addresses or UUIDs who are members of this group.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"permissions": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional permission overrides for group members. Omit to use Open WebUI defaults.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"workspace": schema.MapAttribute{
						Optional:      true,
						Computed:      true,
						ElementType:   types.BoolType,
						Description:   "Workspace-level permissions. Valid keys: `models`, `knowledge`, `prompts`, `tools`.",
						PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
						Validators: []validator.Map{
							mapvalidator.KeysAre(stringvalidator.OneOf(groupPermissionsWorkspaceKeys...)),
						},
					},
					"sharing": schema.MapAttribute{
						Optional:      true,
						Computed:      true,
						ElementType:   types.BoolType,
						Description:   "Sharing permissions. Valid keys: `public_models`, `public_knowledge`, `public_prompts`, `public_tools`.",
						PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
						Validators: []validator.Map{
							mapvalidator.KeysAre(stringvalidator.OneOf(groupPermissionsSharingKeys...)),
						},
					},
					"chat": schema.MapAttribute{
						Optional:      true,
						Computed:      true,
						ElementType:   types.BoolType,
						Description:   "Chat-level permissions. Valid keys: `controls`, `valves`, `system_prompt`, `params`, `file_upload`, `delete`, `delete_message`, `continue_response`, `regenerate_response`, `rate_response`, `edit`, `share`, `export`, `stt`, `tts`, `call`, `multiple_models`, `temporary`, `temporary_enforced`.",
						PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
						Validators: []validator.Map{
							mapvalidator.KeysAre(stringvalidator.OneOf(groupPermissionsChatKeys...)),
						},
					},
					"features": schema.MapAttribute{
						Optional:      true,
						Computed:      true,
						ElementType:   types.BoolType,
						Description:   "Feature access permissions. Valid keys: `direct_tool_servers`, `web_search`, `image_generation`, `code_interpreter`, `notes`.",
						PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
						Validators: []validator.Map{
							mapvalidator.KeysAre(stringvalidator.OneOf(groupPermissionsFeaturesKeys...)),
						},
					},
				},
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Owner user identifier. Set by Open WebUI.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "Creation date in `YYYY-MM-DD` format. Set by Open WebUI.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last-updated date in `YYYY-MM-DD` format. Set by Open WebUI.",
			},
		},
	}
}

// Configure connects the API client to the resource.
func (r *groupResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create provisions a group.
func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing groups.")
		return
	}

	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form := client.GroupForm{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	created, err := r.client.CreateGroup(ctx, form)
	if err != nil {
		resp.Diagnostics.AddError("Create group failed", err.Error())
		return
	}

	updateForm := client.GroupUpdateForm{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	providedUsers := !plan.Users.IsNull() && !plan.Users.IsUnknown()
	providedPermissions := permissionsObjectSpecified(ctx, plan.Permissions, &resp.Diagnostics)
	providedMeta := false
	providedData := false

	usernames := expandStringList(ctx, plan.Users, path.Root("users"), &resp.Diagnostics)
	resolvedUserIDs := uniqueStrings(resolveUsernamesToIDs(ctx, r.client, usernames, path.Root("users"), &resp.Diagnostics))

	if resp.Diagnostics.HasError() {
		return
	}

	if providedUsers {
		if err := r.client.AddGroupUsers(ctx, created.ID, resolvedUserIDs); err != nil {
			resp.Diagnostics.AddError("Add group members failed", err.Error())
			return
		}
	}

	planPermsModel := objectToPermissionsModel(ctx, plan.Permissions, &resp.Diagnostics)
	updateForm.Permissions = expandPermissions(ctx, planPermsModel, &resp.Diagnostics)
	updateForm.Meta = nil
	updateForm.Data = nil

	if resp.Diagnostics.HasError() {
		return
	}

	if providedPermissions || providedMeta || providedData {
		if _, err := r.client.UpdateGroup(ctx, created.ID, updateForm); err != nil {
			resp.Diagnostics.AddError("Update group failed", err.Error())
			return
		}
	}

	current, err := r.client.GetGroup(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Read group failed", err.Error())
		return
	}

	state, diags := groupResponseToModel(ctx, r.client, current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes state from the API.
func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing groups.")
		return
	}

	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.GetGroup(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read group failed", err.Error())
		return
	}

	updated, diags := groupResponseToModel(ctx, r.client, current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update mutates group properties.
func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing groups.")
		return
	}

	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.GetGroup(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read group failed", err.Error())
		return
	}

	form := client.GroupUpdateForm{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	usernames := expandStringList(ctx, plan.Users, path.Root("users"), &resp.Diagnostics)
	desiredIDs := uniqueStrings(resolveUsernamesToIDs(ctx, r.client, usernames, path.Root("users"), &resp.Diagnostics))
	planPermsModel := objectToPermissionsModel(ctx, plan.Permissions, &resp.Diagnostics)
	form.Permissions = expandPermissions(ctx, planPermsModel, &resp.Diagnostics)
	form.Meta = nil
	form.Data = nil

	if resp.Diagnostics.HasError() {
		return
	}

	toAdd, toRemove := diffStringSets(current.UserIDs, desiredIDs)

	if err := r.client.RemoveGroupUsers(ctx, plan.ID.ValueString(), toRemove); err != nil {
		resp.Diagnostics.AddError("Remove group members failed", err.Error())
		return
	}

	if err := r.client.AddGroupUsers(ctx, plan.ID.ValueString(), toAdd); err != nil {
		resp.Diagnostics.AddError("Add group members failed", err.Error())
		return
	}

	if _, err := r.client.UpdateGroup(ctx, plan.ID.ValueString(), form); err != nil {
		resp.Diagnostics.AddError("Update group failed", err.Error())
		return
	}

	fresh, err := r.client.GetGroup(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read group failed", err.Error())
		return
	}

	state, diags := groupResponseToModel(ctx, r.client, fresh)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the group.
func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing groups.")
		return
	}

	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteGroup(ctx, state.ID.ValueString()); err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Delete group failed", err.Error())
		return
	}
}

// ImportState passes the import identifier through to the id attribute.
func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// groupResponseToModel converts API structures to Terraform state.
func groupResponseToModel(ctx context.Context, apiClient *client.Client, resp *client.GroupResponse) (groupResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	permsStruct, permDiags := flattenPermissions(ctx, resp.Permissions)
	diags.Append(permDiags...)

	permissionsObj, objDiags := permissionsModelToObject(ctx, permsStruct)
	diags.Append(objDiags...)

	usernames, nameDiags := fetchUsernamesForIDs(ctx, apiClient, resp.UserIDs)
	diags.Append(nameDiags...)

	usersList, usersDiags := flattenStringSlice(ctx, usernames)
	diags.Append(usersDiags...)

	model := groupResourceModel{
		ID:          types.StringValue(resp.ID),
		Name:        types.StringValue(resp.Name),
		Description: types.StringValue(resp.Description),
		Users:       usersList,
		Permissions: permissionsObj,
		UserID:      types.StringValue(resp.UserID),
		CreatedAt:   formatDateValue(resp.CreatedAt),
		UpdatedAt:   formatDateValue(resp.UpdatedAt),
	}

	return model, diags
}

func resolveUsernamesToIDs(ctx context.Context, apiClient *client.Client, identifiers []string, attribute path.Path, diags *diag.Diagnostics) []string {
	if len(identifiers) == 0 {
		return nil
	}

	var ids []string
	for _, identifier := range identifiers {
		id, err := lookupUserID(ctx, apiClient, identifier)
		if err != nil {
			diags.AddAttributeError(
				attribute,
				"Unable to resolve user identifier",
				fmt.Sprintf("Failed to map %q to an Open WebUI user ID: %v", identifier, err),
			)
			continue
		}
		ids = append(ids, id)
	}

	return ids
}

func lookupUserID(ctx context.Context, apiClient *client.Client, identifier string) (string, error) {
	users, err := apiClient.SearchUsers(ctx, identifier, 50)
	if err != nil {
		return "", err
	}

	if len(users) == 0 {
		return "", fmt.Errorf("no matching users found")
	}

	// Prefer exact matches on email, username, or name.
	for _, u := range users {
		if strings.EqualFold(u.Email, identifier) {
			return u.ID, nil
		}
		if u.Username != nil && strings.EqualFold(*u.Username, identifier) {
			return u.ID, nil
		}
		if strings.EqualFold(u.Name, identifier) {
			return u.ID, nil
		}
	}

	if len(users) == 1 {
		return users[0].ID, nil
	}

	normalized := strings.ToLower(identifier)
	for _, u := range users {
		if strings.Contains(strings.ToLower(u.Email), normalized) {
			return u.ID, nil
		}
		if u.Username != nil && strings.Contains(strings.ToLower(*u.Username), normalized) {
			return u.ID, nil
		}
	}

	return users[0].ID, nil
}

func fetchUsernamesForIDs(ctx context.Context, apiClient *client.Client, ids []string) ([]string, diag.Diagnostics) {
	names := make([]string, 0, len(ids))
	var diags diag.Diagnostics

	for _, id := range ids {
		user, err := apiClient.GetUser(ctx, id)
		if err != nil {
			if err == client.ErrNotFound {
				continue
			}
			diags.AddError(
				"Fetch user failed",
				fmt.Sprintf("Failed to retrieve user %s: %v", id, err),
			)
			continue
		}

		label := user.Email
		if label == "" {
			if user.Username != nil && *user.Username != "" {
				label = *user.Username
			} else {
				label = user.Name
			}
		}

		if label == "" {
			label = id
		}

		names = append(names, label)
	}

	sort.Strings(names)
	return names, diags
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	var result []string
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}

	return result
}

func diffStringSets(current, desired []string) (toAdd, toRemove []string) {
	currentSet := make(map[string]struct{}, len(current))
	desiredSet := make(map[string]struct{}, len(desired))

	for _, id := range current {
		currentSet[id] = struct{}{}
	}
	for _, id := range desired {
		desiredSet[id] = struct{}{}
		if _, ok := currentSet[id]; !ok {
			toAdd = append(toAdd, id)
		}
	}

	for id := range currentSet {
		if _, ok := desiredSet[id]; !ok {
			toRemove = append(toRemove, id)
		}
	}

	return uniqueStrings(toAdd), uniqueStrings(toRemove)
}
