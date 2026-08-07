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
	_ datasource.DataSource              = &viewDataSource{}
	_ datasource.DataSourceWithConfigure = &viewDataSource{}
)

func NewViewDataSource() datasource.DataSource {
	return &viewDataSource{}
}

type viewDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type viewDataSourceModel struct {
	ID          types.String          `tfsdk:"id"`
	Name        types.String          `tfsdk:"name"`
	Description types.String          `tfsdk:"description"`
	Connections []viewConnectionModel `tfsdk:"repository_connection"`
}

func (d *viewDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view"
}

func (d *viewDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing LogScale view by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "View ID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "View name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "View description.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"repository_connection": schema.ListNestedBlock{
				Description: "Repository connections that back the view.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"repository_name": schema.StringAttribute{
							Description: "Repository name connected to the view.",
							Computed:    true,
						},
						"filter": schema.StringAttribute{
							Description: "Filter applied to the repository connection.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *viewDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.config = config
	d.client = newConfiguredHTTPClient(config)
}

func (d *viewDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data viewDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var gqlData struct {
		SearchDomain *struct {
			TypeName    string              `json:"__typename"`
			ID          string              `json:"id"`
			Name        string              `json:"name"`
			Description string              `json:"description"`
			Connections []viewConnectionAPI `json:"connections"`
		} `json:"searchDomain"`
	}

	err := executeGraphQL(ctx, d.client, d.config.Endpoint, `
		query GetView($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				id
				name
				description
				... on View {
					connections {
						filter
						repository {
							name
						}
					}
				}
			}
		}
	`, map[string]interface{}{
		"viewName": data.Name.ValueString(),
	}, &gqlData)
	if err != nil {
		resp.Diagnostics.AddError("Error reading view", err.Error())
		return
	}

	if gqlData.SearchDomain == nil || gqlData.SearchDomain.TypeName != "View" {
		resp.Diagnostics.AddError("View not found", fmt.Sprintf("No view named %q was found.", data.Name.ValueString()))
		return
	}

	applyViewReadResultToState((*viewResourceModel)(&data), viewReadResult{
		ID:          gqlData.SearchDomain.ID,
		Name:        gqlData.SearchDomain.Name,
		Description: gqlData.SearchDomain.Description,
		Connections: gqlData.SearchDomain.Connections,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
