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
	_ resource.Resource                = &scheduledSearchResource{}
	_ resource.ResourceWithConfigure   = &scheduledSearchResource{}
	_ resource.ResourceWithImportState = &scheduledSearchResource{}
)

func NewScheduledSearchResource() resource.Resource {
	return &scheduledSearchResource{}
}

type scheduledSearchResource struct {
	client *http.Client
	config *LogScaleConfig
}

type scheduledSearchResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	ViewName                    types.String `tfsdk:"view_name"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	Enabled                     types.Bool   `tfsdk:"enabled"`
	QueryString                 types.String `tfsdk:"query_string"`
	ActionIDsOrNames            types.Set    `tfsdk:"action_ids_or_names"`
	Labels                      types.Set    `tfsdk:"labels"`
	QueryOwnershipType          types.String `tfsdk:"query_ownership_type"`
	RunAsUserID                 types.String `tfsdk:"run_as_user_id"`
	Schedule                    types.String `tfsdk:"schedule"`
	TimeZone                    types.String `tfsdk:"time_zone"`
	SearchIntervalSeconds       types.Int64  `tfsdk:"search_interval_seconds"`
	SearchIntervalOffsetSeconds types.Int64  `tfsdk:"search_interval_offset_seconds"`
	QueryTimestampType          types.String `tfsdk:"query_timestamp_type"`
	BackfillLimit               types.Int64  `tfsdk:"backfill_limit"`
	MaxWaitTimeSeconds          types.Int64  `tfsdk:"max_wait_time_seconds"`
	TriggerOnEmptyResult        types.Bool   `tfsdk:"trigger_on_empty_result"`
	Resource                    types.String `tfsdk:"resource"`
	YAMLTemplate                types.String `tfsdk:"yaml_template"`
	LastError                   types.String `tfsdk:"last_error"`
	LastWarnings                types.Set    `tfsdk:"last_warnings"`
	TimeOfLastExecution         types.Int64  `tfsdk:"time_of_last_execution"`
	TimeOfLastTrigger           types.Int64  `tfsdk:"time_of_last_trigger"`
	TimeOfNextPlannedExecution  types.Int64  `tfsdk:"time_of_next_planned_execution"`
}

type scheduledSearchReadResult struct {
	TypeName                    string                  `json:"__typename"`
	ID                          string                  `json:"id"`
	Name                        string                  `json:"name"`
	Description                 string                  `json:"description"`
	Enabled                     bool                    `json:"enabled"`
	QueryString                 string                  `json:"queryString"`
	Labels                      []string                `json:"labels"`
	Resource                    string                  `json:"resource"`
	YAMLTemplate                string                  `json:"yamlTemplate"`
	QueryOwnership              filterAlertOwnershipAPI `json:"queryOwnership"`
	ActionsV2                   []filterAlertActionAPI  `json:"actionsV2"`
	QueryTimestampType          string                  `json:"queryTimestampType"`
	Schedule                    string                  `json:"schedule"`
	TimeZone                    string                  `json:"timeZone"`
	SearchIntervalSeconds       int64                   `json:"searchIntervalSeconds"`
	SearchIntervalOffsetSeconds *int64                  `json:"searchIntervalOffsetSeconds"`
	BackfillLimit               *int64                  `json:"backfillLimit"`
	MaxWaitTimeSeconds          *int64                  `json:"maxWaitTimeSeconds"`
	TriggerOnEmptyResult        bool                    `json:"triggerOnEmptyResult"`
	LastError                   string                  `json:"lastError"`
	LastWarnings                []string                `json:"lastWarnings"`
	TimeOfLastExecution         *int64                  `json:"timeOfLastExecution"`
	TimeOfLastTrigger           *int64                  `json:"timeOfLastTrigger"`
	TimeOfNextPlannedExecution  *int64                  `json:"timeOfNextPlannedExecution"`
}

func (r *scheduledSearchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_search"
}

func (r *scheduledSearchResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale scheduled search.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Scheduled search ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"view_name": schema.StringAttribute{
				Description: "Repository or view that owns the scheduled search.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Scheduled search name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional scheduled search description.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the scheduled search is enabled.",
				Required:    true,
			},
			"query_string": schema.StringAttribute{
				Description: "The LogScale query executed by the scheduled search.",
				Required:    true,
			},
			"action_ids_or_names": schema.SetAttribute{
				Description: "Action references used by the scheduled search.",
				Required:    true,
				ElementType: types.StringType,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the scheduled search.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"query_ownership_type": schema.StringAttribute{
				Description: "Ownership type for the scheduled search query, such as Organization or User.",
				Required:    true,
			},
			"run_as_user_id": schema.StringAttribute{
				Description: "User ID used when query_ownership_type is User.",
				Optional:    true,
			},
			"schedule": schema.StringAttribute{
				Description: "Cron schedule used by the scheduled search.",
				Required:    true,
			},
			"time_zone": schema.StringAttribute{
				Description: "Time zone used by the schedule, such as UTC or UTC+2.",
				Required:    true,
			},
			"search_interval_seconds": schema.Int64Attribute{
				Description: "Search interval in seconds.",
				Required:    true,
			},
			"search_interval_offset_seconds": schema.Int64Attribute{
				Description: "Search interval offset in seconds. Required when query_timestamp_type is EventTimestamp.",
				Optional:    true,
			},
			"query_timestamp_type": schema.StringAttribute{
				Description: "Timestamp mode for the scheduled search query, such as EventTimestamp or IngestTimestamp.",
				Required:    true,
			},
			"backfill_limit": schema.Int64Attribute{
				Description: "Backfill limit used when query_timestamp_type is EventTimestamp.",
				Optional:    true,
			},
			"max_wait_time_seconds": schema.Int64Attribute{
				Description: "Max wait time used when query_timestamp_type is IngestTimestamp.",
				Optional:    true,
			},
			"trigger_on_empty_result": schema.BoolAttribute{
				Description: "Whether the scheduled search should trigger on empty results.",
				Optional:    true,
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
				Description: "Last error returned by LogScale for the scheduled search.",
				Computed:    true,
			},
			"last_warnings": schema.SetAttribute{
				Description: "Last warnings returned by LogScale for the scheduled search.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"time_of_last_execution": schema.Int64Attribute{
				Description: "Unix timestamp for the time of last execution.",
				Computed:    true,
			},
			"time_of_last_trigger": schema.Int64Attribute{
				Description: "Unix timestamp for the time of last trigger.",
				Computed:    true,
			},
			"time_of_next_planned_execution": schema.Int64Attribute{
				Description: "Unix timestamp for the next planned execution.",
				Computed:    true,
			},
		},
	}
}

func (r *scheduledSearchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *scheduledSearchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduledSearchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, ok := scheduledSearchInputFromModel(ctx, plan, false, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		CreateScheduledSearchV2 scheduledSearchReadResult `json:"createScheduledSearchV2"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, scheduledSearchSelectionMutation("createScheduledSearchV2", "CreateScheduledSearchV2"), map[string]interface{}{"input": input}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating scheduled search", err.Error())
		return
	}

	applyScheduledSearchReadResultToState(ctx, &plan, data.CreateScheduledSearchV2, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *scheduledSearchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduledSearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, found, err := r.readScheduledSearch(ctx, state.ViewName.ValueString(), state.ID.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading scheduled search", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	applyScheduledSearchReadResultToState(ctx, &state, result, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *scheduledSearchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scheduledSearchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state scheduledSearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, ok := scheduledSearchInputFromModel(ctx, plan, true, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}
	input["id"] = state.ID.ValueString()
	input["viewName"] = state.ViewName.ValueString()

	var data struct {
		UpdateScheduledSearchV3 scheduledSearchReadResult `json:"updateScheduledSearchV3"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, scheduledSearchSelectionMutation("updateScheduledSearchV3", "UpdateScheduledSearchV3"), map[string]interface{}{"input": input}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error updating scheduled search", err.Error())
		return
	}

	plan.ViewName = state.ViewName
	applyScheduledSearchReadResultToState(ctx, &plan, data.UpdateScheduledSearchV3, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *scheduledSearchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduledSearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		DeleteScheduledSearchV2 *bool `json:"deleteScheduledSearchV2"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation DeleteScheduledSearch($input: DeleteScheduledSearchV2!) {
			deleteScheduledSearchV2(input: $input)
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"viewName": state.ViewName.ValueString(),
			"id":       state.ID.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting scheduled search", err.Error())
	}
}

func (r *scheduledSearchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier in the format view_name:scheduled_search_id.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("view_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func scheduledSearchInputFromModel(ctx context.Context, model scheduledSearchResourceModel, update bool, diags *diag.Diagnostics) (map[string]interface{}, bool) {
	if strings.TrimSpace(model.ViewName.ValueString()) == "" {
		diags.AddError("Invalid view_name", "view_name must not be empty.")
	}
	if strings.TrimSpace(model.Name.ValueString()) == "" {
		diags.AddError("Invalid name", "name must not be empty.")
	}
	if strings.TrimSpace(model.QueryString.ValueString()) == "" {
		diags.AddError("Invalid query_string", "query_string must not be empty.")
	}
	if strings.TrimSpace(model.Schedule.ValueString()) == "" {
		diags.AddError("Invalid schedule", "schedule must not be empty.")
	}
	if strings.TrimSpace(model.TimeZone.ValueString()) == "" {
		diags.AddError("Invalid time_zone", "time_zone must not be empty.")
	}
	if !isOneOf(model.QueryOwnershipType.ValueString(), "Organization", "User") {
		diags.AddError("Invalid query_ownership_type", "query_ownership_type must be either Organization or User.")
	}
	if !isOneOf(model.QueryTimestampType.ValueString(), "EventTimestamp", "IngestTimestamp") {
		diags.AddError("Invalid query_timestamp_type", "query_timestamp_type must be either EventTimestamp or IngestTimestamp.")
	}
	if model.QueryOwnershipType.ValueString() == "User" && strings.TrimSpace(model.RunAsUserID.ValueString()) == "" {
		diags.AddError("Invalid run_as_user_id", "run_as_user_id must be set when query_ownership_type is User.")
	}
	if model.QueryOwnershipType.ValueString() == "Organization" && !model.RunAsUserID.IsNull() && strings.TrimSpace(model.RunAsUserID.ValueString()) != "" {
		diags.AddError("Invalid run_as_user_id", "run_as_user_id must be omitted when query_ownership_type is Organization.")
	}

	actionRefs := stringsFromSet(ctx, model.ActionIDsOrNames, diags)
	labels := stringsFromSet(ctx, model.Labels, diags)
	if diags.HasError() {
		return nil, false
	}

	switch model.QueryTimestampType.ValueString() {
	case "EventTimestamp":
		if model.SearchIntervalOffsetSeconds.IsNull() {
			diags.AddError("Invalid search_interval_offset_seconds", "search_interval_offset_seconds must be set when query_timestamp_type is EventTimestamp.")
		}
		if model.BackfillLimit.IsNull() {
			diags.AddError("Invalid backfill_limit", "backfill_limit must be set when query_timestamp_type is EventTimestamp.")
		}
		if !model.MaxWaitTimeSeconds.IsNull() {
			diags.AddError("Invalid max_wait_time_seconds", "max_wait_time_seconds must be omitted when query_timestamp_type is EventTimestamp.")
		}
	case "IngestTimestamp":
		if !model.SearchIntervalOffsetSeconds.IsNull() {
			diags.AddError("Invalid search_interval_offset_seconds", "search_interval_offset_seconds must be omitted when query_timestamp_type is IngestTimestamp.")
		}
		if !model.BackfillLimit.IsNull() {
			diags.AddError("Invalid backfill_limit", "backfill_limit must be omitted when query_timestamp_type is IngestTimestamp.")
		}
		if model.MaxWaitTimeSeconds.IsNull() {
			diags.AddError("Invalid max_wait_time_seconds", "max_wait_time_seconds must be set when query_timestamp_type is IngestTimestamp.")
		}
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
		"labels":                labels,
		"queryOwnershipType":    model.QueryOwnershipType.ValueString(),
		"schedule":              model.Schedule.ValueString(),
		"timeZone":              model.TimeZone.ValueString(),
		"searchIntervalSeconds": model.SearchIntervalSeconds.ValueInt64(),
		"queryTimestampType":    model.QueryTimestampType.ValueString(),
	}

	if !model.RunAsUserID.IsNull() && strings.TrimSpace(model.RunAsUserID.ValueString()) != "" {
		input["runAsUserId"] = model.RunAsUserID.ValueString()
	}
	if !model.SearchIntervalOffsetSeconds.IsNull() {
		input["searchIntervalOffsetSeconds"] = model.SearchIntervalOffsetSeconds.ValueInt64()
	}
	if !model.BackfillLimit.IsNull() {
		input["backfillLimit"] = model.BackfillLimit.ValueInt64()
	}
	if !model.MaxWaitTimeSeconds.IsNull() {
		input["maxWaitTimeSeconds"] = model.MaxWaitTimeSeconds.ValueInt64()
	}
	if update || !model.TriggerOnEmptyResult.IsNull() {
		input["triggerOnEmptyResult"] = model.TriggerOnEmptyResult.ValueBool()
	}

	return input, true
}

func scheduledSearchSelectionMutation(fieldName string, inputType string) string {
	return `
		mutation ScheduledSearchMutation($input: ` + inputType + `!) {
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
				queryOwnership {
					__typename
					id
				}
				actionsV2 {
					id
					name
				}
				queryTimestampType
				schedule
				timeZone
				searchIntervalSeconds
				searchIntervalOffsetSeconds
				backfillLimit
				maxWaitTimeSeconds
				triggerOnEmptyResult
				lastError
				lastWarnings
				timeOfLastExecution
				timeOfLastTrigger
				timeOfNextPlannedExecution
			}
		}
	`
}

func (r *scheduledSearchResource) readScheduledSearch(ctx context.Context, viewName string, id string, name string) (scheduledSearchReadResult, bool, error) {
	var data struct {
		SearchDomain *struct {
			TypeName          string                      `json:"__typename"`
			ScheduledSearches []scheduledSearchReadResult `json:"scheduledSearches"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		query GetScheduledSearches($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository {
					scheduledSearches {
						__typename
						id
						name
						description
						enabled
						queryString
						labels
						resource
						yamlTemplate
						queryOwnership {
							__typename
							id
						}
						actionsV2 {
							id
							name
						}
						queryTimestampType
						schedule
						timeZone
						searchIntervalSeconds
						searchIntervalOffsetSeconds
						backfillLimit
						maxWaitTimeSeconds
						triggerOnEmptyResult
						lastError
						lastWarnings
						timeOfLastExecution
						timeOfLastTrigger
						timeOfNextPlannedExecution
					}
				}
				... on View {
					scheduledSearches {
						__typename
						id
						name
						description
						enabled
						queryString
						labels
						resource
						yamlTemplate
						queryOwnership {
							__typename
							id
						}
						actionsV2 {
							id
							name
						}
						queryTimestampType
						schedule
						timeZone
						searchIntervalSeconds
						searchIntervalOffsetSeconds
						backfillLimit
						maxWaitTimeSeconds
						triggerOnEmptyResult
						lastError
						lastWarnings
						timeOfLastExecution
						timeOfLastTrigger
						timeOfNextPlannedExecution
					}
				}
			}
		}
	`, map[string]interface{}{"viewName": viewName}, &data)
	if err != nil {
		return scheduledSearchReadResult{}, false, err
	}

	if data.SearchDomain == nil {
		return scheduledSearchReadResult{}, false, nil
	}

	return selectScheduledSearch(data.SearchDomain.ScheduledSearches, id, name)
}

func selectScheduledSearch(searches []scheduledSearchReadResult, id string, name string) (scheduledSearchReadResult, bool, error) {
	if id != "" {
		for _, search := range searches {
			if search.ID == id {
				return search, true, nil
			}
		}
	}

	if name != "" {
		for _, search := range searches {
			if search.Name == name {
				return search, true, nil
			}
		}
	}

	return scheduledSearchReadResult{}, false, nil
}

func applyScheduledSearchReadResultToState(ctx context.Context, state *scheduledSearchResourceModel, result scheduledSearchReadResult, diags *diag.Diagnostics) {
	actionRefs := make([]assetReference, 0, len(result.ActionsV2))
	for _, action := range result.ActionsV2 {
		actionRefs = append(actionRefs, assetReference{ID: action.ID, Name: action.Name})
	}

	preservedActionRefs, actionDiags := preserveCurrentReferencesIfResolvable(ctx, state.ActionIDsOrNames, actionRefs)
	diags.Append(actionDiags...)

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
	state.Labels = labels
	state.QueryOwnershipType = types.StringValue(strings.TrimSuffix(result.QueryOwnership.TypeName, "Ownership"))
	if result.QueryOwnership.TypeName == "UserOwnership" {
		state.RunAsUserID = types.StringValue(result.QueryOwnership.ID)
	} else {
		state.RunAsUserID = types.StringNull()
	}
	state.Schedule = types.StringValue(result.Schedule)
	state.TimeZone = types.StringValue(result.TimeZone)
	state.SearchIntervalSeconds = types.Int64Value(result.SearchIntervalSeconds)
	if result.SearchIntervalOffsetSeconds == nil {
		state.SearchIntervalOffsetSeconds = types.Int64Null()
	} else {
		state.SearchIntervalOffsetSeconds = types.Int64Value(*result.SearchIntervalOffsetSeconds)
	}
	state.QueryTimestampType = types.StringValue(result.QueryTimestampType)
	if result.BackfillLimit == nil {
		state.BackfillLimit = types.Int64Null()
	} else {
		state.BackfillLimit = types.Int64Value(*result.BackfillLimit)
	}
	if result.MaxWaitTimeSeconds == nil {
		state.MaxWaitTimeSeconds = types.Int64Null()
	} else {
		state.MaxWaitTimeSeconds = types.Int64Value(*result.MaxWaitTimeSeconds)
	}
	state.TriggerOnEmptyResult = normalizeOptionalBoolFromAPI(state.TriggerOnEmptyResult, result.TriggerOnEmptyResult)
	state.Resource = types.StringValue(result.Resource)
	state.YAMLTemplate = normalizeYAMLTemplateFromAPI(state.YAMLTemplate, result.YAMLTemplate)
	state.LastError = normalizeOptionalStringFromAPI(state.LastError, result.LastError)
	state.LastWarnings = lastWarnings
	if result.TimeOfLastExecution == nil {
		state.TimeOfLastExecution = types.Int64Null()
	} else {
		state.TimeOfLastExecution = types.Int64Value(*result.TimeOfLastExecution)
	}
	if result.TimeOfLastTrigger == nil {
		state.TimeOfLastTrigger = types.Int64Null()
	} else {
		state.TimeOfLastTrigger = types.Int64Value(*result.TimeOfLastTrigger)
	}
	if result.TimeOfNextPlannedExecution == nil {
		state.TimeOfNextPlannedExecution = types.Int64Null()
	} else {
		state.TimeOfNextPlannedExecution = types.Int64Value(*result.TimeOfNextPlannedExecution)
	}
}
