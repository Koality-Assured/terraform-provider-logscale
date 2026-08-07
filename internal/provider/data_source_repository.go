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
	_ datasource.DataSource              = &repositoryDataSource{}
	_ datasource.DataSourceWithConfigure = &repositoryDataSource{}
)

func NewRepositoryDataSource() datasource.DataSource {
	return &repositoryDataSource{}
}

type repositoryDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type repositoryDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
}

func (d *repositoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (d *repositoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing LogScale repository by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Repository ID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Repository name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Repository description.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Repository type.",
				Computed:    true,
			},
		},
	}
}

func (d *repositoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *repositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var gqlData struct {
		Repository *struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Type        string `json:"type"`
		} `json:"repository"`
	}

	err := executeGraphQL(ctx, d.client, d.config.Endpoint, `
		query GetRepository($repoName: String!) {
			repository(name: $repoName) {
				id
				name
				type
				description
			}
		}
	`, map[string]interface{}{
		"repoName": data.Name.ValueString(),
	}, &gqlData)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository", err.Error())
		return
	}

	if gqlData.Repository == nil {
		resp.Diagnostics.AddError("Repository not found", fmt.Sprintf("No repository named %q was found.", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(gqlData.Repository.ID)
	data.Name = types.StringValue(gqlData.Repository.Name)
	data.Description = normalizeOptionalStringFromAPI(data.Description, gqlData.Repository.Description)
	data.Type = types.StringValue(gqlData.Repository.Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
