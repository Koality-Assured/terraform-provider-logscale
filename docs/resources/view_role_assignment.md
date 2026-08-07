---
page_title: "logscale_view_role_assignment Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Manages a view-scoped role assignment for a LogScale group.
---

# logscale_view_role_assignment (Resource)

Manages a view-scoped role assignment for a LogScale group.

See the matching `Usage Examples` and `Important Behavior Notes` sections of [`README.md`](../../README.md) for the current schema, lifecycle constraints, and tenant-validation caveats.

## Behavior notes

- In LogScale a repository is a kind of view, so `view_id` accepts either a view ID or a repository ID. Read preserves state; all attributes are `RequiresReplace`.

## Schema

Generated schema details will be populated when `terraform-plugin-docs` is wired into the build. Until then, the authoritative schema is the corresponding Go file under [`internal/provider/`](../../internal/provider/).
