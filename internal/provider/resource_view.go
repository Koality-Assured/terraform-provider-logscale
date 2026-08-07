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
	_ resource.Resource                = &viewResource{}
	_ resource.ResourceWithConfigure   = &viewResource{}
	_ resource.ResourceWithImportState = &viewResource{}
)

func NewViewResource() resource.Resource {
	return &viewResource{}
}

type viewResource struct {
	client *http.Client
	config *LogScaleConfig
}

type viewResourceModel struct {
	ID          types.String           `tfsdk:"id"`
	Name        types.String           `tfsdk:"name"`
	Description types.String           `tfsdk:"description"`
	Connections []viewConnectionModel  `tfsdk:"repository_connection"`
}

type viewConnectionModel struct {
	RepositoryName types.String `tfsdk:"repository_name"`
	Filter         types.String `tfsdk:"filter"`
}

func (r *viewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view"
}

func (r *viewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale view.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "View ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "View name",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "View description",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"repository_connection": schema.ListNestedBlock{
				Description: "Repository connections that back the view.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"repository_name": schema.StringAttribute{
							Description: "Repository name connected to the view.",
							Required:    true,
						},
						"filter": schema.StringAttribute{
							Description: "Filter applied to the repository connection.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *viewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.APIToken})
	r.client = oauth2.NewClient(context.Background(), src)
}

func (r *viewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan viewResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	description := ""
	if !plan.Description.IsNull() {
		description = plan.Description.ValueString()
	}

	mutationStr := `
		mutation CreateView($viewName: String!, $viewDescription: String, $viewConnections: [ViewConnectionInput!]) {
			createView(name: $viewName, description: $viewDescription, connections: $viewConnections) {
				id
				name
				description
				connections {
					filter
					repository {
						name
					}
				}
			}
		}
	`

	body, err := r.doGraphQLRequest(ctx, mutationStr, map[string]interface{}{
		"viewName":        plan.Name.ValueString(),
		"viewDescription": description,
		"viewConnections": viewConnectionsToAPI(plan.Connections),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating view", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			CreateView viewReadResult `json:"createView"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError("Error creating view", fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body))
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError("Error creating view", fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message))
		return
	}

	applyViewReadResultToState(&plan, gqlResp.Data.CreateView)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *viewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state viewResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	queryStr := `
		query GetView($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				id
				name
				description
				... on View {
					connections {
						filter
						repository {
							name
						}
					}
				}
			}
		}
	`

	body, err := r.doGraphQLRequest(ctx, queryStr, map[string]interface{}{
		"viewName": state.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading view", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			SearchDomain *struct {
				TypeName    string              `json:"__typename"`
				ID          string              `json:"id"`
				Name        string              `json:"name"`
				Description string              `json:"description"`
				Connections []viewConnectionAPI `json:"connections"`
			} `json:"searchDomain"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError("Error reading view", fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body))
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError("Error reading view", fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message))
		return
	}

	if gqlResp.Data.SearchDomain == nil || gqlResp.Data.SearchDomain.TypeName != "View" {
		resp.State.RemoveResource(ctx)
		return
	}

	applyViewReadResultToState(&state, viewReadResult{
		ID:          gqlResp.Data.SearchDomain.ID,
		Name:        gqlResp.Data.SearchDomain.Name,
		Description: gqlResp.Data.SearchDomain.Description,
		Connections: gqlResp.Data.SearchDomain.Connections,
	})

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *viewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan viewResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation UpdateView($viewName: String!, $viewConnections: [ViewConnectionInput!]!) {
			updateView(viewName: $viewName, connections: $viewConnections) {
				id
				name
				description
				connections {
					filter
					repository {
						name
					}
				}
			}
		}
	`

	body, err := r.doGraphQLRequest(ctx, mutationStr, map[string]interface{}{
		"viewName":        plan.Name.ValueString(),
		"viewConnections": viewConnectionsToAPI(plan.Connections),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating view", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			UpdateView viewReadResult `json:"updateView"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError("Error updating view", fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body))
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError("Error updating view", fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message))
		return
	}

	applyViewReadResultToState(&plan, gqlResp.Data.UpdateView)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *viewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state viewResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation DeleteView($input: DeleteSearchDomainByIdInput!) {
			deleteSearchDomainById(input: $input)
		}
	`

	body, err := r.doGraphQLRequest(ctx, mutationStr, map[string]interface{}{
		"input": map[string]interface{}{
			"id": state.ID.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting view", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			DeleteSearchDomainById *bool `json:"deleteSearchDomainById"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError("Error deleting view", fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body))
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError("Error deleting view", fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message))
		return
	}
}

func (r *viewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier to be the view name.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

type viewReadResult struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Connections []viewConnectionAPI `json:"connections"`
}

type viewConnectionAPI struct {
	Filter     string `json:"filter"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
}

func (r *viewResource) doGraphQLRequest(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
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

func viewConnectionsToAPI(connections []viewConnectionModel) []map[string]interface{} {
	apiConnections := make([]map[string]interface{}, 0, len(connections))
	for _, connection := range connections {
		apiConnections = append(apiConnections, map[string]interface{}{
			"repositoryName": connection.RepositoryName.ValueString(),
			"filter":         connection.Filter.ValueString(),
		})
	}
	return apiConnections
}

func applyViewReadResultToState(state *viewResourceModel, result viewReadResult) {
	var connections []viewConnectionModel
	if len(result.Connections) == 0 && state.Connections == nil {
		connections = nil
	} else {
		connections = make([]viewConnectionModel, 0, len(result.Connections))
	}

	for _, connection := range result.Connections {
		connections = append(connections, viewConnectionModel{
			RepositoryName: types.StringValue(connection.Repository.Name),
			Filter:         types.StringValue(connection.Filter),
		})
	}

	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Description = normalizeOptionalStringFromAPI(state.Description, result.Description)
	state.Connections = connections
}
