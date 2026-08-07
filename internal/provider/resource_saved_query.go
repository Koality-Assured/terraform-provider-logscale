package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &savedQueryResource{}
	_ resource.ResourceWithConfigure   = &savedQueryResource{}
	_ resource.ResourceWithImportState = &savedQueryResource{}
)

func NewSavedQueryResource() resource.Resource {
	return &savedQueryResource{}
}

type savedQueryResource struct {
	client *http.Client
	config *LogScaleConfig
}

type savedQueryResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ViewName     types.String `tfsdk:"view_name"`
	Name         types.String `tfsdk:"name"`
	DisplayName  types.String `tfsdk:"display_name"`
	Description  types.String `tfsdk:"description"`
	YAMLTemplate types.String `tfsdk:"yaml_template"`
	Labels       types.Set    `tfsdk:"labels"`
	QueryString  types.String `tfsdk:"query_string"`
	IsLive       types.Bool   `tfsdk:"is_live"`
	QueryStart   types.String `tfsdk:"query_start"`
	QueryEnd     types.String `tfsdk:"query_end"`
	IsStarred    types.Bool   `tfsdk:"is_starred"`
	Resource     types.String `tfsdk:"resource"`
}

type humioQueryReadResult struct {
	QueryString string `json:"queryString"`
	IsLive      bool   `json:"isLive"`
	Start       string `json:"start"`
	End         string `json:"end"`
}

type savedQueryReadResult struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	DisplayName  string               `json:"displayName"`
	Description  string               `json:"description"`
	Labels       []string             `json:"labels"`
	Resource     string               `json:"resource"`
	YAMLTemplate string               `json:"yamlTemplate"`
	IsStarred    bool                 `json:"isStarred"`
	Query        humioQueryReadResult `json:"query"`
}

func (r *savedQueryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_query"
}

func (r *savedQueryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale saved query through the YAML-template lifecycle.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Saved query ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"view_name": schema.StringAttribute{
				Description: "Repository or view that owns the saved query.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Optional saved query name override. If omitted, LogScale uses the YAML template name.",
				Optional:    true,
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "Saved query display name returned by LogScale.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Saved query description returned by LogScale.",
				Computed:    true,
			},
			"yaml_template": schema.StringAttribute{
				Description: "Saved query YAML template used for create and update.",
				Required:    true,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the saved query.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"query_string": schema.StringAttribute{
				Description: "Query string returned by LogScale.",
				Computed:    true,
			},
			"is_live": schema.BoolAttribute{
				Description: "Whether the saved query uses live streaming data.",
				Computed:    true,
			},
			"query_start": schema.StringAttribute{
				Description: "Relative query start returned by LogScale.",
				Computed:    true,
			},
			"query_end": schema.StringAttribute{
				Description: "Relative query end returned by LogScale.",
				Computed:    true,
			},
			"is_starred": schema.BoolAttribute{
				Description: "Whether the saved query is starred in LogScale.",
				Computed:    true,
			},
			"resource": schema.StringAttribute{
				Description: "Provider-facing resource identifier returned by LogScale.",
				Computed:    true,
			},
		},
	}
}

func (r *savedQueryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}

	r.config = config
	r.client = newConfiguredHTTPClient(config)
}

func (r *savedQueryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan savedQueryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if strings.TrimSpace(plan.ViewName.ValueString()) == "" {
		resp.Diagnostics.AddError("Invalid view_name", "view_name must not be empty.")
		return
	}
	if strings.TrimSpace(plan.YAMLTemplate.ValueString()) == "" {
		resp.Diagnostics.AddError("Invalid yaml_template", "yaml_template must not be empty.")
		return
	}

	input := map[string]interface{}{
		"viewName":     plan.ViewName.ValueString(),
		"yamlTemplate": plan.YAMLTemplate.ValueString(),
	}
	if !plan.Name.IsNull() && strings.TrimSpace(plan.Name.ValueString()) != "" {
		input["name"] = plan.Name.ValueString()
	}

	var data struct {
		CreateSavedQueryFromTemplate savedQueryReadResult `json:"createSavedQueryFromTemplate"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation CreateSavedQueryFromTemplate($input: CreateSavedQueryFromTemplateInput!) {
			createSavedQueryFromTemplate(input: $input) {
				id
				name
				displayName
				description
				labels
				resource
				yamlTemplate
				isStarred
				query {
					queryString
					isLive
					start
					end
				}
			}
		}
	`, map[string]interface{}{"input": input}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating saved query", err.Error())
		return
	}

	desiredLabels := stringsFromSet(ctx, plan.Labels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := syncLabelsByID(ctx, r.client, r.config.Endpoint, plan.ViewName.ValueString(), data.CreateSavedQueryFromTemplate.ID, nil, desiredLabels, "addSavedQueryLabels", "AddSavedQueryLabels", "removeSavedQueryLabels", "RemoveSavedQueryLabels"); err != nil {
		resp.Diagnostics.AddError("Error syncing saved query labels", err.Error())
		return
	}

	readResult, found, err := r.readSavedQuery(ctx, plan.ViewName.ValueString(), data.CreateSavedQueryFromTemplate.ID, "")
	if err != nil {
		resp.Diagnostics.AddError("Error reading saved query", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Saved query disappeared after create", fmt.Sprintf("No saved query with ID %q was found after creation.", data.CreateSavedQueryFromTemplate.ID))
		return
	}

	applySavedQueryReadResultToState(ctx, &plan, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *savedQueryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state savedQueryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readResult, found, err := r.readSavedQuery(ctx, state.ViewName.ValueString(), state.ID.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading saved query", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	applySavedQueryReadResultToState(ctx, &state, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *savedQueryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan savedQueryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state savedQueryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if strings.TrimSpace(plan.YAMLTemplate.ValueString()) == "" {
		resp.Diagnostics.AddError("Invalid yaml_template", "yaml_template must not be empty.")
		return
	}

	var data struct {
		UpdateSavedQueryFromTemplate savedQueryReadResult `json:"updateSavedQueryFromTemplate"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation UpdateSavedQueryFromTemplate($input: UpdateSavedQueryFromTemplateInput!) {
			updateSavedQueryFromTemplate(input: $input) {
				id
				name
				displayName
				description
				labels
				resource
				yamlTemplate
				isStarred
				query {
					queryString
					isLive
					start
					end
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"id":           state.ID.ValueString(),
			"viewName":     state.ViewName.ValueString(),
			"yamlTemplate": plan.YAMLTemplate.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error updating saved query", err.Error())
		return
	}

	currentLabels := stringsFromSet(ctx, state.Labels, &resp.Diagnostics)
	desiredLabels := stringsFromSet(ctx, plan.Labels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := syncLabelsByID(ctx, r.client, r.config.Endpoint, state.ViewName.ValueString(), state.ID.ValueString(), currentLabels, desiredLabels, "addSavedQueryLabels", "AddSavedQueryLabels", "removeSavedQueryLabels", "RemoveSavedQueryLabels"); err != nil {
		resp.Diagnostics.AddError("Error syncing saved query labels", err.Error())
		return
	}

	readResult, found, err := r.readSavedQuery(ctx, state.ViewName.ValueString(), state.ID.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError("Error reading saved query", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Saved query disappeared after update", fmt.Sprintf("No saved query with ID %q was found after update.", state.ID.ValueString()))
		return
	}

	plan.ViewName = state.ViewName
	applySavedQueryReadResultToState(ctx, &plan, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *savedQueryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state savedQueryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		DeleteSavedQueryV2 *bool `json:"deleteSavedQueryV2"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation DeleteSavedQuery($input: DeleteSavedQuery!) {
			deleteSavedQueryV2(input: $input)
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"viewName": state.ViewName.ValueString(),
			"id":       state.ID.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting saved query", err.Error())
	}
}

func (r *savedQueryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier in the format view_name:saved_query_id.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("view_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *savedQueryResource) readSavedQuery(ctx context.Context, viewName string, id string, name string) (savedQueryReadResult, bool, error) {
	var data struct {
		SearchDomain *struct {
			TypeName     string                 `json:"__typename"`
			SavedQueries []savedQueryReadResult `json:"savedQueries"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		query GetSavedQueries($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository {
					savedQueries {
						id
						name
						displayName
						description
						labels
						resource
						yamlTemplate
						isStarred
						query {
							queryString
							isLive
							start
							end
						}
					}
				}
				... on View {
					savedQueries {
						id
						name
						displayName
						description
						labels
						resource
						yamlTemplate
						isStarred
						query {
							queryString
							isLive
							start
							end
						}
					}
				}
			}
		}
	`, map[string]interface{}{"viewName": viewName}, &data)
	if err != nil {
		return savedQueryReadResult{}, false, err
	}

	if data.SearchDomain == nil {
		return savedQueryReadResult{}, false, nil
	}

	if id != "" {
		for _, item := range data.SearchDomain.SavedQueries {
			if item.ID == id {
				return item, true, nil
			}
		}
	}
	if name != "" {
		for _, item := range data.SearchDomain.SavedQueries {
			if item.Name == name {
				return item, true, nil
			}
		}
	}

	return savedQueryReadResult{}, false, nil
}

func applySavedQueryReadResultToState(ctx context.Context, state *savedQueryResourceModel, result savedQueryReadResult, diags *diag.Diagnostics) {
	labels, labelDiags := normalizeOptionalStringSetFromAPI(ctx, state.Labels, result.Labels)
	diags.Append(labelDiags...)
	if diags.HasError() {
		return
	}

	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.DisplayName = types.StringValue(result.DisplayName)
	state.Description = normalizeOptionalStringFromAPI(state.Description, result.Description)
	state.YAMLTemplate = normalizeYAMLTemplateFromAPI(state.YAMLTemplate, result.YAMLTemplate)
	state.Labels = labels
	state.QueryString = types.StringValue(result.Query.QueryString)
	state.IsLive = types.BoolValue(result.Query.IsLive)
	state.QueryStart = normalizeOptionalStringFromAPI(state.QueryStart, result.Query.Start)
	state.QueryEnd = normalizeOptionalStringFromAPI(state.QueryEnd, result.Query.End)
	state.IsStarred = types.BoolValue(result.IsStarred)
	state.Resource = types.StringValue(result.Resource)
}
