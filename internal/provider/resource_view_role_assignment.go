package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &viewRoleAssignmentResource{}
	_ resource.ResourceWithConfigure = &viewRoleAssignmentResource{}
)

// NewViewRoleAssignmentResource is a helper function to simplify the provider implementation.
func NewViewRoleAssignmentResource() resource.Resource {
	return &viewRoleAssignmentResource{}
}

// viewRoleAssignmentResource is the resource implementation.
type viewRoleAssignmentResource struct {
	client *http.Client
	config *LogScaleConfig
}

// viewRoleAssignmentResourceModel maps the resource schema data.
type viewRoleAssignmentResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.String `tfsdk:"group_id"`
	ViewID  types.String `tfsdk:"view_id"`
	RoleID  types.String `tfsdk:"role_id"`
}

// Metadata returns the resource type name.
func (r *viewRoleAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view_role_assignment"
}

// Schema defines the schema for the resource.
func (r *viewRoleAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a view-scoped role to a group on a specific view or repository in LogScale. " +
			"Repositories are treated as views in LogScale, so view_id may be either a view ID or a repository ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Assignment identifier (group_id:view_id:role_id)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "The ID of the group",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"view_id": schema.StringAttribute{
				Description: "The ID of the view or repository the role applies to",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Description: "The ID of the role to assign on this view",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *viewRoleAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.config = config

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.APIToken},
	)
	r.client = oauth2.NewClient(context.Background(), src)
}

// Create creates the resource and sets the initial Terraform state.
func (r *viewRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan viewRoleAssignmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation AssignViewRole($input: AssignRoleToGroupInput!) {
			assignRoleToGroup(input: $input) {
				group {
					role {
						id
						displayName
						viewPermissions
					}
				}
			}
		}
	`

	input := map[string]interface{}{
		"groupId": plan.GroupID.ValueString(),
		"roleId":  plan.RoleID.ValueString(),
		"viewId":  plan.ViewID.ValueString(),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	payload := map[string]interface{}{
		"query":     mutationStr,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning view role to group",
			"Could not marshal request: "+err.Error(),
		)
		return
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		r.config.Endpoint,
		bytes.NewBuffer(jsonPayload),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning view role to group",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning view role to group",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning view role to group",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			AssignRoleToGroup struct {
				Group struct {
					Role struct {
						ID              string   `json:"id"`
						DisplayName     string   `json:"displayName"`
						ViewPermissions []string `json:"viewPermissions"`
					} `json:"role"`
				} `json:"group"`
			} `json:"assignRoleToGroup"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error assigning view role to group",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error assigning view role to group",
			fmt.Sprintf("GraphQL error: %s\nFull response: %s", gqlResp.Errors[0].Message, string(body)),
		)
		return
	}

	plan.ID = types.StringValue(viewRoleAssignmentID(
		plan.GroupID.ValueString(),
		plan.ViewID.ValueString(),
		plan.RoleID.ValueString(),
	))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// viewRoleAssignmentID builds the synthetic assignment identifier used in
// Terraform state. The order is group_id:view_id:role_id and is referenced by
// the docs and the matching list data source.
func viewRoleAssignmentID(groupID, viewID, roleID string) string {
	return fmt.Sprintf("%s:%s:%s", groupID, viewID, roleID)
}

// Read refreshes the Terraform state with the latest data.
func (r *viewRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state viewRoleAssignmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Matches the org/sys role-assignment pattern: state is preserved rather
	// than reconciled against the remote assignment. Drift detection on this
	// resource is a known follow-up item.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *viewRoleAssignmentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"View Role Assignment Update Not Supported",
		"View role assignments cannot be updated. They must be recreated.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *viewRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state viewRoleAssignmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation RemoveViewRole($input: RemoveRoleFromGroupInput!) {
			removeRoleFromGroup(input: $input) {
				group {
					id
				}
			}
		}
	`

	input := map[string]interface{}{
		"groupId": state.GroupID.ValueString(),
		"roleId":  state.RoleID.ValueString(),
		"viewId":  state.ViewID.ValueString(),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	payload := map[string]interface{}{
		"query":     mutationStr,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing view role from group",
			"Could not marshal request: "+err.Error(),
		)
		return
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		r.config.Endpoint,
		bytes.NewBuffer(jsonPayload),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing view role from group",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing view role from group",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing view role from group",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			RemoveRoleFromGroup struct {
				Group struct {
					ID string `json:"id"`
				} `json:"group"`
			} `json:"removeRoleFromGroup"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error removing view role from group",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error removing view role from group",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}
}
