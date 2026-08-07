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
	_ datasource.DataSource              = &savedQueryDataSource{}
	_ datasource.DataSourceWithConfigure = &savedQueryDataSource{}
	_ datasource.DataSource              = &savedQueriesDataSource{}
	_ datasource.DataSourceWithConfigure = &savedQueriesDataSource{}
)

func NewSavedQueryDataSource() datasource.DataSource {
	return &savedQueryDataSource{}
}

func NewSavedQueriesDataSource() datasource.DataSource {
	return &savedQueriesDataSource{}
}

type savedQueryDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type savedQueriesDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type savedQueryDataSourceModel struct {
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

type savedQueriesDataSourceModel struct {
	ID           types.String             `tfsdk:"id"`
	ViewName     types.String             `tfsdk:"view_name"`
	SavedQueries []savedQuerySummaryModel `tfsdk:"saved_queries"`
}

type savedQuerySummaryModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	IsStarred   types.Bool   `tfsdk:"is_starred"`
	Resource    types.String `tfsdk:"resource"`
}

func (d *savedQueryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_query"
}

func (d *savedQueriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_queries"
}

func (d *savedQueryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a LogScale saved query by ID or name within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Description: "Saved query ID. Provide either id or name.", Optional: true, Computed: true},
			"view_name":     schema.StringAttribute{Description: "Repository or view that owns the saved query.", Required: true},
			"name":          schema.StringAttribute{Description: "Saved query name. Provide either id or name.", Optional: true, Computed: true},
			"display_name":  schema.StringAttribute{Description: "Saved query display name.", Computed: true},
			"description":   schema.StringAttribute{Description: "Saved query description.", Computed: true},
			"yaml_template": schema.StringAttribute{Description: "Saved query YAML template.", Computed: true},
			"labels":        schema.SetAttribute{Description: "Labels attached to the saved query.", Computed: true, ElementType: types.StringType},
			"query_string":  schema.StringAttribute{Description: "Query string returned by LogScale.", Computed: true},
			"is_live":       schema.BoolAttribute{Description: "Whether the saved query uses live streaming data.", Computed: true},
			"query_start":   schema.StringAttribute{Description: "Relative query start returned by LogScale.", Computed: true},
			"query_end":     schema.StringAttribute{Description: "Relative query end returned by LogScale.", Computed: true},
			"is_starred":    schema.BoolAttribute{Description: "Whether the saved query is starred.", Computed: true},
			"resource":      schema.StringAttribute{Description: "Provider-facing resource identifier returned by LogScale.", Computed: true},
		},
	}
}

func (d *savedQueriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists LogScale saved queries within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Description: "Synthetic identifier for this saved-query listing request.", Computed: true},
			"view_name": schema.StringAttribute{Description: "Repository or view whose saved queries should be returned.", Required: true},
			"saved_queries": schema.ListNestedAttribute{
				Description: "Saved queries returned for the supplied view.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true},
						"name":         schema.StringAttribute{Computed: true},
						"display_name": schema.StringAttribute{Computed: true},
						"is_starred":   schema.BoolAttribute{Computed: true},
						"resource":     schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *savedQueryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *savedQueriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *savedQueryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data savedQueryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.ID.IsNull() == data.Name.IsNull() {
		resp.Diagnostics.AddError("Invalid saved query lookup", "Specify exactly one of id or name.")
		return
	}

	resource := &savedQueryResource{client: d.client, config: d.config}
	result, found, err := resource.readSavedQuery(ctx, data.ViewName.ValueString(), data.ID.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading saved query", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Saved query not found", "No saved query matched the supplied lookup criteria.")
		return
	}

	state := savedQueryResourceModel{
		ID:       data.ID,
		ViewName: data.ViewName,
		Name:     data.Name,
		Labels:   types.SetNull(types.StringType),
	}
	applySavedQueryReadResultToState(ctx, &state, result, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = state.ID
	data.Name = state.Name
	data.DisplayName = state.DisplayName
	data.Description = state.Description
	data.YAMLTemplate = state.YAMLTemplate
	data.Labels = state.Labels
	data.QueryString = state.QueryString
	data.IsLive = state.IsLive
	data.QueryStart = state.QueryStart
	data.QueryEnd = state.QueryEnd
	data.IsStarred = state.IsStarred
	data.Resource = state.Resource

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *savedQueriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data savedQueriesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	results, err := lookupSavedQueries(ctx, d.client, d.config.Endpoint, data.ViewName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading saved queries", err.Error())
		return
	}

	savedQueries := make([]savedQuerySummaryModel, 0, len(results))
	for _, item := range results {
		savedQueries = append(savedQueries, savedQuerySummaryModel{
			ID:          types.StringValue(item.ID),
			Name:        types.StringValue(item.Name),
			DisplayName: types.StringValue(item.DisplayName),
			IsStarred:   types.BoolValue(item.IsStarred),
			Resource:    types.StringValue(item.Resource),
		})
	}

	data.ID = types.StringValue(fmt.Sprintf("view=%s", data.ViewName.ValueString()))
	data.SavedQueries = savedQueries
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func lookupSavedQueries(ctx context.Context, client *http.Client, endpoint string, viewName string) ([]savedQueryReadResult, error) {
	var data struct {
		SearchDomain *struct {
			TypeName     string                 `json:"__typename"`
			SavedQueries []savedQueryReadResult `json:"savedQueries"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, client, endpoint, `
		query GetSavedQueriesList($viewName: String!) {
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
		return nil, err
	}
	if data.SearchDomain == nil {
		return nil, nil
	}
	return data.SearchDomain.SavedQueries, nil
}
