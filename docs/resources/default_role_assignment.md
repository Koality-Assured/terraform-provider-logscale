---
page_title: "logscale_default_role_assignment Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Configures the default role applied to a LogScale group's view-scoped role assignment behavior via updateDefaultRole.
---

# logscale_default_role_assignment (Resource)

Configures the default role applied to a LogScale group's view-scoped role-assignment behavior via the `updateDefaultRole` GraphQL mutation.

Read preserves Terraform state rather than reconciling against the remote group, matching the other role-assignment resources. Destroy emits a warning and detaches state because the upstream API does not expose a clean "unset default role" mutation.

## Example Usage

```terraform
resource "logscale_default_role_assignment" "secops_default" {
  group_id = logscale_group.secops.id
  role_id  = logscale_role.analyst.id
}
```

## Schema

### Required

- `group_id` (String) The ID of the group whose default role is being configured. Changing this attribute forces a new resource to be created.
- `role_id` (String) The ID of the role to set as the default for this group. Updating this attribute re-invokes `updateDefaultRole` in place.

### Read-Only

- `id` (String) Assignment identifier (`group_id:role_id`).

## Behavior notes

- `group_id` is `RequiresReplace`; `role_id` updates in place.
- Destroy does not unset the default role on the remote group. If you need to remove the default role entirely, do so manually in the LogScale UI.
