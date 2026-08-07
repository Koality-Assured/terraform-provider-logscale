package provider

import (
	"context"
	"testing"

	frameworkdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProviderMetadata(t *testing.T) {
	providerInstance := New("test-version")().(*ScaffoldingProvider)

	var resp frameworkprovider.MetadataResponse
	providerInstance.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &resp)

	if resp.TypeName != "logscale" {
		t.Fatalf("expected provider type name logscale, got %q", resp.TypeName)
	}

	if resp.Version != "test-version" {
		t.Fatalf("expected provider version test-version, got %q", resp.Version)
	}
}

func TestProviderResources(t *testing.T) {
	providerInstance := New("test")().(*ScaffoldingProvider)
	resourceFactories := providerInstance.Resources(context.Background())

	got := make([]string, 0, len(resourceFactories))
	for _, factory := range resourceFactories {
		resourceInstance := factory()
		var resp frameworkresource.MetadataResponse
		resourceInstance.Metadata(context.Background(), frameworkresource.MetadataRequest{
			ProviderTypeName: "logscale",
		}, &resp)
		got = append(got, resp.TypeName)
	}

	expected := []string{
		"logscale_repository",
		"logscale_parser",
		"logscale_view",
		"logscale_dashboard",
		"logscale_saved_query",
		"logscale_lookup_file",
		"logscale_aws_s3_sqs_ingest_feed",
		"logscale_webhook_action",
		"logscale_filter_alert",
		"logscale_aggregate_alert",
		"logscale_scheduled_search",
		"logscale_ingest_token",
		"logscale_group",
		"logscale_role",
		"logscale_organization_role_assignment",
		"logscale_system_role_assignment",
		"logscale_view_role_assignment",
		"logscale_default_role_assignment",
		"logscale_group_membership",
	}

	assertStringSlicesEqual(t, expected, got)
}

func TestProviderDataSources(t *testing.T) {
	providerInstance := New("test")().(*ScaffoldingProvider)
	dataSourceFactories := providerInstance.DataSources(context.Background())

	got := make([]string, 0, len(dataSourceFactories))
	for _, factory := range dataSourceFactories {
		dataSourceInstance := factory()
		var resp frameworkdatasource.MetadataResponse
		dataSourceInstance.Metadata(context.Background(), frameworkdatasource.MetadataRequest{
			ProviderTypeName: "logscale",
		}, &resp)
		got = append(got, resp.TypeName)
	}

	expected := []string{
		"logscale_role",
		"logscale_user",
		"logscale_users",
		"logscale_repository",
		"logscale_view",
		"logscale_saved_query",
		"logscale_saved_queries",
		"logscale_lookup_file",
		"logscale_lookup_files",
		"logscale_validate_query",
		"logscale_aws_s3_sqs_ingest_feed",
		"logscale_webhook_action",
		"logscale_webhook_actions",
		"logscale_filter_alert",
		"logscale_filter_alerts",
		"logscale_aggregate_alert",
		"logscale_aggregate_alerts",
		"logscale_scheduled_search",
		"logscale_scheduled_searches",
		"logscale_dashboard",
		"logscale_dashboards",
		"logscale_view_role_assignments",
	}

	assertStringSlicesEqual(t, expected, got)
}

func assertStringSlicesEqual(t *testing.T, expected []string, got []string) {
	t.Helper()

	if len(expected) != len(got) {
		t.Fatalf("expected %d items, got %d (%v)", len(expected), len(got), got)
	}

	for i := range expected {
		if expected[i] != got[i] {
			t.Fatalf("expected item %d to be %q, got %q", i, expected[i], got[i])
		}
	}
}
