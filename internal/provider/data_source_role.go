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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &roleDataSource{}
	_ datasource.DataSourceWithConfigure = &roleDataSource{}
)

// NewRoleDataSource is a helper function to simplify the provider implementation.
func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

// roleDataSource is the data source implementation.
type roleDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

// roleDataSourceModel maps the data source schema data.
type roleDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	DisplayName     types.String `tfsdk:"display_name"`
	ViewPermissions types.List   `tfsdk:"view_permissions"`
}

// Metadata returns the data source type name.
func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the data source.
func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches details about an existing LogScale role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Role ID. Either id or display_name must be specified.",
				Optional:    true,
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "Role display name. Either id or display_name must be specified.",
				Optional:    true,
				Computed:    true,
			},
			"view_permissions": schema.ListAttribute{
				Description: "List of view permissions assigned to this role",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	// Create OAuth2 client
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.APIToken},
	)
	d.client = oauth2.NewClient(context.Background(), src)
}

// Read refreshes the Terraform state with the latest data.
func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data roleDataSourceModel

	// Read configuration
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that either ID or DisplayName is provided
	if data.ID.IsNull() && data.DisplayName.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'display_name' must be specified.",
		)
		return
	}

	// If we have display_name but not ID, we need to find the role by name
	if !data.DisplayName.IsNull() && data.ID.IsNull() {
		// Query to get all roles and find by display name
		queryStr := `
			query GetRoles {
				roles {
					id
					displayName
					viewPermissions
				}
			}
		`

		payload := map[string]interface{}{
			"query": queryStr,
		}

		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading roles",
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
				"Error reading roles",
				"Could not create HTTP request: "+err.Error(),
			)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := d.client.Do(httpReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading roles",
				"Could not execute HTTP request: "+err.Error(),
			)
			return
		}
		defer httpResp.Body.Close()

		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading roles",
				"Could not read response body: "+err.Error(),
			)
			return
		}

		var gqlResp struct {
			Data struct {
				Roles []struct {
					ID              string   `json:"id"`
					DisplayName     string   `json:"displayName"`
					ViewPermissions []string `json:"viewPermissions"`
				} `json:"roles"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}

		if err := json.Unmarshal(body, &gqlResp); err != nil {
			resp.Diagnostics.AddError(
				"Error reading roles",
				fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
			)
			return
		}

		if len(gqlResp.Errors) > 0 {
			resp.Diagnostics.AddError(
				"Error reading roles",
				fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
			)
			return
		}

		// Find the role by display name
		found := false
		for _, role := range gqlResp.Data.Roles {
			if role.DisplayName == data.DisplayName.ValueString() {
				data.ID = types.StringValue(role.ID)
				data.DisplayName = types.StringValue(role.DisplayName)

				// Convert permissions to list
				permissionsList, diags := types.ListValueFrom(ctx, types.StringType, role.ViewPermissions)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				data.ViewPermissions = permissionsList

				found = true
				break
			}
		}

		if !found {
			resp.Diagnostics.AddError(
				"Role Not Found",
				fmt.Sprintf("No role found with display_name: %s", data.DisplayName.ValueString()),
			)
			return
		}
	} else {
		// We have an ID, query directly by ID
		queryStr := `
			query GetRoleDetails($roleId: String!) {
				role(roleId: $roleId) {
					displayName
					viewPermissions
				}
			}
		`

		variables := map[string]interface{}{
			"roleId": data.ID.ValueString(),
		}

		payload := map[string]interface{}{
			"query":     queryStr,
			"variables": variables,
		}

		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading role",
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
				"Error reading role",
				"Could not create HTTP request: "+err.Error(),
			)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := d.client.Do(httpReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading role",
				"Could not execute HTTP request: "+err.Error(),
			)
			return
		}
		defer httpResp.Body.Close()

		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading role",
				"Could not read response body: "+err.Error(),
			)
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
			resp.Diagnostics.AddError(
				"Role Not Found",
				fmt.Sprintf("No role found with ID: %s", data.ID.ValueString()),
			)
			return
		}

		// Update state
		data.DisplayName = types.StringValue(gqlResp.Data.Role.DisplayName)

		// Convert permissions to list
		permissionsList, diags := types.ListValueFrom(ctx, types.StringType, gqlResp.Data.Role.ViewPermissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.ViewPermissions = permissionsList
	}

	// Set state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
