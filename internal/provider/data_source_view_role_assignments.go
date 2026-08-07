package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
)

var (
	_ datasource.DataSource              = &viewRoleAssignmentsDataSource{}
	_ datasource.DataSourceWithConfigure = &viewRoleAssignmentsDataSource{}
)

// NewViewRoleAssignmentsDataSource lists view-scoped role assignments held by
// a LogScale group. View-scoped role assignments live on the group object as
// the `roles` field, where each entry carries both the role and the search
// domain (view or repository) the assignment applies to.
func NewViewRoleAssignmentsDataSource() datasource.DataSource {
	return &viewRoleAssignmentsDataSource{}
}

type viewRoleAssignmentsDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type viewRoleAssignmentsDataSourceModel struct {
	ID          types.String                    `tfsdk:"id"`
	GroupID     types.String                    `tfsdk:"group_id"`
	Assignments []viewRoleAssignmentsDataModel `tfsdk:"assignments"`
}

type viewRoleAssignmentsDataModel struct {
	ID              types.String `tfsdk:"id"`
	RoleID          types.String `tfsdk:"role_id"`
	RoleDisplayName types.String `tfsdk:"role_display_name"`
	ViewID          types.String `tfsdk:"view_id"`
	ViewName        types.String `tfsdk:"view_name"`
}

func (d *viewRoleAssignmentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view_role_assignments"
}

func (d *viewRoleAssignmentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists view-scoped role assignments held by a LogScale group. " +
			"Each entry corresponds to a role granted on a specific view or repository " +
			"(LogScale models repositories as a kind of view).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic identifier for this listing (echoes the group_id).",
				Computed:    true,
			},
			"group_id": schema.StringAttribute{
				Description: "The ID of the group whose view-scoped role assignments are listed.",
				Required:    true,
			},
			"assignments": schema.ListNestedAttribute{
				Description: "View-scoped role assignments associated with the group.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Synthetic assignment identifier (group_id:view_id:role_id), matching the resource ID format used by logscale_view_role_assignment.",
							Computed:    true,
						},
						"role_id": schema.StringAttribute{
							Description: "The ID of the assigned role.",
							Computed:    true,
						},
						"role_display_name": schema.StringAttribute{
							Description: "The display name of the assigned role.",
							Computed:    true,
						},
						"view_id": schema.StringAttribute{
							Description: "The ID of the view or repository the role applies to.",
							Computed:    true,
						},
						"view_name": schema.StringAttribute{
							Description: "The name of the view or repository the role applies to.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *viewRoleAssignmentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.config = config

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.APIToken},
	)
	d.client = oauth2.NewClient(context.Background(), src)
}

func (d *viewRoleAssignmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data viewRoleAssignmentsDataSourceModel

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.GroupID.IsNull() || data.GroupID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"'group_id' must be specified.",
		)
		return
	}

	queryStr := `
		query GroupViewRoleAssignments($groupId: String!) {
			group(groupId: $groupId) {
				id
				displayName
				roles {
					role {
						id
						displayName
					}
					searchDomain {
						id
						name
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"groupId": data.GroupID.ValueString(),
	}

	payload := map[string]interface{}{
		"query":     queryStr,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing view role assignments",
			"Could not marshal request: "+err.Error(),
		)
		return
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		d.config.Endpoint,
		bytes.NewBuffer(jsonPayload),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing view role assignments",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := d.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing view role assignments",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing view role assignments",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			Group *struct {
				ID    string `json:"id"`
				Roles []struct {
					Role struct {
						ID          string `json:"id"`
						DisplayName string `json:"displayName"`
					} `json:"role"`
					SearchDomain *struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"searchDomain"`
				} `json:"roles"`
			} `json:"group"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error listing view role assignments",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error listing view role assignments",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}

	if gqlResp.Data.Group == nil {
		resp.Diagnostics.AddError(
			"Group Not Found",
			fmt.Sprintf("No group found with ID: %s", data.GroupID.ValueString()),
		)
		return
	}

	groupID := gqlResp.Data.Group.ID
	if groupID == "" {
		groupID = data.GroupID.ValueString()
	}

	assignments := make([]viewRoleAssignmentsDataModel, 0, len(gqlResp.Data.Group.Roles))
	for _, gr := range gqlResp.Data.Group.Roles {
		// Skip entries without a search domain — those are organization- or
		// system-scoped role grants that live elsewhere in the API surface.
		if gr.SearchDomain == nil || gr.SearchDomain.ID == "" {
			continue
		}
		assignments = append(assignments, viewRoleAssignmentsDataModel{
			ID:              types.StringValue(viewRoleAssignmentID(groupID, gr.SearchDomain.ID, gr.Role.ID)),
			RoleID:          types.StringValue(gr.Role.ID),
			RoleDisplayName: types.StringValue(gr.Role.DisplayName),
			ViewID:          types.StringValue(gr.SearchDomain.ID),
			ViewName:        types.StringValue(gr.SearchDomain.Name),
		})
	}

	data.ID = types.StringValue(groupID)
	data.Assignments = assignments

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
