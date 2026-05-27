package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &userDataSource{}
var _ datasource.DataSourceWithConfigure = &userDataSource{}

// userDataSource exposes user details.
type userDataSource struct {
	client *client.Client
}

type userDataSourceModel struct {
	UserID          types.String `tfsdk:"user_id"`
	Query           types.String `tfsdk:"query"`
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Email           types.String `tfsdk:"email"`
	Username        types.String `tfsdk:"username"`
	Role            types.String `tfsdk:"role"`
	ProfileImageURL types.String `tfsdk:"profile_image_url"`
	Bio             types.String `tfsdk:"bio"`
	LastActiveAt    types.Int64  `tfsdk:"last_active_at"`
	UpdatedAt       types.Int64  `tfsdk:"updated_at"`
	CreatedAt       types.Int64  `tfsdk:"created_at"`
}

// NewUserDataSource constructs a new user data source.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// Metadata sets the data source type name.
func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the user data source schema.
func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Optional:    true,
				Description: "Identifier of the user to retrieve.",
			},
			"query": schema.StringAttribute{
				Optional:    true,
				Description: "Search query used to locate a user when user_id is not provided.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier assigned by Open WebUI.",
			},
			"name":              schema.StringAttribute{Computed: true},
			"email":             schema.StringAttribute{Computed: true},
			"username":          schema.StringAttribute{Computed: true},
			"role":              schema.StringAttribute{Computed: true},
			"profile_image_url": schema.StringAttribute{Computed: true},
			"bio":               schema.StringAttribute{Computed: true},
			"last_active_at":    schema.Int64Attribute{Computed: true},
			"updated_at":        schema.Int64Attribute{Computed: true},
			"created_at":        schema.Int64Attribute{Computed: true},
		},
	}
}

// Configure assigns the API client.
func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read retrieves user details.
func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the user data source.")
		return
	}

	var config userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := ""
	if !config.UserID.IsNull() && !config.UserID.IsUnknown() {
		userID = strings.TrimSpace(config.UserID.ValueString())
	}
	query := ""
	if !config.Query.IsNull() && !config.Query.IsUnknown() {
		query = strings.TrimSpace(config.Query.ValueString())
	}

	if userID == "" {
		if query == "" {
			resp.Diagnostics.AddError(
				"Missing user lookup value",
				"Either user_id or query must be provided to query a user.",
			)
			return
		}

		resolved, err := lookupUserID(ctx, d.client, query)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("query"),
				"Unable to resolve user",
				"No Open WebUI user could be resolved from the supplied query.",
			)
			return
		}
		userID = resolved
	}

	user, err := d.client.GetUser(ctx, userID)
	if err != nil {
		if err == client.ErrNotFound {
			resp.Diagnostics.AddAttributeError(
				path.Root("user_id"),
				"User not found",
				"No Open WebUI user was found with the supplied identifier.",
			)
			return
		}
		resp.Diagnostics.AddError("Read user failed", err.Error())
		return
	}

	username := types.StringNull()
	if user.Username != nil {
		username = types.StringValue(*user.Username)
	}
	profileImage := types.StringNull()
	if user.ProfileImage != "" {
		profileImage = types.StringValue(user.ProfileImage)
	}
	bio := types.StringNull()
	if user.Bio != nil {
		bio = types.StringValue(*user.Bio)
	}

	state := userDataSourceModel{
		UserID:          types.StringValue(user.ID),
		Query:           config.Query,
		ID:              types.StringValue(user.ID),
		Name:            types.StringValue(user.Name),
		Email:           types.StringValue(user.Email),
		Username:        username,
		Role:            types.StringValue(user.Role),
		ProfileImageURL: profileImage,
		Bio:             bio,
		LastActiveAt:    types.Int64Value(user.LastActiveAt),
		UpdatedAt:       types.Int64Value(user.UpdatedAt),
		CreatedAt:       types.Int64Value(user.CreatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
