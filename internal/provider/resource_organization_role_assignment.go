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
	_ resource.Resource              = &organizationRoleAssignmentResource{}
	_ resource.ResourceWithConfigure = &organizationRoleAssignmentResource{}
)

// NewOrganizationRoleAssignmentResource is a helper function to simplify the provider implementation.
func NewOrganizationRoleAssignmentResource() resource.Resource {
	return &organizationRoleAssignmentResource{}
}

// organizationRoleAssignmentResource is the resource implementation.
type organizationRoleAssignmentResource struct {
	client *http.Client
	config *LogScaleConfig
}

// organizationRoleAssignmentResourceModel maps the resource schema data.
type organizationRoleAssignmentResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.String `tfsdk:"group_id"`
	RoleID  types.String `tfsdk:"role_id"`
}

// Metadata returns the resource type name.
func (r *organizationRoleAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role_assignment"
}

// Schema defines the schema for the resource.
func (r *organizationRoleAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns an organization-level role to a group in LogScale.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Assignment identifier (group_id:role_id)",
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
			"role_id": schema.StringAttribute{
				Description: "The ID of the organization role to assign",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *organizationRoleAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	// Create OAuth2 client
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.APIToken},
	)
	r.client = oauth2.NewClient(context.Background(), src)
}

// Create creates the resource and sets the initial Terraform state.
func (r *organizationRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan organizationRoleAssignmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// GraphQL mutation - exactly as in your reference code
	mutationStr := `
		mutation AssignOrgRole($input: AssignOrganizationRoleToGroupInput!) {
			assignOrganizationRoleToGroup(input: $input) {
				group {
					role {
						id
						displayName
						organizationPermissions
					}
				}
			}
		}
	`

	input := map[string]interface{}{
		"groupId": plan.GroupID.ValueString(),
		"roleId":  plan.RoleID.ValueString(),
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
			"Error assigning organization role to group",
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
			"Error assigning organization role to group",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning organization role to group",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning organization role to group",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			AssignOrganizationRoleToGroup struct {
				Group struct {
					Role struct {
						ID                      string   `json:"id"`
						DisplayName             string   `json:"displayName"`
						OrganizationPermissions []string `json:"organizationPermissions"`
					} `json:"role"`
				} `json:"group"`
			} `json:"assignOrganizationRoleToGroup"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error assigning organization role to group",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error assigning organization role to group",
			fmt.Sprintf("GraphQL error: %s\nFull response: %s", gqlResp.Errors[0].Message, string(body)),
		)
		return
	}

	// Create unique ID for this assignment
	plan.ID = types.StringValue(fmt.Sprintf("%s:%s",
		plan.GroupID.ValueString(),
		plan.RoleID.ValueString()))

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *organizationRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state organizationRoleAssignmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For now, assume the assignment exists if it's in state
	// In production, you might query the group and verify the role is assigned
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *organizationRoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Organization role assignments cannot be updated - they must be recreated
	resp.Diagnostics.AddError(
		"Organization Role Assignment Update Not Supported",
		"Organization role assignments cannot be updated. They must be recreated.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *organizationRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var state organizationRoleAssignmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// GraphQL mutation to remove organization role from group
	mutationStr := `
		mutation RemoveOrgRoleFromGroup($input: RemoveOrganizationRoleFromGroupInput!) {
			removeOrganizationRoleFromGroup(input: $input) {
				group {
					id
				}
			}
		}
	`

	input := map[string]interface{}{
		"groupId": state.GroupID.ValueString(),
		"roleId":  state.RoleID.ValueString(),
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
			"Error removing organization role from group",
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
			"Error removing organization role from group",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing organization role from group",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing organization role from group",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			RemoveOrganizationRoleFromGroup struct {
				Group struct {
					ID string `json:"id"`
				} `json:"group"`
			} `json:"removeOrganizationRoleFromGroup"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error removing organization role from group",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error removing organization role from group",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}

	// Resource is deleted, Terraform will remove from state automatically
}
