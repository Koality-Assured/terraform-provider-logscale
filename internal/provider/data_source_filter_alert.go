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
	_ datasource.DataSource              = &filterAlertDataSource{}
	_ datasource.DataSourceWithConfigure = &filterAlertDataSource{}
)

func NewFilterAlertDataSource() datasource.DataSource {
	return &filterAlertDataSource{}
}

type filterAlertDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type filterAlertDataSourceModel struct {
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

func (d *filterAlertDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_filter_alert"
}

func (d *filterAlertDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a LogScale filter alert by ID or by name within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Filter alert ID.",
				Optional:    true,
				Computed:    true,
			},
			"view_name": schema.StringAttribute{
				Description: "Repository or view that owns the alert.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Filter alert name. Provide either id or name.",
				Optional:    true,
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Filter alert description.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the alert is enabled.",
				Computed:    true,
			},
			"query_string": schema.StringAttribute{
				Description: "The LogScale query executed by the alert.",
				Computed:    true,
			},
			"action_ids_or_names": schema.SetAttribute{
				Description: "Action references resolved by the provider.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"query_ownership_type": schema.StringAttribute{
				Description: "Ownership type for the alert query.",
				Computed:    true,
			},
			"run_as_user_id": schema.StringAttribute{
				Description: "User ID used when query_ownership_type is User.",
				Computed:    true,
			},
			"throttle_time_seconds": schema.Int64Attribute{
				Description: "Throttle window in seconds.",
				Computed:    true,
			},
			"throttle_fields": schema.SetAttribute{
				Description: "Fields used to scope throttling.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the alert.",
				Computed:    true,
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

func (d *filterAlertDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *filterAlertDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data filterAlertDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() == data.Name.IsNull() {
		resp.Diagnostics.AddError("Invalid filter alert lookup", "Specify exactly one of id or name.")
		return
	}

	resource := &filterAlertResource{client: d.client, config: d.config}
	result, found, err := resource.readFilterAlert(ctx, data.ViewName.ValueString(), data.ID.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading filter alert", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Filter alert not found", "No filter alert matched the supplied lookup criteria.")
		return
	}

	state := filterAlertResourceModel{
		ID:               data.ID,
		ViewName:         data.ViewName,
		Name:             data.Name,
		ActionIDsOrNames: types.SetNull(types.StringType),
		ThrottleFields:   types.SetNull(types.StringType),
		Labels:           types.SetNull(types.StringType),
	}
	resource.applyFilterAlertReadResultToState(ctx, &state, result, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = state.ID
	data.Name = state.Name
	data.Description = state.Description
	data.Enabled = state.Enabled
	data.QueryString = state.QueryString
	data.ActionIDsOrNames = state.ActionIDsOrNames
	data.QueryOwnershipType = state.QueryOwnershipType
	data.RunAsUserID = state.RunAsUserID
	data.ThrottleTimeSeconds = state.ThrottleTimeSeconds
	data.ThrottleFields = state.ThrottleFields
	data.Labels = state.Labels
	data.Resource = state.Resource
	data.YAMLTemplate = state.YAMLTemplate

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
