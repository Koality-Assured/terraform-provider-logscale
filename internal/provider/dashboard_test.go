package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSelectDashboardByID(t *testing.T) {
	result, found, err := selectDashboard([]dashboardListItem{
		{ID: "dash-1", Name: "First"},
		{ID: "dash-2", Name: "Second"},
	}, "dash-2", "")
	if err != nil {
		t.Fatalf("unexpected error selecting dashboard: %v", err)
	}
	if !found {
		t.Fatalf("expected dashboard to be found")
	}
	if result.Name != "Second" {
		t.Fatalf("expected Second, got %q", result.Name)
	}
}

func TestSelectDashboardByNameCaseInsensitive(t *testing.T) {
	result, found, err := selectDashboard([]dashboardListItem{
		{ID: "dash-1", Name: "Hosts"},
	}, "", "hosts")
	if err != nil {
		t.Fatalf("unexpected error selecting dashboard: %v", err)
	}
	if !found {
		t.Fatalf("expected dashboard to be found")
	}
	if result.ID != "dash-1" {
		t.Fatalf("expected dash-1, got %q", result.ID)
	}
}

func TestApplyDashboardReadResultToStatePreservesEquivalentYAMLTemplate(t *testing.T) {
	state := dashboardResourceModel{
		ViewName:     types.StringValue("security-ops"),
		Name:         types.StringValue("hosts"),
		YAMLTemplate: types.StringValue("widgets:\r\n  - name: hosts\r\n"),
		Labels:       types.SetNull(types.StringType),
	}

	var diagnostics diag.Diagnostics
	applyDashboardReadResultToState(context.Background(), &state, dashboardListItem{
		ID:           "dash-1",
		Name:         "hosts",
		DisplayName:  "Hosts",
		Description:  "Host dashboard",
		YAMLTemplate: "widgets:\n  - name: hosts\n",
		Labels:       []string{"infra"},
		Resource:     "dashboard/hosts",
		SearchDomain: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   "view-1",
			Name: "security-ops",
		},
	}, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics applying dashboard state: %v", diagnostics)
	}

	if state.YAMLTemplate.ValueString() != "widgets:\r\n  - name: hosts\r\n" {
		t.Fatalf("expected current YAML template formatting to be preserved, got %q", state.YAMLTemplate.ValueString())
	}
}

func TestValidateDashboardModel(t *testing.T) {
	model := dashboardResourceModel{
		ViewName:     types.StringValue("security-ops"),
		Name:         types.StringValue("hosts"),
		YAMLTemplate: types.StringValue("widgets:\n  - name: hosts\n"),
	}

	var diagnostics diag.Diagnostics
	validateDashboardModel(model, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("expected dashboard model to validate, got %v", diagnostics)
	}
}
