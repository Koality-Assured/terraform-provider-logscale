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
	_ resource.Resource              = &ingestTokenResource{}
	_ resource.ResourceWithConfigure = &ingestTokenResource{}
)

// NewIngestTokenResource is a helper function to simplify the provider implementation.
func NewIngestTokenResource() resource.Resource {
	return &ingestTokenResource{}
}

// ingestTokenResource is the resource implementation.
type ingestTokenResource struct {
	client *http.Client
	config *LogScaleConfig
}

// ingestTokenResourceModel maps the resource schema data.
type ingestTokenResourceModel struct {
	ID             types.String `tfsdk:"id"`
	RepositoryName types.String `tfsdk:"repository_name"`
	Name           types.String `tfsdk:"name"`
	Parser         types.String `tfsdk:"parser"`
	Token          types.String `tfsdk:"token"`
}

// Metadata returns the resource type name.
func (r *ingestTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingest_token"
}

// Schema defines the schema for the resource.
func (r *ingestTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale ingest token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Ingest token identifier (repository:name)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository_name": schema.StringAttribute{
				Description: "Repository name where the token will be created",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Ingest token name",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parser": schema.StringAttribute{
				Description: "Parser name to associate with this token",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				Description: "The generated ingest token value",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ingestTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ingestTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan ingestTokenResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set default parser if not specified
	parserName := "json"
	if !plan.Parser.IsNull() && plan.Parser.ValueString() != "" {
		parserName = plan.Parser.ValueString()
	}

	// GraphQL mutation
	mutationStr := `
		mutation AddIngestToken($input: AddIngestTokenV3Input!) {
			addIngestTokenV3(input: $input) {
				name
				token
				parser {
					id
					name
					displayName
				}
			}
		}
	`

	input := map[string]interface{}{
		"repositoryName": plan.RepositoryName.ValueString(),
		"name":           plan.Name.ValueString(),
		"parser":         parserName,
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
			"Error creating ingest token",
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
			"Error creating ingest token",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ingest token",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ingest token",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			AddIngestTokenV3 struct {
				Name   string `json:"name"`
				Token  string `json:"token"`
				Parser struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
				} `json:"parser"`
			} `json:"addIngestTokenV3"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error creating ingest token",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error creating ingest token",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", plan.RepositoryName.ValueString(), plan.Name.ValueString()))
	plan.Name = types.StringValue(gqlResp.Data.AddIngestTokenV3.Name)
	plan.Token = types.StringValue(gqlResp.Data.AddIngestTokenV3.Token)
	plan.Parser = types.StringValue(gqlResp.Data.AddIngestTokenV3.Parser.Name)

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ingestTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state ingestTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Note: LogScale doesn't have a direct API to read a single ingest token
	// In a real implementation, you might need to list all tokens and find this one
	// For now, we'll just keep the state as-is
	// The token value cannot be retrieved after creation, so we keep what we have

	// Set state (unchanged)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ingestTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Ingest tokens cannot be updated - they must be recreated
	// This should be handled by ForceNew in the schema
	resp.Diagnostics.AddError(
		"Ingest Token Update Not Supported",
		"Ingest tokens cannot be updated. They must be recreated.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ingestTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var state ingestTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// GraphQL mutation
	mutationStr := `
		mutation RemoveIngestToken($repositoryName: String!, $name: String!) {
			removeIngestToken(repositoryName: $repositoryName, name: $name) {
				result
			}
		}
	`

	variables := map[string]interface{}{
		"repositoryName": state.RepositoryName.ValueString(),
		"name":           state.Name.ValueString(),
	}

	payload := map[string]interface{}{
		"query":     mutationStr,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ingest token",
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
			"Error deleting ingest token",
			"Could not create HTTP request: "+err.Error(),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ingest token",
			"Could not execute HTTP request: "+err.Error(),
		)
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ingest token",
			"Could not read response body: "+err.Error(),
		)
		return
	}

	var gqlResp struct {
		Data struct {
			RemoveIngestToken struct {
				Result string `json:"result"`
			} `json:"removeIngestToken"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ingest token",
			fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body),
		)
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError(
			"Error deleting ingest token",
			fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message),
		)
		return
	}

	// Resource is deleted, Terraform will remove from state automatically
}
