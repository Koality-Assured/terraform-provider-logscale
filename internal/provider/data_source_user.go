package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
)

var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

type userDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type userDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	Email       types.String `tfsdk:"email"`
	DisplayName types.String `tfsdk:"display_name"`
	IsRoot      types.Bool   `tfsdk:"is_root"`
	IsOrgRoot   types.Bool   `tfsdk:"is_org_root"`
}

type userResult struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	IsRoot      bool   `json:"isRoot"`
	IsOrgRoot   bool   `json:"isOrgRoot"`
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a single existing LogScale user by id, username, email, or display_name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "User ID. Provide exactly one lookup field.",
				Optional:    true,
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "Username. Provide exactly one lookup field.",
				Optional:    true,
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "Email address. Provide exactly one lookup field.",
				Optional:    true,
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "Display name. Provide exactly one lookup field.",
				Optional:    true,
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
	}
}

func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.APIToken})
	d.client = oauth2.NewClient(context.Background(), src)
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data userDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	lookupCount := 0
	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		lookupCount++
	}
	if !data.Username.IsNull() && data.Username.ValueString() != "" {
		lookupCount++
	}
	if !data.Email.IsNull() && data.Email.ValueString() != "" {
		lookupCount++
	}
	if !data.DisplayName.IsNull() && data.DisplayName.ValueString() != "" {
		lookupCount++
	}

	if lookupCount != 1 {
		resp.Diagnostics.AddError(
			"Invalid User Lookup Configuration",
			"Specify exactly one of id, username, email, or display_name.",
		)
		return
	}

	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		user, ok, err := d.lookupUserByID(ctx, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading user", err.Error())
			return
		}
		if !ok {
			resp.Diagnostics.AddError("User Not Found", fmt.Sprintf("No user found with id %q.", data.ID.ValueString()))
			return
		}
		applyUserResultToState(&data, user)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	matchField := ""
	matchValue := ""
	switch {
	case !data.Username.IsNull() && data.Username.ValueString() != "":
		matchField = "username"
		matchValue = data.Username.ValueString()
	case !data.Email.IsNull() && data.Email.ValueString() != "":
		matchField = "email"
		matchValue = data.Email.ValueString()
	default:
		matchField = "display_name"
		matchValue = data.DisplayName.ValueString()
	}

	users, err := d.lookupUsersPage(ctx, matchValue, 1, 100)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	var matches []userResult
	for _, user := range users.Page {
		switch matchField {
		case "username":
			if user.Username == matchValue {
				matches = append(matches, user)
			}
		case "email":
			if user.Email == matchValue {
				matches = append(matches, user)
			}
		case "display_name":
			if user.DisplayName == matchValue {
				matches = append(matches, user)
			}
		}
	}

	if len(matches) == 0 {
		resp.Diagnostics.AddError(
			"User Not Found",
			fmt.Sprintf("No user found with %s %q.", matchField, matchValue),
		)
		return
	}

	if len(matches) > 1 {
		resp.Diagnostics.AddError(
			"User Lookup Ambiguous",
			fmt.Sprintf("Multiple users matched %s %q. Use id for an exact lookup.", matchField, matchValue),
		)
		return
	}

	applyUserResultToState(&data, matches[0])
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *userDataSource) lookupUserByID(ctx context.Context, id string) (userResult, bool, error) {
	queryStr := `
		query GetUser($id: String!) {
			user(id: $id) {
				id
				username
				email
				displayName
				isRoot
				isOrgRoot
			}
		}
	`

	body, err := d.doGraphQLRequest(ctx, queryStr, map[string]interface{}{"id": id})
	if err != nil {
		return userResult{}, false, err
	}

	var gqlResp struct {
		Data struct {
			User *userResult `json:"user"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return userResult{}, false, fmt.Errorf("could not unmarshal response: %v\nBody: %s", err, body)
	}

	if len(gqlResp.Errors) > 0 {
		return userResult{}, false, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	if gqlResp.Data.User == nil {
		return userResult{}, false, nil
	}

	return *gqlResp.Data.User, true, nil
}

type usersPageQueryResult struct {
	Page []userResult `json:"page"`
	PageInfo struct {
		Number            int64 `json:"number"`
		TotalNumberOfRows int64 `json:"totalNumberOfRows"`
		Total             int64 `json:"total"`
	} `json:"pageInfo"`
}

func (d *userDataSource) lookupUsersPage(ctx context.Context, search string, pageNumber int64, pageSize int64) (usersPageQueryResult, error) {
	queryStr := `
		query GetUsersPage($search: String, $pageNumber: Int!, $pageSize: Int!) {
			usersPage(
				search: $search,
				orderBy: { userField: USERNAME, order: ASC },
				pageNumber: $pageNumber,
				pageSize: $pageSize
			) {
				page {
					id
					username
					email
					displayName
					isRoot
					isOrgRoot
				}
				pageInfo {
					number
					totalNumberOfRows
					total
				}
			}
		}
	`

	body, err := d.doGraphQLRequest(ctx, queryStr, map[string]interface{}{
		"search":     search,
		"pageNumber": pageNumber,
		"pageSize":   pageSize,
	})
	if err != nil {
		return usersPageQueryResult{}, err
	}

	var gqlResp struct {
		Data struct {
			UsersPage usersPageQueryResult `json:"usersPage"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return usersPageQueryResult{}, fmt.Errorf("could not unmarshal response: %v\nBody: %s", err, body)
	}

	if len(gqlResp.Errors) > 0 {
		return usersPageQueryResult{}, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data.UsersPage, nil
}

func (d *userDataSource) doGraphQLRequest(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", d.config.Endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := d.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	return body, nil
}

func applyUserResultToState(state *userDataSourceModel, user userResult) {
	state.ID = types.StringValue(user.ID)
	state.Username = types.StringValue(user.Username)
	state.Email = types.StringValue(user.Email)
	state.DisplayName = types.StringValue(user.DisplayName)
	state.IsRoot = types.BoolValue(user.IsRoot)
	state.IsOrgRoot = types.BoolValue(user.IsOrgRoot)
}
