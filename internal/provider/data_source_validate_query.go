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
	_ datasource.DataSource              = &validateQueryDataSource{}
	_ datasource.DataSourceWithConfigure = &validateQueryDataSource{}
)

func NewValidateQueryDataSource() datasource.DataSource {
	return &validateQueryDataSource{}
}

type validateQueryDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type validateQueryDataSourceModel struct {
	ID          types.String                   `tfsdk:"id"`
	QueryString types.String                   `tfsdk:"query_string"`
	Version     types.String                   `tfsdk:"version"`
	IsLive      types.Bool                     `tfsdk:"is_live"`
	Arguments   []validateQueryArgumentModel   `tfsdk:"argument"`
	IsValid     types.Bool                     `tfsdk:"is_valid"`
	Diagnostics []validateQueryDiagnosticModel `tfsdk:"diagnostic"`
}

type validateQueryArgumentModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type validateQueryDiagnosticModel struct {
	Severity types.String `tfsdk:"severity"`
	Message  types.String `tfsdk:"message"`
	Code     types.String `tfsdk:"code"`
}

type validateQueryResult struct {
	IsValid     bool                      `json:"isValid"`
	Diagnostics []validateQueryDiagnostic `json:"diagnostics"`
}

type validateQueryDiagnostic struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Code     string `json:"code"`
}

func (d *validateQueryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_validate_query"
}

func (d *validateQueryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Validates a LogScale query using validateQuery().",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Description: "Synthetic identifier for this validation request.", Computed: true},
			"query_string": schema.StringAttribute{Description: "Query text to validate.", Required: true},
			"version":      schema.StringAttribute{Description: "Language version to validate, such as legacy, xdr1, or federated1.", Required: true},
			"is_live":      schema.BoolAttribute{Description: "Whether to validate in live-query mode.", Optional: true, Computed: true},
			"is_valid":     schema.BoolAttribute{Description: "Whether LogScale considers the query valid.", Computed: true},
			"argument": schema.ListNestedAttribute{
				Description: "Optional query arguments used during validation.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":  schema.StringAttribute{Required: true},
						"value": schema.StringAttribute{Required: true},
					},
				},
			},
			"diagnostic": schema.ListNestedAttribute{
				Description: "Diagnostics returned by LogScale validation.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"severity": schema.StringAttribute{Computed: true},
						"message":  schema.StringAttribute{Computed: true},
						"code":     schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *validateQueryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *validateQueryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data validateQueryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !isOneOf(data.Version.ValueString(), "legacy", "xdr1", "federated1") {
		resp.Diagnostics.AddError("Invalid version", "version must be one of legacy, xdr1, or federated1.")
		return
	}

	arguments := make([]map[string]string, 0, len(data.Arguments))
	for _, argument := range data.Arguments {
		arguments = append(arguments, map[string]string{
			"name":  argument.Name.ValueString(),
			"value": argument.Value.ValueString(),
		})
	}

	isLive := false
	if !data.IsLive.IsNull() && !data.IsLive.IsUnknown() {
		isLive = data.IsLive.ValueBool()
	}

	var result struct {
		ValidateQuery validateQueryResult `json:"validateQuery"`
	}
	err := executeGraphQL(ctx, d.client, d.config.Endpoint, `
		query ValidateQuery($queryString: String!, $version: LanguageVersionEnum!, $isLive: Boolean!, $arguments: [QueryArgument!]) {
			validateQuery(queryString: $queryString, version: $version, isLive: $isLive, arguments: $arguments) {
				isValid
				diagnostics {
					severity
					message
					code
				}
			}
		}
	`, map[string]interface{}{
		"queryString": data.QueryString.ValueString(),
		"version":     data.Version.ValueString(),
		"isLive":      isLive,
		"arguments":   arguments,
	}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error validating query", err.Error())
		return
	}

	diagnostics := make([]validateQueryDiagnosticModel, 0, len(result.ValidateQuery.Diagnostics))
	for _, item := range result.ValidateQuery.Diagnostics {
		diagnostics = append(diagnostics, validateQueryDiagnosticModel{
			Severity: types.StringValue(item.Severity),
			Message:  types.StringValue(item.Message),
			Code:     types.StringValue(item.Code),
		})
	}

	data.ID = types.StringValue(fmt.Sprintf("version=%s|live=%t|query=%s", data.Version.ValueString(), isLive, data.QueryString.ValueString()))
	data.IsLive = types.BoolValue(isLive)
	data.IsValid = types.BoolValue(result.ValidateQuery.IsValid)
	data.Diagnostics = diagnostics
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
