package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
)

func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

type usersDataSource struct {
	userDataSource
}

type usersDataSourceModel struct {
	ID                types.String              `tfsdk:"id"`
	Search            types.String              `tfsdk:"search"`
	PageNumber        types.Int64               `tfsdk:"page_number"`
	PageSize          types.Int64               `tfsdk:"page_size"`
	TotalRows         types.Int64               `tfsdk:"total_rows"`
	TotalPages        types.Int64               `tfsdk:"total_pages"`
	Users             []usersDataSourceUser     `tfsdk:"users"`
}

type usersDataSourceUser struct {
	ID          types.String `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	Email       types.String `tfsdk:"email"`
	DisplayName types.String `tfsdk:"display_name"`
	IsRoot      types.Bool   `tfsdk:"is_root"`
	IsOrgRoot   types.Bool   `tfsdk:"is_org_root"`
}

func (d *usersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *usersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists existing LogScale users using the organization-scoped usersPage query.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic identifier for this user listing request.",
				Computed:    true,
			},
			"search": schema.StringAttribute{
				Description: "Optional search text applied to usernames, emails, or display names.",
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
				Description: "Total number of matching users across all pages.",
				Computed:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "Total number of pages for this query.",
				Computed:    true,
			},
			"users": schema.ListNestedAttribute{
				Description: "Users returned for this page.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "User ID.",
							Computed:    true,
						},
						"username": schema.StringAttribute{
							Description: "Username.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "Email address.",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "Display name.",
							Computed:    true,
						},
						"is_root": schema.BoolAttribute{
							Description: "Whether the user has system root access.",
							Computed:    true,
						},
						"is_org_root": schema.BoolAttribute{
							Description: "Whether the user has organization ownership.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data usersDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
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

	if pageNumber < 1 {
		resp.Diagnostics.AddError("Invalid page_number", "page_number must be at least 1.")
		return
	}
	if pageSize < 1 {
		resp.Diagnostics.AddError("Invalid page_size", "page_size must be at least 1.")
		return
	}

	search := ""
	if !data.Search.IsNull() && !data.Search.IsUnknown() {
		search = data.Search.ValueString()
	}

	result, err := d.lookupUsersPage(ctx, search, pageNumber, pageSize)
	if err != nil {
		resp.Diagnostics.AddError("Error reading users", err.Error())
		return
	}

	users := make([]usersDataSourceUser, 0, len(result.Page))
	for _, user := range result.Page {
		users = append(users, usersDataSourceUser{
			ID:          types.StringValue(user.ID),
			Username:    types.StringValue(user.Username),
			Email:       types.StringValue(user.Email),
			DisplayName: types.StringValue(user.DisplayName),
			IsRoot:      types.BoolValue(user.IsRoot),
			IsOrgRoot:   types.BoolValue(user.IsOrgRoot),
		})
	}

	totalPages := int64(0)
	if pageSize > 0 {
		totalPages = (result.PageInfo.Total + pageSize - 1) / pageSize
	}

	data.ID = types.StringValue(fmt.Sprintf("search=%s|page=%d|size=%d", search, pageNumber, pageSize))
	data.Search = types.StringValue(search)
	data.PageNumber = types.Int64Value(pageNumber)
	data.PageSize = types.Int64Value(pageSize)
	data.TotalRows = types.Int64Value(result.PageInfo.TotalNumberOfRows)
	data.TotalPages = types.Int64Value(totalPages)
	data.Users = users

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
