---
page_title: "logscale_role Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Manages a LogScale role. Roles bundle a set of view-level permissions that can be granted to a group via logscale_view_role_assignment.
---

# logscale_role (Resource)

Manages a LogScale role through the `createRole` / `updateRole` / `removeRole` GraphQL mutations. Roles bundle a set of view-level permissions that can be granted to a group via [`logscale_view_role_assignment`](./view_role_assignment.md).

## Example Usage

```terraform
resource "logscale_role" "analyst" {
  display_name = "Analyst"
  view_permissions = [
    "ReadAccess",
    "ChangeDashboards",
  ]
}
```

## Schema

### Required

- `display_name` (String) Display name for the role.
- `view_permissions` (Set of String) Set of view-level permission identifiers granted by this role. The exact set accepted by your tenant is defined by the LogScale permissions enumeration.

### Read-Only

- `id` (String) Role ID assigned by LogScale at create time.

## Import

```sh
terraform import logscale_role.example <role_id>
```
