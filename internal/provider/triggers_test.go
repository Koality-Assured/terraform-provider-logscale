package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSelectAggregateAlertByID(t *testing.T) {
	result, found, err := selectAggregateAlert([]aggregateAlertReadResult{
		{ID: "agg-1", Name: "First"},
		{ID: "agg-2", Name: "Second"},
	}, "agg-2", "")
	if err != nil {
		t.Fatalf("unexpected error selecting aggregate alert: %v", err)
	}
	if !found {
		t.Fatalf("expected aggregate alert to be found")
	}
	if result.Name != "Second" {
		t.Fatalf("expected Second, got %q", result.Name)
	}
}

func TestAggregateAlertInputRejectsMultipleThrottleFieldsOnCreate(t *testing.T) {
	throttleFields, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"host", "cluster"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics creating throttle_fields: %v", diags)
	}

	model := aggregateAlertResourceModel{
		ViewName:              types.StringValue("security-ops"),
		Name:                  types.StringValue("agg-search"),
		Enabled:               types.BoolValue(true),
		QueryString:           types.StringValue("#type=event"),
		ActionIDsOrNames:      types.SetNull(types.StringType),
		QueryOwnershipType:    types.StringValue("Organization"),
		QueryTimestampType:    types.StringValue("EventTimestamp"),
		SearchIntervalSeconds: types.Int64Value(300),
		ThrottleTimeSeconds:   types.Int64Value(300),
		ThrottleFields:        throttleFields,
		Labels:                types.SetNull(types.StringType),
	}

	var validationDiags diag.Diagnostics
	_, ok := aggregateAlertInputFromModel(context.Background(), model, false, &validationDiags)
	if ok {
		t.Fatalf("expected aggregate alert input validation to fail")
	}
	if !validationDiags.HasError() {
		t.Fatalf("expected diagnostics for aggregate alert validation failure")
	}
}

func TestSelectScheduledSearchByID(t *testing.T) {
	result, found, err := selectScheduledSearch([]scheduledSearchReadResult{
		{ID: "sched-1", Name: "Daily"},
		{ID: "sched-2", Name: "Hourly"},
	}, "sched-2", "")
	if err != nil {
		t.Fatalf("unexpected error selecting scheduled search: %v", err)
	}
	if !found {
		t.Fatalf("expected scheduled search to be found")
	}
	if result.Name != "Hourly" {
		t.Fatalf("expected Hourly, got %q", result.Name)
	}
}

func TestScheduledSearchInputRejectsMissingEventTimestampFields(t *testing.T) {
	model := scheduledSearchResourceModel{
		ViewName:              types.StringValue("security-ops"),
		Name:                  types.StringValue("daily-check"),
		Description:           types.StringNull(),
		Enabled:               types.BoolValue(true),
		QueryString:           types.StringValue("#type=event"),
		ActionIDsOrNames:      types.SetNull(types.StringType),
		Labels:                types.SetNull(types.StringType),
		QueryOwnershipType:    types.StringValue("Organization"),
		Schedule:              types.StringValue("0 * * * *"),
		TimeZone:              types.StringValue("UTC"),
		SearchIntervalSeconds: types.Int64Value(300),
		QueryTimestampType:    types.StringValue("EventTimestamp"),
	}

	var validationDiags diag.Diagnostics
	_, ok := scheduledSearchInputFromModel(context.Background(), model, false, &validationDiags)
	if ok {
		t.Fatalf("expected scheduled search input validation to fail")
	}
	if !validationDiags.HasError() {
		t.Fatalf("expected diagnostics for scheduled search validation failure")
	}
}

func TestScheduledSearchInputAcceptsIngestTimestampFields(t *testing.T) {
	model := scheduledSearchResourceModel{
		ViewName:              types.StringValue("security-ops"),
		Name:                  types.StringValue("daily-check"),
		Description:           types.StringNull(),
		Enabled:               types.BoolValue(true),
		QueryString:           types.StringValue("#type=event"),
		ActionIDsOrNames:      types.SetNull(types.StringType),
		Labels:                types.SetNull(types.StringType),
		QueryOwnershipType:    types.StringValue("Organization"),
		Schedule:              types.StringValue("0 * * * *"),
		TimeZone:              types.StringValue("UTC"),
		SearchIntervalSeconds: types.Int64Value(300),
		QueryTimestampType:    types.StringValue("IngestTimestamp"),
		MaxWaitTimeSeconds:    types.Int64Value(60),
	}

	var validationDiags diag.Diagnostics
	input, ok := scheduledSearchInputFromModel(context.Background(), model, false, &validationDiags)
	if !ok {
		t.Fatalf("expected scheduled search input validation to pass")
	}
	if validationDiags.HasError() {
		t.Fatalf("unexpected diagnostics validating scheduled search input: %v", validationDiags)
	}
	if input["queryString"] != "#type=event" {
		t.Fatalf("expected queryString to use camelCase key, got %#v", input)
	}
}
