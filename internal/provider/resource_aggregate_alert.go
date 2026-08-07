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
	_ resource.Resource                = &aggregateAlertResource{}
	_ resource.ResourceWithConfigure   = &aggregateAlertResource{}
	_ resource.ResourceWithImportState = &aggregateAlertResource{}
)

func NewAggregateAlertResource() resource.Resource {
	return &aggregateAlertResource{}
}

type aggregateAlertResource struct {
	client *http.Client
	config *LogScaleConfig
}

type aggregateAlertResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ViewName              types.String `tfsdk:"view_name"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	QueryString           types.String `tfsdk:"query_string"`
	ActionIDsOrNames      types.Set    `tfsdk:"action_ids_or_names"`
	QueryOwnershipType    types.String `tfsdk:"query_ownership_type"`
	RunAsUserID           types.String `tfsdk:"run_as_user_id"`
	QueryTimestampType    types.String `tfsdk:"query_timestamp_type"`
	SearchIntervalSeconds types.Int64  `tfsdk:"search_interval_seconds"`
	ThrottleTimeSeconds   types.Int64  `tfsdk:"throttle_time_seconds"`
	ThrottleFields        types.Set    `tfsdk:"throttle_fields"`
	TriggerMode           types.String `tfsdk:"trigger_mode"`
	Labels                types.Set    `tfsdk:"labels"`
	Resource              types.String `tfsdk:"resource"`
	YAMLTemplate          types.String `tfsdk:"yaml_template"`
	LastError             types.String `tfsdk:"last_error"`
	LastWarnings          types.Set    `tfsdk:"last_warnings"`
	LastSuccessfulPoll    types.Int64  `tfsdk:"last_successful_poll"`
	LastTriggered         types.Int64  `tfsdk:"last_triggered"`
}

type aggregateAlertReadResult struct {
	TypeName              string                  `json:"__typename"`
	ID                    string                  `json:"id"`
	Name                  string                  `json:"name"`
	Description           string                  `json:"description"`
	Enabled               bool                    `json:"enabled"`
	QueryString           string                  `json:"queryString"`
	Labels                []string                `json:"labels"`
	Resource              string                  `json:"resource"`
	YAMLTemplate          string                  `json:"yamlTemplate"`
	QueryTimestampType    string                  `json:"queryTimestampType"`
	SearchIntervalSeconds int64                   `json:"searchIntervalSeconds"`
	ThrottleTimeSeconds   *int64                  `json:"throttleTimeSeconds"`
	ThrottleFields        []string                `json:"throttleFields"`
	TriggerMode           string                  `json:"triggerMode"`
	LastError             string                  `json:"lastError"`
	LastWarnings          []string                `json:"lastWarnings"`
	LastSuccessfulPoll    *int64                  `json:"lastSuccessfulPoll"`
	LastTriggered         *int64                  `json:"lastTriggered"`
	QueryOwnership        filterAlertOwnershipAPI `json:"queryOwnership"`
	Actions               []filterAlertActionAPI  `json:"actions"`
}

func (r *aggregateAlertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aggregate_alert"
}

func (r *aggregateAlertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale aggregate alert.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Aggregate alert ID.",
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
				Description: "Aggregate alert name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional aggregate alert description.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the alert is enabled.",
				Required:    true,
			},
			"query_string": schema.StringAttribute{
				Description: "The LogScale query executed by the aggregate alert.",
				Required:    true,
			},
			"action_ids_or_names": schema.SetAttribute{
				Description: "Action references used by the aggregate alert.",
				Required:    true,
				ElementType: types.StringType,
			},
			"query_ownership_type": schema.StringAttribute{
				Description: "Ownership type for the aggregate alert query, such as Organization or User.",
				Required:    true,
			},
			"run_as_user_id": schema.StringAttribute{
				Description: "User ID used when query_ownership_type is User.",
				Optional:    true,
			},
			"query_timestamp_type": schema.StringAttribute{
				Description: "Timestamp mode for the alert query, such as EventTimestamp or IngestTimestamp.",
				Required:    true,
			},
			"search_interval_seconds": schema.Int64Attribute{
				Description: "Search interval in seconds.",
				Required:    true,
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
			"trigger_mode": schema.StringAttribute{
				Description: "Trigger mode used by the aggregate alert, such as ImmediateMode or CompleteMode.",
				Optional:    true,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the aggregate alert.",
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
			"last_error": schema.StringAttribute{
				Description: "Last error returned by LogScale for the aggregate alert.",
				Computed:    true,
			},
			"last_warnings": schema.SetAttribute{
				Description: "Last warnings returned by LogScale for the aggregate alert.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"last_successful_poll": schema.Int64Attribute{
				Description: "Unix timestamp of the last successful poll.",
				Computed:    true,
			},
			"last_triggered": schema.Int64Attribute{
				Description: "Unix timestamp of the last trigger event.",
				Computed:    true,
			},
		},
	}
}

func (r *aggregateAlertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *aggregateAlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan aggregateAlertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, ok := aggregateAlertInputFromModel(ctx, plan, false, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		CreateAggregateAlert aggregateAlertReadResult `json:"createAggregateAlert"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, aggregateAlertSelectionMutation("createAggregateAlert", "CreateAggregateAlert"), map[string]interface{}{"input": input}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating aggregate alert", err.Error())
		return
	}

	applyAggregateAlertReadResultToState(ctx, &plan, data.CreateAggregateAlert, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *aggregateAlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state aggregateAlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, found, err := r.readAggregateAlert(ctx, state.ViewName.ValueString(), state.ID.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading aggregate alert", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	applyAggregateAlertReadResultToState(ctx, &state, result, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *aggregateAlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan aggregateAlertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state aggregateAlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, ok := aggregateAlertInputFromModel(ctx, plan, true, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}
	input["id"] = state.ID.ValueString()
	input["viewName"] = state.ViewName.ValueString()

	var data struct {
		UpdateAggregateAlertV2 aggregateAlertReadResult `json:"updateAggregateAlertV2"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, aggregateAlertSelectionMutation("updateAggregateAlertV2", "UpdateAggregateAlertV2"), map[string]interface{}{"input": input}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error updating aggregate alert", err.Error())
		return
	}

	plan.ViewName = state.ViewName
	applyAggregateAlertReadResultToState(ctx, &plan, data.UpdateAggregateAlertV2, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *aggregateAlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state aggregateAlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		DeleteAggregateAlertV2 *bool `json:"deleteAggregateAlertV2"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation DeleteAggregateAlert($input: DeleteAggregateAlert!) {
			deleteAggregateAlertV2(input: $input)
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"viewName": state.ViewName.ValueString(),
			"id":       state.ID.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting aggregate alert", err.Error())
	}
}

func (r *aggregateAlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier in the format view_name:aggregate_alert_id.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("view_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func aggregateAlertInputFromModel(ctx context.Context, model aggregateAlertResourceModel, update bool, diags *diag.Diagnostics) (map[string]interface{}, bool) {
	if strings.TrimSpace(model.ViewName.ValueString()) == "" {
		diags.AddError("Invalid view_name", "view_name must not be empty.")
	}
	if strings.TrimSpace(model.Name.ValueString()) == "" {
		diags.AddError("Invalid name", "name must not be empty.")
	}
	if strings.TrimSpace(model.QueryString.ValueString()) == "" {
		diags.AddError("Invalid query_string", "query_string must not be empty.")
	}
	if !isOneOf(model.QueryOwnershipType.ValueString(), "Organization", "User") {
		diags.AddError("Invalid query_ownership_type", "query_ownership_type must be either Organization or User.")
	}
	if !isOneOf(model.QueryTimestampType.ValueString(), "EventTimestamp", "IngestTimestamp") {
		diags.AddError("Invalid query_timestamp_type", "query_timestamp_type must be either EventTimestamp or IngestTimestamp.")
	}
	if !model.TriggerMode.IsNull() && !isOneOf(model.TriggerMode.ValueString(), "ImmediateMode", "CompleteMode") {
		diags.AddError("Invalid trigger_mode", "trigger_mode must be ImmediateMode or CompleteMode when set.")
	}
	if model.QueryOwnershipType.ValueString() == "User" && strings.TrimSpace(model.RunAsUserID.ValueString()) == "" {
		diags.AddError("Invalid run_as_user_id", "run_as_user_id must be set when query_ownership_type is User.")
	}
	if model.QueryOwnershipType.ValueString() == "Organization" && !model.RunAsUserID.IsNull() && strings.TrimSpace(model.RunAsUserID.ValueString()) != "" {
		diags.AddError("Invalid run_as_user_id", "run_as_user_id must be omitted when query_ownership_type is Organization.")
	}

	actionRefs := stringsFromSet(ctx, model.ActionIDsOrNames, diags)
	throttleFields := stringsFromSet(ctx, model.ThrottleFields, diags)
	labels := stringsFromSet(ctx, model.Labels, diags)
	if diags.HasError() {
		return nil, false
	}

	if !update && len(throttleFields) > 1 {
		diags.AddError("Invalid throttle_fields", "Aggregate alert creation currently supports at most one throttle field because the create mutation still documents a legacy throttleField input path.")
	}
	if len(throttleFields) > 0 && model.ThrottleTimeSeconds.IsNull() {
		diags.AddError("Invalid throttle_fields", "throttle_fields can only be set when throttle_time_seconds is set.")
	}
	if diags.HasError() {
		return nil, false
	}

	input := map[string]interface{}{
		"viewName":              model.ViewName.ValueString(),
		"name":                  model.Name.ValueString(),
		"description":           model.Description.ValueString(),
		"enabled":               model.Enabled.ValueBool(),
		"queryString":           model.QueryString.ValueString(),
		"actionIdsOrNames":      actionRefs,
		"queryOwnershipType":    model.QueryOwnershipType.ValueString(),
		"queryTimestampType":    model.QueryTimestampType.ValueString(),
		"searchIntervalSeconds": model.SearchIntervalSeconds.ValueInt64(),
		"labels":                labels,
	}

	if !model.RunAsUserID.IsNull() && strings.TrimSpace(model.RunAsUserID.ValueString()) != "" {
		input["runAsUserId"] = model.RunAsUserID.ValueString()
	}
	if !model.ThrottleTimeSeconds.IsNull() {
		input["throttleTimeSeconds"] = model.ThrottleTimeSeconds.ValueInt64()
	}
	if len(throttleFields) > 0 {
		if update {
			input["throttleFields"] = throttleFields
		} else {
			input["throttleField"] = throttleFields[0]
		}
	}
	if !model.TriggerMode.IsNull() && strings.TrimSpace(model.TriggerMode.ValueString()) != "" {
		input["triggerMode"] = model.TriggerMode.ValueString()
	}

	return input, true
}

func aggregateAlertSelectionMutation(fieldName string, inputType string) string {
	return `
		mutation AggregateAlertMutation($input: ` + inputType + `!) {
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
				queryTimestampType
				searchIntervalSeconds
				throttleTimeSeconds
				throttleFields
				triggerMode
				lastError
				lastWarnings
				lastSuccessfulPoll
				lastTriggered
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

func (r *aggregateAlertResource) readAggregateAlert(ctx context.Context, viewName string, id string, name string) (aggregateAlertReadResult, bool, error) {
	var data struct {
		SearchDomain *struct {
			TypeName        string                     `json:"__typename"`
			AggregateAlerts []aggregateAlertReadResult `json:"aggregateAlerts"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		query GetAggregateAlerts($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository {
					aggregateAlerts {
						__typename
						id
						name
						description
						enabled
						queryString
						labels
						resource
						yamlTemplate
						queryTimestampType
						searchIntervalSeconds
						throttleTimeSeconds
						throttleFields
						triggerMode
						lastError
						lastWarnings
						lastSuccessfulPoll
						lastTriggered
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
					aggregateAlerts {
						__typename
						id
						name
						description
						enabled
						queryString
						labels
						resource
						yamlTemplate
						queryTimestampType
						searchIntervalSeconds
						throttleTimeSeconds
						throttleFields
						triggerMode
						lastError
						lastWarnings
						lastSuccessfulPoll
						lastTriggered
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
		return aggregateAlertReadResult{}, false, err
	}

	if data.SearchDomain == nil {
		return aggregateAlertReadResult{}, false, nil
	}

	return selectAggregateAlert(data.SearchDomain.AggregateAlerts, id, name)
}

func selectAggregateAlert(alerts []aggregateAlertReadResult, id string, name string) (aggregateAlertReadResult, bool, error) {
	if id != "" {
		for _, alert := range alerts {
			if alert.ID == id {
				return alert, true, nil
			}
		}
	}

	if name != "" {
		for _, alert := range alerts {
			if alert.Name == name {
				return alert, true, nil
			}
		}
	}

	return aggregateAlertReadResult{}, false, nil
}

func applyAggregateAlertReadResultToState(ctx context.Context, state *aggregateAlertResourceModel, result aggregateAlertReadResult, diags *diag.Diagnostics) {
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

	lastWarnings, warningDiags := normalizeOptionalStringSetFromAPI(ctx, state.LastWarnings, result.LastWarnings)
	diags.Append(warningDiags...)
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
	} else {
		state.RunAsUserID = types.StringNull()
	}
	state.QueryTimestampType = types.StringValue(result.QueryTimestampType)
	state.SearchIntervalSeconds = types.Int64Value(result.SearchIntervalSeconds)
	if result.ThrottleTimeSeconds == nil {
		state.ThrottleTimeSeconds = types.Int64Null()
	} else {
		state.ThrottleTimeSeconds = types.Int64Value(*result.ThrottleTimeSeconds)
	}
	state.ThrottleFields = throttleFields
	state.TriggerMode = normalizeOptionalStringFromAPI(state.TriggerMode, result.TriggerMode)
	state.Labels = labels
	state.Resource = types.StringValue(result.Resource)
	state.YAMLTemplate = normalizeYAMLTemplateFromAPI(state.YAMLTemplate, result.YAMLTemplate)
	state.LastError = normalizeOptionalStringFromAPI(state.LastError, result.LastError)
	state.LastWarnings = lastWarnings
	if result.LastSuccessfulPoll == nil {
		state.LastSuccessfulPoll = types.Int64Null()
	} else {
		state.LastSuccessfulPoll = types.Int64Value(*result.LastSuccessfulPoll)
	}
	if result.LastTriggered == nil {
		state.LastTriggered = types.Int64Null()
	} else {
		state.LastTriggered = types.Int64Value(*result.LastTriggered)
	}
}
