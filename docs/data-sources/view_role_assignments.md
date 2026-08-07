---
page_title: "logscale_view_role_assignments Data Source - terraform-provider-logscale"
subcategory: ""
description: |-
  Lists view-scoped role assignments held by a LogScale group via group(groupId).roles.
---

# logscale_view_role_assignments (Data Source)

Lists view-scoped role assignments held by a LogScale group via `group(groupId).roles`. Each entry corresponds to a role granted on a specific view or repository (LogScale models repositories as a kind of view).

Entries without a search domain (organization-scoped or system-scoped role grants) are filtered out — those live on the sibling resources [`logscale_organization_role_assignment`](../resources/organization_role_assignment.md) and [`logscale_system_role_assignment`](../resources/system_role_assignment.md).

## Example Usage

```terraform
data "logscale_view_role_assignments" "secops" {
  group_id = logscale_group.secops.id
}

output "secops_view_assignments" {
  value = data.logscale_view_role_assignments.secops.assignments
}
```

## Schema

### Required

- `group_id` (String) The ID of the group whose view-scoped role assignments are listed.

### Read-Only

- `id` (String) Synthetic identifier for this listing (echoes the `group_id`).
- `assignments` (List of Object) View-scoped role assignments associated with the group. Each entry includes:
  - `id` (String) Synthetic assignment identifier (`group_id:view_id:role_id`), matching the resource ID format used by [`logscale_view_role_assignment`](../resources/view_role_assignment.md).
  - `role_id` (String) The ID of the assigned role.
  - `role_display_name` (String) The display name of the assigned role.
  - `view_id` (String) The ID of the view or repository the role applies to.
  - `view_name` (String) The name of the view or repository the role applies to.
