package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func TestViewRoleAssignmentID(t *testing.T) {
	got := viewRoleAssignmentID("group-1", "view-2", "role-3")
	want := "group-1:view-2:role-3"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestViewRoleAssignmentSchema(t *testing.T) {
	r := NewViewRoleAssignmentResource()

	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics from Schema: %v", resp.Diagnostics)
	}

	cases := map[string]struct {
		required        bool
		computed        bool
		requiresReplace bool
		useStateForUnknown bool
	}{
		"id":       {required: false, computed: true, requiresReplace: false, useStateForUnknown: true},
		"group_id": {required: true, computed: false, requiresReplace: true},
		"view_id":  {required: true, computed: false, requiresReplace: true},
		"role_id":  {required: true, computed: false, requiresReplace: true},
	}

	for name, want := range cases {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("expected attribute %q on view_role_assignment schema", name)
		}

		strAttr, ok := attr.(schema.StringAttribute)
		if !ok {
			t.Fatalf("expected attribute %q to be a StringAttribute, got %T", name, attr)
		}

		if strAttr.Required != want.required {
			t.Fatalf("attribute %q: expected Required=%v, got %v", name, want.required, strAttr.Required)
		}
		if strAttr.Computed != want.computed {
			t.Fatalf("attribute %q: expected Computed=%v, got %v", name, want.computed, strAttr.Computed)
		}

		hasRequiresReplace := false
		hasUseStateForUnknown := false
		for _, pm := range strAttr.PlanModifiers {
			switch pm.(type) {
			case planmodifier.String:
				// Plan modifiers from stringplanmodifier are unexported types,
				// so identify them by their Description text.
				desc := pm.Description(context.Background())
				if desc == stringplanmodifier.RequiresReplace().Description(context.Background()) {
					hasRequiresReplace = true
				}
				if desc == stringplanmodifier.UseStateForUnknown().Description(context.Background()) {
					hasUseStateForUnknown = true
				}
			}
		}

		if hasRequiresReplace != want.requiresReplace {
			t.Fatalf("attribute %q: expected RequiresReplace=%v, got %v", name, want.requiresReplace, hasRequiresReplace)
		}
		if hasUseStateForUnknown != want.useStateForUnknown {
			t.Fatalf("attribute %q: expected UseStateForUnknown=%v, got %v", name, want.useStateForUnknown, hasUseStateForUnknown)
		}
	}
}

func TestViewRoleAssignmentUpdateRejected(t *testing.T) {
	r := NewViewRoleAssignmentResource().(*viewRoleAssignmentResource)

	var resp resource.UpdateResponse
	r.Update(context.Background(), resource.UpdateRequest{}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected Update to surface a diagnostic error, got none")
	}

	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "View Role Assignment Update Not Supported" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'View Role Assignment Update Not Supported' diagnostic, got %v", resp.Diagnostics)
	}
}
