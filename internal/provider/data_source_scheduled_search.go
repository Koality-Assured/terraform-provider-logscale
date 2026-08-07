package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &scheduledSearchDataSource{}
	_ datasource.DataSourceWithConfigure = &scheduledSearchDataSource{}
)

func NewScheduledSearchDataSource() datasource.DataSource {
	return &scheduledSearchDataSource{}
}

type scheduledSearchDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type scheduledSearchDataSourceModel struct {
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

func (d *scheduledSearchDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_search"
}

func (d *scheduledSearchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a LogScale scheduled search by ID or by name within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Scheduled search ID.",
				Optional:    true,
				Computed:    true,
			},
			"view_name": schema.StringAttribute{
				Description: "Repository or view that owns the scheduled search.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Scheduled search name. Provide either id or name.",
				Optional:    true,
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Scheduled search description.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the scheduled search is enabled.",
				Computed:    true,
			},
			"query_string": schema.StringAttribute{
				Description: "The LogScale query executed by the scheduled search.",
				Computed:    true,
			},
			"action_ids_or_names": schema.SetAttribute{
				Description: "Action references resolved by the provider.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the scheduled search.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"query_ownership_type": schema.StringAttribute{
				Description: "Ownership type for the scheduled search query.",
				Computed:    true,
			},
			"run_as_user_id": schema.StringAttribute{
				Description: "User ID used when query_ownership_type is User.",
				Computed:    true,
			},
			"schedule": schema.StringAttribute{
				Description: "Cron schedule used by the scheduled search.",
				Computed:    true,
			},
			"time_zone": schema.StringAttribute{
				Description: "Time zone used by the schedule.",
				Computed:    true,
			},
			"search_interval_seconds": schema.Int64Attribute{
				Description: "Search interval in seconds.",
				Computed:    true,
			},
			"search_interval_offset_seconds": schema.Int64Attribute{
				Description: "Search interval offset in seconds.",
				Computed:    true,
			},
			"query_timestamp_type": schema.StringAttribute{
				Description: "Timestamp mode for the scheduled search query.",
				Computed:    true,
			},
			"backfill_limit": schema.Int64Attribute{
				Description: "Backfill limit used for EventTimestamp searches.",
				Computed:    true,
			},
			"max_wait_time_seconds": schema.Int64Attribute{
				Description: "Max wait time used for IngestTimestamp searches.",
				Computed:    true,
			},
			"trigger_on_empty_result": schema.BoolAttribute{
				Description: "Whether the scheduled search triggers on empty results.",
				Computed:    true,
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

func (d *scheduledSearchDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}

	d.config = config
	d.client = newConfiguredHTTPClient(config)
}

func (d *scheduledSearchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data scheduledSearchDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() == data.Name.IsNull() {
		resp.Diagnostics.AddError("Invalid scheduled search lookup", "Specify exactly one of id or name.")
		return
	}

	resource := &scheduledSearchResource{client: d.client, config: d.config}
	result, found, err := resource.readScheduledSearch(ctx, data.ViewName.ValueString(), data.ID.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading scheduled search", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Scheduled search not found", "No scheduled search matched the supplied lookup criteria.")
		return
	}

	state := scheduledSearchResourceModel{
		ID:               data.ID,
		ViewName:         data.ViewName,
		Name:             data.Name,
		ActionIDsOrNames: types.SetNull(types.StringType),
		Labels:           types.SetNull(types.StringType),
		LastWarnings:     types.SetNull(types.StringType),
	}
	applyScheduledSearchReadResultToState(ctx, &state, result, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = state.ID
	data.Name = state.Name
	data.Description = state.Description
	data.Enabled = state.Enabled
	data.QueryString = state.QueryString
	data.ActionIDsOrNames = state.ActionIDsOrNames
	data.Labels = state.Labels
	data.QueryOwnershipType = state.QueryOwnershipType
	data.RunAsUserID = state.RunAsUserID
	data.Schedule = state.Schedule
	data.TimeZone = state.TimeZone
	data.SearchIntervalSeconds = state.SearchIntervalSeconds
	data.SearchIntervalOffsetSeconds = state.SearchIntervalOffsetSeconds
	data.QueryTimestampType = state.QueryTimestampType
	data.BackfillLimit = state.BackfillLimit
	data.MaxWaitTimeSeconds = state.MaxWaitTimeSeconds
	data.TriggerOnEmptyResult = state.TriggerOnEmptyResult
	data.Resource = state.Resource
	data.YAMLTemplate = state.YAMLTemplate
	data.LastError = state.LastError
	data.LastWarnings = state.LastWarnings
	data.TimeOfLastExecution = state.TimeOfLastExecution
	data.TimeOfLastTrigger = state.TimeOfLastTrigger
	data.TimeOfNextPlannedExecution = state.TimeOfNextPlannedExecution

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
