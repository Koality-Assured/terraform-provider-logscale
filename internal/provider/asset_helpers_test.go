package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDiffStringSets(t *testing.T) {
	toAdd, toRemove := diffStringSets([]string{"b", "a"}, []string{"b", "c", "c"})

	assertStringSlicesEqual(t, []string{"c"}, toAdd)
	assertStringSlicesEqual(t, []string{"a"}, toRemove)
}

func TestPreserveCurrentReferencesIfResolvable(t *testing.T) {
	current, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"action-id-1", "second-action"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building test set: %v", diags)
	}

	resolved, resolvedDiags := preserveCurrentReferencesIfResolvable(context.Background(), current, []assetReference{
		{ID: "action-id-1", Name: "first-action"},
		{ID: "action-id-2", Name: "second-action"},
	})
	if resolvedDiags.HasError() {
		t.Fatalf("unexpected diagnostics resolving references: %v", resolvedDiags)
	}

	var actual []string
	resolvedDiags = resolved.ElementsAs(context.Background(), &actual, false)
	if resolvedDiags.HasError() {
		t.Fatalf("unexpected diagnostics reading resolved set: %v", resolvedDiags)
	}

	assertStringSlicesEqual(t, []string{"action-id-1", "second-action"}, actual)
}

func TestPreserveCurrentReferencesFallsBackToNames(t *testing.T) {
	resolved, diags := preserveCurrentReferencesIfResolvable(context.Background(), types.SetNull(types.StringType), []assetReference{
		{ID: "action-id-1", Name: "first-action"},
		{ID: "action-id-2", Name: "second-action"},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics resolving references: %v", diags)
	}

	var actual []string
	diags = resolved.ElementsAs(context.Background(), &actual, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics reading resolved set: %v", diags)
	}

	assertStringSlicesEqual(t, []string{"first-action", "second-action"}, actual)
}

func TestSelectWebhookActionPrefersWebhookType(t *testing.T) {
	result, found, err := selectWebhookAction([]webhookActionReadResult{
		{TypeName: "EmailAction", ID: "action-1", Name: "notify-email"},
		{TypeName: "WebhookAction", ID: "action-2", Name: "notify-webhook", URL: "https://example.invalid"},
	}, "action-2", "")
	if err != nil {
		t.Fatalf("unexpected error selecting webhook action: %v", err)
	}
	if !found {
		t.Fatalf("expected webhook action to be found")
	}
	if result.Name != "notify-webhook" {
		t.Fatalf("expected notify-webhook, got %q", result.Name)
	}
}
