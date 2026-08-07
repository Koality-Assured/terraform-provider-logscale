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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &repositoryResource{}
	_ resource.ResourceWithConfigure = &repositoryResource{}
)

// NewRepositoryResource is a helper function to simplify the provider implementation.
func NewRepositoryResource() resource.Resource {
	return &repositoryResource{}
}

// repositoryResource is the resource implementation.
type repositoryResource struct {
	client *http.Client
	config *LogScaleConfig
}

// repositoryResourceModel maps the resource schema data.
type repositoryResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Type          types.String `tfsdk:"type"`
	RetentionDays types.Int64  `tfsdk:"retention_days"`
}

// LogScaleConfig holds the provider configuration
type LogScaleConfig struct {
	Endpoint string
	APIToken string
}

// Metadata returns the resource type name.
func (r *repositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

// Schema defines the schema for the resource.
func (r *repositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale repository.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Repository ID",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Repository name",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Repository description",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "Repository type (PERSONAL, TRIAL, DEFAULT, SYSTEM, MANAGED). Defaults to DEFAULT.",
				Optional:    true,
				Computed:    true,
			},
			"retention_days": schema.Int64Attribute{
				Description: "Data retention in days. Defaults to 365.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *repositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *repositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan repositoryResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults - matching your working repoCreate.go
	repoType := "DEFAULT"
	if !plan.Type.IsNull() && plan.Type.ValueString() != "" {
		repoType = plan.Type.ValueString()
	}

	repoDescription := ""
	if !plan.Description.IsNull() {
		repoDescription = plan.Description.ValueString()
	}

	retentionDays := int64(365)
	if !plan.RetentionDays.IsNull() {
		retentionDays = plan.RetentionDays.ValueInt64()
	}

	retentionMS := retentionDays * 86400000

	// GraphQL mutation - EXACTLY as in your working repoCreate.go
	mutationQuery := `
		mutation CreateRepo(
			$repoName: String!, 
			$repoDescription: String!, 
			$repoType: RepositoryType!, 
			$repoRetentionMS: Long!
		) {
			createRepository(
				name: $repoName, 
				description: $repoDescription, 
				type: $repoType, 
				retentionInMillis: $repoRetentionMS
			) {
				repository {
					id
					name
				}
			}
		}
	`

	variables := map[string]interface{}{
		"repoName":        plan.Name.ValueString(),
		"repoDescription": repoDescription,
		"repoType":        repoType,
		"repoRetentionMS": retentionMS,
	}

	payload := map[string]interface{}{
		"query":     mutationQuery,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating repository",
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
			"Error creating repository",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating repository",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating repository",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	// Response structure - matching your working code
	var gqlResp struct {
		Data struct {
			CreateRepository struct {
				Repository struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"repository"`
			} `json:"createRepository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error creating repository",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error creating repository",
			fmt.Sprintf("GraphQL error: %s\nFull response: %s", gqlResp.Errors[0].Message, string(body)),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(gqlResp.Data.CreateRepository.Repository.ID)
	plan.Name = types.StringValue(gqlResp.Data.CreateRepository.Repository.Name)
	plan.Description = normalizeOptionalStringFromAPI(plan.Description, repoDescription)
	plan.Type = types.StringValue(repoType)
	plan.RetentionDays = types.Int64Value(retentionDays)

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *repositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state repositoryResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// GraphQL query - matching your working repoGet.go
	queryStr := `
		query GetRepository($repoName: String!) {
			repository(name: $repoName) {
				id
				name
				type
				description
			}
		}
	`

	variables := map[string]interface{}{
		"repoName": state.Name.ValueString(),
	}

	payload := map[string]interface{}{
		"query":     queryStr,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading repository",
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
			"Error reading repository",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading repository",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading repository",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			Repository *struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Type        string `json:"type"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error reading repository",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error reading repository",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}

	if gqlResp.Data.Repository == nil {
		// Repository not found - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state
	state.ID = types.StringValue(gqlResp.Data.Repository.ID)
	state.Name = types.StringValue(gqlResp.Data.Repository.Name)
	state.Description = normalizeOptionalStringFromAPI(state.Description, gqlResp.Data.Repository.Description)
	state.Type = types.StringValue(gqlResp.Data.Repository.Type)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *repositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// LogScale repositories don't support updates via API
	// For now, we'll just read the current state
	resp.Diagnostics.AddWarning(
		"Repository Update Not Supported",
		"LogScale repositories cannot be updated after creation. No changes will be made.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *repositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var state repositoryResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// As noted in your repoDelete.go - this endpoint doesn't exist
	resp.Diagnostics.AddWarning(
		"Repository Deletion Not Supported",
		fmt.Sprintf("LogScale repositories cannot be deleted via API. Repository '%s' (ID: %s) will be removed from Terraform state, but will still exist in LogScale. Manual deletion is required.", state.Name.ValueString(), state.ID.ValueString()),
	)

	// Resource will be removed from state automatically by Terraform
}
