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
	_ datasource.DataSource              = &webhookActionsDataSource{}
	_ datasource.DataSourceWithConfigure = &webhookActionsDataSource{}
	_ datasource.DataSource              = &filterAlertsDataSource{}
	_ datasource.DataSourceWithConfigure = &filterAlertsDataSource{}
	_ datasource.DataSource              = &aggregateAlertsDataSource{}
	_ datasource.DataSourceWithConfigure = &aggregateAlertsDataSource{}
	_ datasource.DataSource              = &scheduledSearchesDataSource{}
	_ datasource.DataSourceWithConfigure = &scheduledSearchesDataSource{}
)

func NewWebhookActionsDataSource() datasource.DataSource    { return &webhookActionsDataSource{} }
func NewFilterAlertsDataSource() datasource.DataSource      { return &filterAlertsDataSource{} }
func NewAggregateAlertsDataSource() datasource.DataSource   { return &aggregateAlertsDataSource{} }
func NewScheduledSearchesDataSource() datasource.DataSource { return &scheduledSearchesDataSource{} }

type webhookActionsDataSource struct {
	client *http.Client
	config *LogScaleConfig
}
type filterAlertsDataSource struct {
	client *http.Client
	config *LogScaleConfig
}
type aggregateAlertsDataSource struct {
	client *http.Client
	config *LogScaleConfig
}
type scheduledSearchesDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type viewScopedListingModel struct {
	ID       types.String `tfsdk:"id"`
	ViewName types.String `tfsdk:"view_name"`
}

type webhookActionSummaryModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Method   types.String `tfsdk:"method"`
	URL      types.String `tfsdk:"url"`
	Resource types.String `tfsdk:"resource"`
}

type filterAlertSummaryModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Resource types.String `tfsdk:"resource"`
}

type aggregateAlertSummaryModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Resource types.String `tfsdk:"resource"`
}

type scheduledSearchSummaryModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Resource types.String `tfsdk:"resource"`
}

type webhookActionsListModel struct {
	ID             types.String                `tfsdk:"id"`
	ViewName       types.String                `tfsdk:"view_name"`
	WebhookActions []webhookActionSummaryModel `tfsdk:"webhook_actions"`
}

type filterAlertsListModel struct {
	ID           types.String              `tfsdk:"id"`
	ViewName     types.String              `tfsdk:"view_name"`
	FilterAlerts []filterAlertSummaryModel `tfsdk:"filter_alerts"`
}

type aggregateAlertsListModel struct {
	ID              types.String                 `tfsdk:"id"`
	ViewName        types.String                 `tfsdk:"view_name"`
	AggregateAlerts []aggregateAlertSummaryModel `tfsdk:"aggregate_alerts"`
}

type scheduledSearchesListModel struct {
	ID                types.String                  `tfsdk:"id"`
	ViewName          types.String                  `tfsdk:"view_name"`
	ScheduledSearches []scheduledSearchSummaryModel `tfsdk:"scheduled_searches"`
}

func configureViewScopedDataSource(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) (*http.Client, *LogScaleConfig, bool) {
	if req.ProviderData == nil {
		return nil, nil, false
	}
	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return nil, nil, false
	}
	return newConfiguredHTTPClient(config), config, true
}

func (d *webhookActionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, config, ok := configureViewScopedDataSource(req, resp)
	if ok {
		d.client = client
		d.config = config
	}
}
func (d *filterAlertsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, config, ok := configureViewScopedDataSource(req, resp)
	if ok {
		d.client = client
		d.config = config
	}
}
func (d *aggregateAlertsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, config, ok := configureViewScopedDataSource(req, resp)
	if ok {
		d.client = client
		d.config = config
	}
}
func (d *scheduledSearchesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, config, ok := configureViewScopedDataSource(req, resp)
	if ok {
		d.client = client
		d.config = config
	}
}

func (d *webhookActionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_actions"
}
func (d *filterAlertsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_filter_alerts"
}
func (d *aggregateAlertsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aggregate_alerts"
}
func (d *scheduledSearchesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_searches"
}

func (d *webhookActionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists webhook actions within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true},
			"view_name": schema.StringAttribute{Required: true},
			"webhook_actions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Computed: true},
						"name":     schema.StringAttribute{Computed: true},
						"method":   schema.StringAttribute{Computed: true},
						"url":      schema.StringAttribute{Computed: true},
						"resource": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *filterAlertsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists filter alerts within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true},
			"view_name": schema.StringAttribute{Required: true},
			"filter_alerts": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Computed: true},
						"name":     schema.StringAttribute{Computed: true},
						"enabled":  schema.BoolAttribute{Computed: true},
						"resource": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *aggregateAlertsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists aggregate alerts within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true},
			"view_name": schema.StringAttribute{Required: true},
			"aggregate_alerts": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Computed: true},
						"name":     schema.StringAttribute{Computed: true},
						"enabled":  schema.BoolAttribute{Computed: true},
						"resource": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *scheduledSearchesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists scheduled searches within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true},
			"view_name": schema.StringAttribute{Required: true},
			"scheduled_searches": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Computed: true},
						"name":     schema.StringAttribute{Computed: true},
						"enabled":  schema.BoolAttribute{Computed: true},
						"resource": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *webhookActionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data webhookActionsListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := lookupWebhookActions(ctx, d.client, d.config.Endpoint, data.ViewName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook actions", err.Error())
		return
	}
	summaries := make([]webhookActionSummaryModel, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, webhookActionSummaryModel{ID: types.StringValue(item.ID), Name: types.StringValue(item.Name), Method: types.StringValue(item.Method), URL: types.StringValue(item.URL), Resource: types.StringValue(item.Resource)})
	}
	data.ID = types.StringValue(fmt.Sprintf("view=%s", data.ViewName.ValueString()))
	data.WebhookActions = summaries
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *filterAlertsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data filterAlertsListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := lookupFilterAlerts(ctx, d.client, d.config.Endpoint, data.ViewName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading filter alerts", err.Error())
		return
	}
	summaries := make([]filterAlertSummaryModel, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, filterAlertSummaryModel{ID: types.StringValue(item.ID), Name: types.StringValue(item.Name), Enabled: types.BoolValue(item.Enabled), Resource: types.StringValue(item.Resource)})
	}
	data.ID = types.StringValue(fmt.Sprintf("view=%s", data.ViewName.ValueString()))
	data.FilterAlerts = summaries
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *aggregateAlertsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data aggregateAlertsListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := lookupAggregateAlerts(ctx, d.client, d.config.Endpoint, data.ViewName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading aggregate alerts", err.Error())
		return
	}
	summaries := make([]aggregateAlertSummaryModel, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, aggregateAlertSummaryModel{ID: types.StringValue(item.ID), Name: types.StringValue(item.Name), Enabled: types.BoolValue(item.Enabled), Resource: types.StringValue(item.Resource)})
	}
	data.ID = types.StringValue(fmt.Sprintf("view=%s", data.ViewName.ValueString()))
	data.AggregateAlerts = summaries
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *scheduledSearchesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data scheduledSearchesListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := lookupScheduledSearches(ctx, d.client, d.config.Endpoint, data.ViewName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading scheduled searches", err.Error())
		return
	}
	summaries := make([]scheduledSearchSummaryModel, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, scheduledSearchSummaryModel{ID: types.StringValue(item.ID), Name: types.StringValue(item.Name), Enabled: types.BoolValue(item.Enabled), Resource: types.StringValue(item.Resource)})
	}
	data.ID = types.StringValue(fmt.Sprintf("view=%s", data.ViewName.ValueString()))
	data.ScheduledSearches = summaries
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func lookupWebhookActions(ctx context.Context, client *http.Client, endpoint string, viewName string) ([]webhookActionReadResult, error) {
	var data struct {
		SearchDomain *struct {
			TypeName string                    `json:"__typename"`
			Actions  []webhookActionReadResult `json:"actions"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, client, endpoint, `
		query GetWebhookActionsList($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository {
					actions {
						__typename
						id
						name
						resource
						... on WebhookAction {
							url
							method
						}
					}
				}
				... on View {
					actions {
						__typename
						id
						name
						resource
						... on WebhookAction {
							url
							method
						}
					}
				}
			}
		}
	`, map[string]interface{}{"viewName": viewName}, &data)
	if err != nil {
		return nil, err
	}
	if data.SearchDomain == nil {
		return nil, nil
	}
	out := make([]webhookActionReadResult, 0)
	for _, item := range data.SearchDomain.Actions {
		if item.TypeName == "WebhookAction" {
			out = append(out, item)
		}
	}
	return out, nil
}

func lookupFilterAlerts(ctx context.Context, client *http.Client, endpoint string, viewName string) ([]filterAlertReadResult, error) {
	var data struct {
		SearchDomain *struct {
			TypeName     string                  `json:"__typename"`
			FilterAlerts []filterAlertReadResult `json:"filterAlerts"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, client, endpoint, `
		query GetFilterAlertsList($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository { filterAlerts { id name enabled resource } }
				... on View { filterAlerts { id name enabled resource } }
			}
		}
	`, map[string]interface{}{"viewName": viewName}, &data)
	if err != nil {
		return nil, err
	}
	if data.SearchDomain == nil {
		return nil, nil
	}
	return data.SearchDomain.FilterAlerts, nil
}

func lookupAggregateAlerts(ctx context.Context, client *http.Client, endpoint string, viewName string) ([]aggregateAlertReadResult, error) {
	var data struct {
		SearchDomain *struct {
			TypeName        string                     `json:"__typename"`
			AggregateAlerts []aggregateAlertReadResult `json:"aggregateAlerts"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, client, endpoint, `
		query GetAggregateAlertsList($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository { aggregateAlerts { id name enabled resource } }
				... on View { aggregateAlerts { id name enabled resource } }
			}
		}
	`, map[string]interface{}{"viewName": viewName}, &data)
	if err != nil {
		return nil, err
	}
	if data.SearchDomain == nil {
		return nil, nil
	}
	return data.SearchDomain.AggregateAlerts, nil
}

func lookupScheduledSearches(ctx context.Context, client *http.Client, endpoint string, viewName string) ([]scheduledSearchReadResult, error) {
	var data struct {
		SearchDomain *struct {
			TypeName          string                      `json:"__typename"`
			ScheduledSearches []scheduledSearchReadResult `json:"scheduledSearches"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, client, endpoint, `
		query GetScheduledSearchesList($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository { scheduledSearches { id name enabled resource } }
				... on View { scheduledSearches { id name enabled resource } }
			}
		}
	`, map[string]interface{}{"viewName": viewName}, &data)
	if err != nil {
		return nil, err
	}
	if data.SearchDomain == nil {
		return nil, nil
	}
	return data.SearchDomain.ScheduledSearches, nil
}
