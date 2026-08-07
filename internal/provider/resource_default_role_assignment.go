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

var (
	_ resource.Resource              = &defaultRoleAssignmentResource{}
	_ resource.ResourceWithConfigure = &defaultRoleAssignmentResource{}
)

// NewDefaultRoleAssignmentResource configures the default role for a LogScale
// group via the updateDefaultRole mutation. The default role is a single piece
// of state on the group, so destroy is best-effort — the mutation surface does
// not expose a clean "unset" verb. The resource emits a warning on destroy and
// detaches state.
func NewDefaultRoleAssignmentResource() resource.Resource {
	return &defaultRoleAssignmentResource{}
}

type defaultRoleAssignmentResource struct {
	client *http.Client
	config *LogScaleConfig
}

type defaultRoleAssignmentResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.String `tfsdk:"group_id"`
	RoleID  types.String `tfsdk:"role_id"`
}

func (r *defaultRoleAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_default_role_assignment"
}

func (r *defaultRoleAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Configures the default role applied to a LogScale group's view-scoped " +
			"role assignment behavior via updateDefaultRole. Read preserves Terraform state " +
			"rather than reconciling against the remote group, mirroring the other role-assignment " +
			"resources. Destroy emits a warning and detaches state because the upstream API does " +
			"not expose a clean unset path.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Assignment identifier (group_id:role_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "The ID of the group whose default role is being configured.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Description: "The ID of the role to set as the default for this group. Updating this attribute re-invokes updateDefaultRole.",
				Required:    true,
			},
		},
	}
}

func (r *defaultRoleAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *defaultRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan defaultRoleAssignmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.callUpdateDefaultRole(ctx, plan.GroupID.ValueString(), plan.RoleID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error setting default role on group", err.Error())
		return
	}

	plan.ID = types.StringValue(defaultRoleAssignmentID(plan.GroupID.ValueString(), plan.RoleID.ValueString()))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *defaultRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state defaultRoleAssignmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Matches the org/sys role-assignment pattern: state is preserved rather
	// than reconciled against the remote default-role configuration.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *defaultRoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan defaultRoleAssignmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.callUpdateDefaultRole(ctx, plan.GroupID.ValueString(), plan.RoleID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating default role on group", err.Error())
		return
	}

	plan.ID = types.StringValue(defaultRoleAssignmentID(plan.GroupID.ValueString(), plan.RoleID.ValueString()))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *defaultRoleAssignmentResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Default Role Assignment Destroy Detaches State Only",
		"LogScale does not expose a clean 'unset default role' mutation. Terraform has removed the "+
			"resource from state, but the group's default role on the remote tenant is unchanged. "+
			"If you intend to switch the default role to a different role, configure the new role on this resource "+
			"and apply. If you intend to remove the default role entirely, do so manually in the LogScale UI.",
	)
}

func defaultRoleAssignmentID(groupID, roleID string) string {
	return fmt.Sprintf("%s:%s", groupID, roleID)
}

func (r *defaultRoleAssignmentResource) callUpdateDefaultRole(ctx context.Context, groupID, roleID string) error {
	mutationStr := `
		mutation UpdateDefaultRole($input: UpdateDefaultRoleInput!) {
			updateDefaultRole(input: $input) {
				group {
					id
					displayName
				}
			}
		}
	`

	input := map[string]interface{}{
		"groupId": groupID,
		"roleId":  roleID,
	}

	payload := map[string]interface{}{
		"query":     mutationStr,
		"variables": map[string]interface{}{"input": input},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", r.config.Endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("could not create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("could not execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %w", err)
	}

	var gqlResp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return fmt.Errorf("could not unmarshal response: %v\nBody: %s", err, body)
	}
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}
	return nil
}
