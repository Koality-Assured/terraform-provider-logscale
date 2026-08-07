// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &ScaffoldingProvider{}

// ScaffoldingProvider defines the provider implementation.
type ScaffoldingProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaffoldingProviderModel describes the provider data model.
type ScaffoldingProviderModel struct {
	APIUrl   types.String `tfsdk:"api_url"`
	APIToken types.String `tfsdk:"api_token"`
}

func (p *ScaffoldingProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "logscale"
	resp.Version = p.version
}

func (p *ScaffoldingProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Falcon LogScale resources.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Description: "LogScale API URL (e.g., https://tenant.logscale.us-2.crowdstrike.com/graphql)",
				Required:    true,
			},
			"api_token": schema.StringAttribute{
				Description: "LogScale API token for authentication",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *ScaffoldingProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ScaffoldingProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that required fields are set
	if data.APIUrl.IsNull() || data.APIUrl.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing API URL",
			"The provider requires an api_url to be configured.",
		)
		return
	}

	if data.APIToken.IsNull() || data.APIToken.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"The provider requires an api_token to be configured.",
		)
		return
	}

	// Create configuration that will be passed to resources
	config := &LogScaleConfig{
		Endpoint: data.APIUrl.ValueString(),
		APIToken: data.APIToken.ValueString(),
	}

	// Make the configuration available to resources and data sources
	resp.DataSourceData = config
	resp.ResourceData = config
}

func (p *ScaffoldingProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRepositoryResource,
		NewParserResource,
		NewViewResource,
		NewDashboardResource,
		NewSavedQueryResource,
		NewLookupFileResource,
		NewAwsS3SqsIngestFeedResource,
		NewWebhookActionResource,
		NewFilterAlertResource,
		NewAggregateAlertResource,
		NewScheduledSearchResource,
		NewIngestTokenResource,
		NewGroupResource,
		NewRoleResource,
		NewOrganizationRoleAssignmentResource,
		NewSystemRoleAssignmentResource,
		NewViewRoleAssignmentResource,
		NewDefaultRoleAssignmentResource,
		NewGroupMembershipResource, // NEW: Add/remove users from groups
	}
}

func (p *ScaffoldingProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRoleDataSource,
		NewUserDataSource,
		NewUsersDataSource,
		NewRepositoryDataSource,
		NewViewDataSource,
		NewSavedQueryDataSource,
		NewSavedQueriesDataSource,
		NewLookupFileDataSource,
		NewLookupFilesDataSource,
		NewValidateQueryDataSource,
		NewAwsS3SqsIngestFeedDataSource,
		NewWebhookActionDataSource,
		NewWebhookActionsDataSource,
		NewFilterAlertDataSource,
		NewFilterAlertsDataSource,
		NewAggregateAlertDataSource,
		NewAggregateAlertsDataSource,
		NewScheduledSearchDataSource,
		NewScheduledSearchesDataSource,
		NewDashboardDataSource,
		NewDashboardsDataSource,
		NewViewRoleAssignmentsDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ScaffoldingProvider{
			version: version,
		}
	}
}
