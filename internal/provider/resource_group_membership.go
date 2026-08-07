package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &groupMembershipResource{}
	_ resource.ResourceWithConfigure = &groupMembershipResource{}
)

// NewGroupMembershipResource is a helper function to simplify the provider implementation.
func NewGroupMembershipResource() resource.Resource {
	return &groupMembershipResource{}
}

// groupMembershipResource is the resource implementation.
type groupMembershipResource struct {
	client *http.Client
	config *LogScaleConfig
}

// groupMembershipResourceModel maps the resource schema data.
type groupMembershipResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.String `tfsdk:"group_id"`
	Users   types.Set    `tfsdk:"users"`
}

// Metadata returns the resource type name.
func (r *groupMembershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

// Schema defines the schema for the resource.
func (r *groupMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages users in a LogScale group. Add or remove multiple users from a group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Group membership identifier",
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
			"users": schema.SetAttribute{
				Description: "Set of usernames to add to the group",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *groupMembershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *groupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan groupMembershipResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get users list
	var users []string
	diags = plan.Users.ElementsAs(ctx, &users, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// GraphQL mutation - exactly as in your reference code
	mutationStr := `
		mutation AddUsers($input: AddUsersToGroupInput!) {
			addUsersToGroup(input: $input) {
				group {
					displayName
					userCount
				}
			}
		}
	`

	input := map[string]interface{}{
		"groupId": plan.GroupID.ValueString(),
		"users":   users,
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
			"Error adding users to group",
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
			"Error adding users to group",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding users to group",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding users to group",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			AddUsersToGroup struct {
				Group struct {
					DisplayName string `json:"displayName"`
					UserCount   int    `json:"userCount"`
				} `json:"group"`
			} `json:"addUsersToGroup"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error adding users to group",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error adding users to group",
			fmt.Sprintf("GraphQL error: %s\nFull response: %s", gqlResp.Errors[0].Message, string(body)),
		)
		return
	}

	// Set ID
	plan.ID = types.StringValue(plan.GroupID.ValueString())

	r.refreshGroupMembershipState(ctx, plan.GroupID.ValueString(), users, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *groupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state groupMembershipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remoteUsers, found := r.queryGroupUsers(ctx, state.GroupID.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	userSet, diags := types.SetValueFrom(ctx, types.StringType, remoteUsers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(state.GroupID.ValueString())
	state.Users = userSet

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *groupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get plan and current state
	var plan groupMembershipResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state groupMembershipResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current and planned users
	var currentUsers []string
	diags = state.Users.ElementsAs(ctx, &currentUsers, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plannedUsers []string
	diags = plan.Users.ElementsAs(ctx, &plannedUsers, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Calculate users to add and remove
	currentUserSet := make(map[string]bool)
	for _, user := range currentUsers {
		currentUserSet[user] = true
	}

	plannedUserSet := make(map[string]bool)
	for _, user := range plannedUsers {
		plannedUserSet[user] = true
	}

	// Users to add (in planned but not in current)
	var usersToAdd []string
	for _, user := range plannedUsers {
		if !currentUserSet[user] {
			usersToAdd = append(usersToAdd, user)
		}
	}

	// Users to remove (in current but not in planned)
	var usersToRemove []string
	for _, user := range currentUsers {
		if !plannedUserSet[user] {
			usersToRemove = append(usersToRemove, user)
		}
	}

	// Add new users
	if len(usersToAdd) > 0 {
		mutationStr := `
			mutation AddUsers($input: AddUsersToGroupInput!) {
				addUsersToGroup(input: $input) {
					group {
						displayName
						userCount
					}
				}
			}
		`

		input := map[string]interface{}{
			"groupId": plan.GroupID.ValueString(),
			"users":   usersToAdd,
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
				"Error adding users to group",
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
				"Error adding users to group",
				"Could not create HTTP request: "+err.Error(),
			)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := r.client.Do(httpReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding users to group",
				"Could not execute HTTP request: "+err.Error(),
			)
			return
		}
		defer httpResp.Body.Close()

		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding users to group",
				"Could not read response body: "+err.Error(),
			)
			return
		}

		var gqlResp struct {
			Data struct {
				AddUsersToGroup struct {
					Group struct {
						DisplayName string `json:"displayName"`
						UserCount   int    `json:"userCount"`
					} `json:"group"`
				} `json:"addUsersToGroup"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}

		if err := json.Unmarshal(body, &gqlResp); err != nil {
			resp.Diagnostics.AddError(
				"Error adding users to group",
				fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
			)
			return
		}

		if len(gqlResp.Errors) > 0 {
			resp.Diagnostics.AddError(
				"Error adding users to group",
				fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
			)
			return
		}
	}

	// Remove users
	if len(usersToRemove) > 0 {
		mutationStr := `
			mutation RemoveUsers($input: RemoveUsersFromGroupInput!) {
				removeUsersFromGroup(input: $input) {
					group {
						displayName
						userCount
					}
				}
			}
		`

		input := map[string]interface{}{
			"groupId": plan.GroupID.ValueString(),
			"users":   usersToRemove,
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
				"Error removing users from group",
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
				"Error removing users from group",
				"Could not create HTTP request: "+err.Error(),
			)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := r.client.Do(httpReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing users from group",
				"Could not execute HTTP request: "+err.Error(),
			)
			return
		}
		defer httpResp.Body.Close()

		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing users from group",
				"Could not read response body: "+err.Error(),
			)
			return
		}

		var gqlResp struct {
			Data struct {
				RemoveUsersFromGroup struct {
					Group struct {
						DisplayName string `json:"displayName"`
						UserCount   int    `json:"userCount"`
					} `json:"group"`
				} `json:"removeUsersFromGroup"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}

		if err := json.Unmarshal(body, &gqlResp); err != nil {
			resp.Diagnostics.AddError(
				"Error removing users from group",
				fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
			)
			return
		}

		if len(gqlResp.Errors) > 0 {
			resp.Diagnostics.AddError(
				"Error removing users from group",
				fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
			)
			return
		}
	}

	r.refreshGroupMembershipState(ctx, plan.GroupID.ValueString(), plannedUsers, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *groupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var state groupMembershipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get users to remove
	var users []string
	diags = state.Users.ElementsAs(ctx, &users, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove all users from the group
	mutationStr := `
		mutation RemoveUsers($input: RemoveUsersFromGroupInput!) {
			removeUsersFromGroup(input: $input) {
				group {
					displayName
					userCount
				}
			}
		}
	`

	input := map[string]interface{}{
		"groupId": state.GroupID.ValueString(),
		"users":   users,
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
			"Error removing users from group",
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
			"Error removing users from group",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing users from group",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing users from group",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			RemoveUsersFromGroup struct {
				Group struct {
					DisplayName string `json:"displayName"`
					UserCount   int    `json:"userCount"`
				} `json:"group"`
			} `json:"removeUsersFromGroup"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error removing users from group",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error removing users from group",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}

	// Resource is deleted, Terraform will remove from state automatically
}

func (r *groupMembershipResource) queryGroupUsers(ctx context.Context, groupID string, diagnostics *diag.Diagnostics) ([]string, bool) {
	queryStr := `
		query GetGroupUsers($groupId: String!) {
			group(groupId: $groupId) {
				id
				users {
					username
				}
			}
		}
	`

	variables := map[string]interface{}{
		"groupId": groupID,
	}

	payload := map[string]interface{}{
		"query":     queryStr,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		diagnostics.AddError("Error reading group membership", "Could not marshal request: "+err.Error())
		return nil, false
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", r.config.Endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		diagnostics.AddError("Error reading group membership", "Could not create HTTP request: "+err.Error())
		return nil, false
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		diagnostics.AddError("Error reading group membership", "Could not execute HTTP request: "+err.Error())
		return nil, false
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		diagnostics.AddError("Error reading group membership", "Could not read response body: "+err.Error())
		return nil, false
	}

	var gqlResp struct {
		Data struct {
			Group *struct {
				ID    string `json:"id"`
				Users []struct {
					Username string `json:"username"`
				} `json:"users"`
			} `json:"group"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		diagnostics.AddError("Error reading group membership", fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body))
		return nil, false
	}

	if len(gqlResp.Errors) > 0 {
		diagnostics.AddError("Error reading group membership", fmt.Sprintf("GraphQL error: %s\nFull response: %s", gqlResp.Errors[0].Message, string(body)))
		return nil, false
	}

	if gqlResp.Data.Group == nil {
		return nil, false
	}

	users := make([]string, 0, len(gqlResp.Data.Group.Users))
	for _, user := range gqlResp.Data.Group.Users {
		if user.Username != "" {
			users = append(users, user.Username)
		}
	}
	sort.Strings(users)

	return users, true
}

func (r *groupMembershipResource) refreshGroupMembershipState(ctx context.Context, groupID string, expectedUsers []string, state *groupMembershipResourceModel, diagnostics *diag.Diagnostics) {
	remoteUsers, found := r.queryGroupUsers(ctx, groupID, diagnostics)
	if diagnostics.HasError() {
		return
	}

	if !found {
		diagnostics.AddWarning(
			"Group Membership Readback Failed",
			fmt.Sprintf("The group %q could not be read after mutation. Terraform will keep the requested state, but the remote membership should be verified manually.", groupID),
		)

		userSet, diags := types.SetValueFrom(ctx, types.StringType, expectedUsers)
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return
		}
		state.ID = types.StringValue(groupID)
		state.Users = userSet
		return
	}

	userSet, diags := types.SetValueFrom(ctx, types.StringType, remoteUsers)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(groupID)
	state.GroupID = types.StringValue(groupID)
	state.Users = userSet

	expectedSet := make(map[string]struct{}, len(expectedUsers))
	for _, user := range expectedUsers {
		expectedSet[user] = struct{}{}
	}
	remoteSet := make(map[string]struct{}, len(remoteUsers))
	for _, user := range remoteUsers {
		remoteSet[user] = struct{}{}
	}

	var missing []string
	for _, user := range expectedUsers {
		if _, ok := remoteSet[user]; !ok {
			missing = append(missing, user)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		diagnostics.AddWarning(
			"Group Membership Requires Manual Verification",
			fmt.Sprintf("The requested usernames %q were not confirmed in the remote group membership after apply. This may indicate username-format mismatch or API behavior differences. Verify the group membership manually in LogScale.", missing),
		)
	}

	if len(expectedUsers) != len(remoteUsers) {
		diagnostics.AddWarning(
			"Group Membership Readback Differs From Requested State",
			"The remote group membership does not exactly match the requested set of users. Terraform state has been updated to the remote usernames returned by LogScale.",
		)
		return
	}

	for _, user := range expectedUsers {
		if _, ok := remoteSet[user]; !ok {
			return
		}
	}
}
