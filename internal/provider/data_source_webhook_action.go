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
	_ datasource.DataSource              = &webhookActionDataSource{}
	_ datasource.DataSourceWithConfigure = &webhookActionDataSource{}
)

func NewWebhookActionDataSource() datasource.DataSource {
	return &webhookActionDataSource{}
}

type webhookActionDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type webhookActionDataSourceModel struct {
	ID                                      types.String               `tfsdk:"id"`
	ViewName                                types.String               `tfsdk:"view_name"`
	Name                                    types.String               `tfsdk:"name"`
	URL                                     types.String               `tfsdk:"url"`
	Method                                  types.String               `tfsdk:"method"`
	IgnoreSSL                               types.Bool                 `tfsdk:"ignore_ssl"`
	UseProxy                                types.Bool                 `tfsdk:"use_proxy"`
	BodyTemplate                            types.String               `tfsdk:"body_template"`
	Resource                                types.String               `tfsdk:"resource"`
	YAMLTemplate                            types.String               `tfsdk:"yaml_template"`
	RequiresOrgOwnedQueriesPermissionToEdit types.Bool                 `tfsdk:"requires_org_owned_queries_permission_to_edit"`
	Labels                                  types.Set                  `tfsdk:"labels"`
	Headers                                 []webhookActionHeaderModel `tfsdk:"header"`
}

func (d *webhookActionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_action"
}

func (d *webhookActionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a LogScale webhook action by ID or by name within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Webhook action ID.",
				Optional:    true,
				Computed:    true,
			},
			"view_name": schema.StringAttribute{
				Description: "Repository or view that owns the action.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Action name. Provide either id or name.",
				Optional:    true,
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "Webhook target URL.",
				Computed:    true,
			},
			"method": schema.StringAttribute{
				Description: "HTTP method used by the action.",
				Computed:    true,
			},
			"ignore_ssl": schema.BoolAttribute{
				Description: "Whether SSL validation errors are ignored.",
				Computed:    true,
			},
			"use_proxy": schema.BoolAttribute{
				Description: "Whether the action uses the configured proxy.",
				Computed:    true,
			},
			"body_template": schema.StringAttribute{
				Description: "Webhook request body template.",
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
			"requires_org_owned_queries_permission_to_edit": schema.BoolAttribute{
				Description: "Whether editing this action requires organization-owned query permissions.",
				Computed:    true,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the action.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"header": schema.ListNestedBlock{
				Description: "HTTP headers attached to the webhook request.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"header": schema.StringAttribute{
							Description: "Header name.",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "Header value.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *webhookActionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *webhookActionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data webhookActionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() == data.Name.IsNull() {
		resp.Diagnostics.AddError("Invalid webhook action lookup", "Specify exactly one of id or name.")
		return
	}

	resource := &webhookActionResource{client: d.client, config: d.config}
	readResult, found, err := resource.readWebhookAction(ctx, data.ViewName.ValueString(), data.ID.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook action", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Webhook action not found", "No webhook action matched the supplied lookup criteria.")
		return
	}

	labels, diags := stringSetValue(ctx, readResult.Labels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(readResult.ID)
	data.Name = types.StringValue(readResult.Name)
	data.URL = types.StringValue(readResult.URL)
	data.Method = types.StringValue(readResult.Method)
	data.IgnoreSSL = types.BoolValue(readResult.IgnoreSSL)
	data.UseProxy = types.BoolValue(readResult.UseProxy)
	data.BodyTemplate = types.StringValue(readResult.BodyTemplate)
	data.Resource = types.StringValue(readResult.Resource)
	data.YAMLTemplate = types.StringValue(readResult.YAMLTemplate)
	data.RequiresOrgOwnedQueriesPermissionToEdit = types.BoolValue(readResult.RequiresOrgOwnedQueriesPermissionToEdit)
	data.Labels = labels
	data.Headers = webhookHeadersFromAPI(readResult.Headers)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
