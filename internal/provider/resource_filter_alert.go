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
	_ resource.Resource                = &filterAlertResource{}
	_ resource.ResourceWithConfigure   = &filterAlertResource{}
	_ resource.ResourceWithImportState = &filterAlertResource{}
)

func NewFilterAlertResource() resource.Resource {
	return &filterAlertResource{}
}

type filterAlertResource struct {
	client *http.Client
	config *LogScaleConfig
}

type filterAlertResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ViewName            types.String `tfsdk:"view_name"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	QueryString         types.String `tfsdk:"query_string"`
	ActionIDsOrNames    types.Set    `tfsdk:"action_ids_or_names"`
	QueryOwnershipType  types.String `tfsdk:"query_ownership_type"`
	RunAsUserID         types.String `tfsdk:"run_as_user_id"`
	ThrottleTimeSeconds types.Int64  `tfsdk:"throttle_time_seconds"`
	ThrottleFields      types.Set    `tfsdk:"throttle_fields"`
	Labels              types.Set    `tfsdk:"labels"`
	Resource            types.String `tfsdk:"resource"`
	YAMLTemplate        types.String `tfsdk:"yaml_template"`
}

type filterAlertReadResult struct {
	TypeName            string                  `json:"__typename"`
	ID                  string                  `json:"id"`
	Name                string                  `json:"name"`
	Description         string                  `json:"description"`
	Enabled             bool                    `json:"enabled"`
	QueryString         string                  `json:"queryString"`
	Labels              []string                `json:"labels"`
	Resource            string                  `json:"resource"`
	YAMLTemplate        string                  `json:"yamlTemplate"`
	ThrottleTimeSeconds *int64                  `json:"throttleTimeSeconds"`
	ThrottleFields      []string                `json:"throttleFields"`
	QueryOwnership      filterAlertOwnershipAPI `json:"queryOwnership"`
	Actions             []filterAlertActionAPI  `json:"actions"`
}

type filterAlertOwnershipAPI struct {
	TypeName string `json:"__typename"`
	ID       string `json:"id"`
}

type filterAlertActionAPI struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (r *filterAlertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_filter_alert"
}

func (r *filterAlertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale filter alert.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Filter alert ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"view_name": schema.StringAttribute{
				Description: "Repository or view that owns the alert.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter alert name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional filter alert description.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the alert is enabled.",
				Required:    true,
			},
			"query_string": schema.StringAttribute{
				Description: "The LogScale query executed by the alert.",
				Required:    true,
			},
			"action_ids_or_names": schema.SetAttribute{
				Description: "Action references used by the alert.",
				Required:    true,
				ElementType: types.StringType,
			},
			"query_ownership_type": schema.StringAttribute{
				Description: "Ownership type for the alert query, such as Organization or User.",
				Required:    true,
			},
			"run_as_user_id": schema.StringAttribute{
				Description: "User ID used when query_ownership_type is User.",
				Optional:    true,
			},
			"throttle_time_seconds": schema.Int64Attribute{
				Description: "Throttle window in seconds.",
				Optional:    true,
			},
			"throttle_fields": schema.SetAttribute{
				Description: "Fields used to scope throttling.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the alert.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"resource": schema.StringAttribute{
				Description: "Provider-facing resource identifier returned by LogScale.",
				Computed:    true,
			},
			"yaml_template": schema.StringAttribute{
				Description: "YAML template returned by LogScale.",
				Computed:    true,
			},
		},
	}
}

func (r *filterAlertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *filterAlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan filterAlertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, ok := r.filterAlertInputFromModel(ctx, plan, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		CreateFilterAlert filterAlertReadResult `json:"createFilterAlert"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, filterAlertSelectionMutation("createFilterAlert", "CreateFilterAlert"), map[string]interface{}{"input": input}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating filter alert", err.Error())
		return
	}

	r.applyFilterAlertReadResultToState(ctx, &plan, data.CreateFilterAlert, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *filterAlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state filterAlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, found, err := r.readFilterAlert(ctx, state.ViewName.ValueString(), state.ID.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading filter alert", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	r.applyFilterAlertReadResultToState(ctx, &state, result, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *filterAlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan filterAlertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state filterAlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, ok := r.filterAlertInputFromModel(ctx, plan, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}
	input["id"] = state.ID.ValueString()
	input["viewName"] = state.ViewName.ValueString()

	var data struct {
		UpdateFilterAlertV2 filterAlertReadResult `json:"updateFilterAlertV2"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, filterAlertSelectionMutation("updateFilterAlertV2", "UpdateFilterAlertV2"), map[string]interface{}{"input": input}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error updating filter alert", err.Error())
		return
	}

	plan.ViewName = state.ViewName
	r.applyFilterAlertReadResultToState(ctx, &plan, data.UpdateFilterAlertV2, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *filterAlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state filterAlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		DeleteFilterAlertV2 *bool `json:"deleteFilterAlertV2"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation DeleteFilterAlert($input: DeleteFilterAlertV2!) {
			deleteFilterAlertV2(input: $input)
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"viewName": state.ViewName.ValueString(),
			"id":       state.ID.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting filter alert", err.Error())
	}
}

func (r *filterAlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier in the format view_name:filter_alert_id.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("view_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *filterAlertResource) filterAlertInputFromModel(ctx context.Context, model filterAlertResourceModel, diags *diag.Diagnostics) (map[string]interface{}, bool) {
	queryOwnershipType := model.QueryOwnershipType.ValueString()
	if queryOwnershipType == "" {
		diags.AddError("Invalid query_ownership_type", "query_ownership_type must not be empty.")
		return nil, false
	}

	actionRefs := stringsFromSet(ctx, model.ActionIDsOrNames, diags)
	throttleFields := stringsFromSet(ctx, model.ThrottleFields, diags)
	labels := stringsFromSet(ctx, model.Labels, diags)
	if diags.HasError() {
		return nil, false
	}

	input := map[string]interface{}{
		"viewName":           model.ViewName.ValueString(),
		"name":               model.Name.ValueString(),
		"description":        model.Description.ValueString(),
		"enabled":            model.Enabled.ValueBool(),
		"queryString":        model.QueryString.ValueString(),
		"actionIdsOrNames":   actionRefs,
		"queryOwnershipType": queryOwnershipType,
		"labels":             labels,
	}

	if !model.RunAsUserID.IsNull() && model.RunAsUserID.ValueString() != "" {
		input["runAsUserId"] = model.RunAsUserID.ValueString()
	}
	if !model.ThrottleTimeSeconds.IsNull() {
		input["throttleTimeSeconds"] = model.ThrottleTimeSeconds.ValueInt64()
	}
	if len(throttleFields) > 0 {
		input["throttleFields"] = throttleFields
	}

	return input, true
}

func filterAlertSelectionMutation(fieldName string, inputType string) string {
	return `
		mutation FilterAlertMutation($input: ` + inputType + `!) {
			` + fieldName + `(input: $input) {
				__typename
				id
				name
				description
				enabled
				queryString
				labels
				resource
				yamlTemplate
				throttleTimeSeconds
				throttleFields
				queryOwnership {
					__typename
					id
				}
				actions {
					id
					name
				}
			}
		}
	`
}

func (r *filterAlertResource) readFilterAlert(ctx context.Context, viewName string, id string, name string) (filterAlertReadResult, bool, error) {
	var data struct {
		SearchDomain *struct {
			TypeName     string                  `json:"__typename"`
			FilterAlerts []filterAlertReadResult `json:"filterAlerts"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		query GetFilterAlerts($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository {
					filterAlerts {
						__typename
						id
						name
						description
						enabled
						queryString
						labels
						resource
						yamlTemplate
						throttleTimeSeconds
						throttleFields
						queryOwnership {
							__typename
							id
						}
						actions {
							id
							name
						}
					}
				}
				... on View {
					filterAlerts {
						__typename
						id
						name
						description
						enabled
						queryString
						labels
						resource
						yamlTemplate
						throttleTimeSeconds
						throttleFields
						queryOwnership {
							__typename
							id
						}
						actions {
							id
							name
						}
					}
				}
			}
		}
	`, map[string]interface{}{"viewName": viewName}, &data)
	if err != nil {
		return filterAlertReadResult{}, false, err
	}

	if data.SearchDomain == nil {
		return filterAlertReadResult{}, false, nil
	}

	if id != "" {
		for _, alert := range data.SearchDomain.FilterAlerts {
			if alert.ID == id {
				return alert, true, nil
			}
		}
	}

	if name != "" {
		for _, alert := range data.SearchDomain.FilterAlerts {
			if alert.Name == name {
				return alert, true, nil
			}
		}
	}

	return filterAlertReadResult{}, false, nil
}

func (r *filterAlertResource) applyFilterAlertReadResultToState(ctx context.Context, state *filterAlertResourceModel, result filterAlertReadResult, diags *diag.Diagnostics) {
	actionRefs := make([]assetReference, 0, len(result.Actions))
	for _, action := range result.Actions {
		actionRefs = append(actionRefs, assetReference{ID: action.ID, Name: action.Name})
	}

	preservedActionRefs, actionDiags := preserveCurrentReferencesIfResolvable(ctx, state.ActionIDsOrNames, actionRefs)
	diags.Append(actionDiags...)

	throttleFields, throttleDiags := normalizeOptionalStringSetFromAPI(ctx, state.ThrottleFields, result.ThrottleFields)
	diags.Append(throttleDiags...)

	labels, labelDiags := normalizeOptionalStringSetFromAPI(ctx, state.Labels, result.Labels)
	diags.Append(labelDiags...)
	if diags.HasError() {
		return
	}

	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Description = normalizeOptionalStringFromAPI(state.Description, result.Description)
	state.Enabled = types.BoolValue(result.Enabled)
	state.QueryString = types.StringValue(result.QueryString)
	state.ActionIDsOrNames = preservedActionRefs
	state.QueryOwnershipType = types.StringValue(strings.TrimSuffix(result.QueryOwnership.TypeName, "Ownership"))
	if result.QueryOwnership.TypeName == "UserOwnership" {
		state.RunAsUserID = types.StringValue(result.QueryOwnership.ID)
	} else if state.RunAsUserID.IsNull() {
		state.RunAsUserID = types.StringNull()
	} else {
		state.RunAsUserID = types.StringNull()
	}
	if result.ThrottleTimeSeconds == nil {
		state.ThrottleTimeSeconds = types.Int64Null()
	} else {
		state.ThrottleTimeSeconds = types.Int64Value(*result.ThrottleTimeSeconds)
	}
	state.ThrottleFields = throttleFields
	state.Labels = labels
	state.Resource = types.StringValue(result.Resource)
	state.YAMLTemplate = types.StringValue(result.YAMLTemplate)
}
