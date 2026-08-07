package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &dashboardsDataSource{}
	_ datasource.DataSourceWithConfigure = &dashboardsDataSource{}
	_ datasource.DataSource              = &dashboardDataSource{}
	_ datasource.DataSourceWithConfigure = &dashboardDataSource{}
)

func NewDashboardsDataSource() datasource.DataSource {
	return &dashboardsDataSource{}
}

func NewDashboardDataSource() datasource.DataSource {
	return &dashboardDataSource{}
}

type dashboardsDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type dashboardDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type dashboardsDataSourceModel struct {
	ID         types.String            `tfsdk:"id"`
	Search     types.String            `tfsdk:"search"`
	PageNumber types.Int64             `tfsdk:"page_number"`
	PageSize   types.Int64             `tfsdk:"page_size"`
	TotalRows  types.Int64             `tfsdk:"total_rows"`
	TotalPages types.Int64             `tfsdk:"total_pages"`
	Dashboards []dashboardSummaryModel `tfsdk:"dashboards"`
}

type dashboardDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	DisplayName      types.String `tfsdk:"display_name"`
	Description      types.String `tfsdk:"description"`
	SearchDomainID   types.String `tfsdk:"search_domain_id"`
	SearchDomainName types.String `tfsdk:"search_domain_name"`
	Resource         types.String `tfsdk:"resource"`
	YAMLTemplate     types.String `tfsdk:"yaml_template"`
	Labels           types.Set    `tfsdk:"labels"`
}

type dashboardSummaryModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	DisplayName      types.String `tfsdk:"display_name"`
	SearchDomainID   types.String `tfsdk:"search_domain_id"`
	SearchDomainName types.String `tfsdk:"search_domain_name"`
}

type dashboardPageResult struct {
	Page     []dashboardListItem `json:"page"`
	PageInfo struct {
		Number            int64 `json:"number"`
		TotalNumberOfRows int64 `json:"totalNumberOfRows"`
		Total             int64 `json:"total"`
	} `json:"pageInfo"`
}

type dashboardListItem struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	Description  string   `json:"description"`
	Resource     string   `json:"resource"`
	YAMLTemplate string   `json:"yamlTemplate"`
	Labels       []string `json:"labels"`
	SearchDomain struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"searchDomain"`
}

func (d *dashboardsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboards"
}

func (d *dashboardDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (d *dashboardsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists LogScale dashboards using dashboardsPage().",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic identifier for this dashboard listing request.",
				Computed:    true,
			},
			"search": schema.StringAttribute{
				Description: "Optional search text passed to dashboardsPage().",
				Optional:    true,
			},
			"page_number": schema.Int64Attribute{
				Description: "Page number to return. Defaults to 1.",
				Optional:    true,
				Computed:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "Page size to return. Defaults to 50.",
				Optional:    true,
				Computed:    true,
			},
			"total_rows": schema.Int64Attribute{
				Description: "Total number of matching dashboards.",
				Computed:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "Total number of pages for the current search.",
				Computed:    true,
			},
			"dashboards": schema.ListNestedAttribute{
				Description: "Dashboards returned for the current page.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                 schema.StringAttribute{Computed: true},
						"name":               schema.StringAttribute{Computed: true},
						"display_name":       schema.StringAttribute{Computed: true},
						"search_domain_id":   schema.StringAttribute{Computed: true},
						"search_domain_name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *dashboardDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a LogScale dashboard by ID or name using dashboardsPage().",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Dashboard ID. Provide either id or name.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Dashboard name. Provide either id or name.",
				Optional:    true,
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "Dashboard display name.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Dashboard description.",
				Computed:    true,
			},
			"search_domain_id": schema.StringAttribute{
				Description: "Owning search domain ID.",
				Computed:    true,
			},
			"search_domain_name": schema.StringAttribute{
				Description: "Owning search domain name.",
				Computed:    true,
			},
			"resource": schema.StringAttribute{
				Description: "Provider-facing resource identifier returned by LogScale.",
				Computed:    true,
			},
			"yaml_template": schema.StringAttribute{
				Description: "Dashboard YAML template returned by LogScale.",
				Computed:    true,
			},
			"labels": schema.SetAttribute{
				Description: "Labels associated with the dashboard.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *dashboardsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dashboardDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dashboardsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dashboardsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pageNumber := int64(1)
	if !data.PageNumber.IsNull() && !data.PageNumber.IsUnknown() {
		pageNumber = data.PageNumber.ValueInt64()
	}
	pageSize := int64(50)
	if !data.PageSize.IsNull() && !data.PageSize.IsUnknown() {
		pageSize = data.PageSize.ValueInt64()
	}

	search := ""
	if !data.Search.IsNull() && !data.Search.IsUnknown() {
		search = data.Search.ValueString()
	}

	result, err := lookupDashboardsPage(ctx, d.client, d.config.Endpoint, search, pageNumber, pageSize)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dashboards", err.Error())
		return
	}

	dashboards := make([]dashboardSummaryModel, 0, len(result.Page))
	for _, dashboard := range result.Page {
		dashboards = append(dashboards, dashboardSummaryModel{
			ID:               types.StringValue(dashboard.ID),
			Name:             types.StringValue(dashboard.Name),
			DisplayName:      types.StringValue(dashboard.DisplayName),
			SearchDomainID:   types.StringValue(dashboard.SearchDomain.ID),
			SearchDomainName: types.StringValue(dashboard.SearchDomain.Name),
		})
	}

	data.ID = types.StringValue(fmt.Sprintf("search=%s|page=%d|size=%d", search, pageNumber, pageSize))
	data.Search = types.StringValue(search)
	data.PageNumber = types.Int64Value(pageNumber)
	data.PageSize = types.Int64Value(pageSize)
	data.TotalRows = types.Int64Value(result.PageInfo.TotalNumberOfRows)
	data.TotalPages = types.Int64Value(result.PageInfo.Total)
	data.Dashboards = dashboards

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *dashboardDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dashboardDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() == data.Name.IsNull() {
		resp.Diagnostics.AddError("Invalid dashboard lookup", "Specify exactly one of id or name.")
		return
	}

	item, found, err := lookupDashboard(ctx, d.client, d.config.Endpoint, data.ID.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading dashboard", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Dashboard not found", "No dashboard matched the supplied lookup criteria.")
		return
	}

	data.ID = types.StringValue(item.ID)
	data.Name = types.StringValue(item.Name)
	data.DisplayName = types.StringValue(item.DisplayName)
	data.Description = normalizeOptionalStringFromAPI(data.Description, item.Description)
	data.SearchDomainID = types.StringValue(item.SearchDomain.ID)
	data.SearchDomainName = types.StringValue(item.SearchDomain.Name)
	data.Resource = types.StringValue(item.Resource)
	data.YAMLTemplate = normalizeYAMLTemplateFromAPI(data.YAMLTemplate, item.YAMLTemplate)
	labelSet, labelDiags := stringSetValue(ctx, item.Labels)
	data.Labels = labelSet
	resp.Diagnostics.Append(labelDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func lookupDashboardsPage(ctx context.Context, client *http.Client, endpoint string, search string, pageNumber int64, pageSize int64) (dashboardPageResult, error) {
	var data struct {
		DashboardsPage dashboardPageResult `json:"dashboardsPage"`
	}
	err := executeGraphQL(ctx, client, endpoint, `
		query DashboardsPage($search: String, $pageNumber: Int!, $pageSize: Int!) {
			dashboardsPage(search: $search, pageNumber: $pageNumber, pageSize: $pageSize) {
				page {
					id
					name
					displayName
					description
					resource
					yamlTemplate
					labels
					searchDomain {
						id
						name
					}
				}
				pageInfo {
					number
					totalNumberOfRows
					total
				}
			}
		}
	`, map[string]interface{}{
		"search":     search,
		"pageNumber": pageNumber,
		"pageSize":   pageSize,
	}, &data)

	return data.DashboardsPage, err
}

func lookupDashboard(ctx context.Context, client *http.Client, endpoint string, id string, name string) (dashboardListItem, bool, error) {
	if id == "" && name == "" {
		return dashboardListItem{}, false, nil
	}

	search := name
	if search == "" {
		search = ""
	}

	pageNumber := int64(1)
	pageSize := int64(100)

	for {
		page, err := lookupDashboardsPage(ctx, client, endpoint, search, pageNumber, pageSize)
		if err != nil {
			return dashboardListItem{}, false, err
		}

		selected, found, err := selectDashboard(page.Page, id, name)
		if err != nil {
			return dashboardListItem{}, false, err
		}
		if found {
			return selected, true, nil
		}

		if pageNumber >= page.PageInfo.Total || page.PageInfo.Total == 0 {
			break
		}
		pageNumber++
	}

	return dashboardListItem{}, false, nil
}

func selectDashboard(items []dashboardListItem, id string, name string) (dashboardListItem, bool, error) {
	if id != "" {
		for _, item := range items {
			if item.ID == id {
				return item, true, nil
			}
		}
	}

	if name != "" {
		for _, item := range items {
			if strings.EqualFold(item.Name, name) {
				return item, true, nil
			}
		}
	}

	return dashboardListItem{}, false, nil
}
