package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
)

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithConfigure   = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

// NewRoleResource manages LogScale roles via the createRole / updateRole /
// removeRole mutations. View permissions are modelled as a set of strings
// matching the permission identifiers accepted by the LogScale GraphQL surface
// (see the LogScale "permissions" enumeration in upstream docs).
func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	client *http.Client
	config *LogScaleConfig
}

type roleResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DisplayName     types.String `tfsdk:"display_name"`
	ViewPermissions types.Set    `tfsdk:"view_permissions"`
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale role. Roles bundle a set of view-level permissions " +
			"that can be granted to a group via logscale_view_role_assignment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Role ID assigned by LogScale at create time.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Display name for the role.",
				Required:    true,
			},
			"view_permissions": schema.SetAttribute{
				Description: "Set of view-level permission identifiers granted by this role.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions := make([]string, 0, len(plan.ViewPermissions.Elements()))
	diags = plan.ViewPermissions.ElementsAs(ctx, &permissions, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation CreateRole($input: CreateRoleInput!) {
			createRole(input: $input) {
				role {
					id
					displayName
					viewPermissions
				}
			}
		}
	`

	input := map[string]interface{}{
		"displayName":     plan.DisplayName.ValueString(),
		"viewPermissions": permissions,
	}

	body, err := r.gqlCall(ctx, mutationStr, map[string]interface{}{"input": input})
	if err != nil {
		resp.Diagnostics.AddError("Error creating role", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			CreateRole struct {
				Role struct {
					ID              string   `json:"id"`
					DisplayName     string   `json:"displayName"`
					ViewPermissions []string `json:"viewPermissions"`
				} `json:"role"`
			} `json:"createRole"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error creating role",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error creating role",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}

	role := gqlResp.Data.CreateRole.Role
	plan.ID = types.StringValue(role.ID)
	plan.DisplayName = types.StringValue(role.DisplayName)

	perms, permDiags := types.SetValueFrom(ctx, types.StringType, role.ViewPermissions)
	resp.Diagnostics.Append(permDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ViewPermissions = perms

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	queryStr := `
		query GetRole($roleId: String!) {
			role(roleId: $roleId) {
				displayName
				viewPermissions
			}
		}
	`

	body, err := r.gqlCall(ctx, queryStr, map[string]interface{}{"roleId": state.ID.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error reading role", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			Role *struct {
				DisplayName     string   `json:"displayName"`
				ViewPermissions []string `json:"viewPermissions"`
			} `json:"role"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error reading role",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error reading role",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}

	if gqlResp.Data.Role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.DisplayName = types.StringValue(gqlResp.Data.Role.DisplayName)
	perms, permDiags := types.SetValueFrom(ctx, types.StringType, gqlResp.Data.Role.ViewPermissions)
	resp.Diagnostics.Append(permDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ViewPermissions = perms

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state roleResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions := make([]string, 0, len(plan.ViewPermissions.Elements()))
	diags = plan.ViewPermissions.ElementsAs(ctx, &permissions, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation UpdateRole($input: UpdateRoleInput!) {
			updateRole(input: $input) {
				role {
					id
					displayName
					viewPermissions
				}
			}
		}
	`

	input := map[string]interface{}{
		"roleId":          state.ID.ValueString(),
		"displayName":     plan.DisplayName.ValueString(),
		"viewPermissions": permissions,
	}

	body, err := r.gqlCall(ctx, mutationStr, map[string]interface{}{"input": input})
	if err != nil {
		resp.Diagnostics.AddError("Error updating role", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			UpdateRole struct {
				Role struct {
					ID              string   `json:"id"`
					DisplayName     string   `json:"displayName"`
					ViewPermissions []string `json:"viewPermissions"`
				} `json:"role"`
			} `json:"updateRole"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error updating role",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error updating role",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}

	role := gqlResp.Data.UpdateRole.Role
	plan.ID = state.ID
	if role.DisplayName != "" {
		plan.DisplayName = types.StringValue(role.DisplayName)
	}
	if role.ViewPermissions != nil {
		perms, permDiags := types.SetValueFrom(ctx, types.StringType, role.ViewPermissions)
		resp.Diagnostics.Append(permDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.ViewPermissions = perms
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation RemoveRole($roleId: String!) {
			removeRole(roleId: $roleId) {
				result
			}
		}
	`

	body, err := r.gqlCall(ctx, mutationStr, map[string]interface{}{"roleId": state.ID.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error removing role", err.Error())
		return
	}

	var gqlResp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error removing role",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}
	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error removing role",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier to be the role ID.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// gqlCall executes a GraphQL POST against the configured endpoint and returns
// the raw response body. The caller is responsible for unmarshalling and
// inspecting the GraphQL `errors` array.
func (r *roleResource) gqlCall(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", r.config.Endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}
	return body, nil
}
