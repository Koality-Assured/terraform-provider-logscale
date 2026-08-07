---
page_title: "logscale_organization_role_assignment Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Manages an organization-scoped role assignment for a LogScale group.
---

# logscale_organization_role_assignment (Resource)

Manages an organization-scoped role assignment for a LogScale group.

See the matching `Usage Examples` and `Important Behavior Notes` sections of [`README.md`](../../README.md) for the current schema, lifecycle constraints, and tenant-validation caveats.

## Behavior notes

- Assign/remove supported. Read preserves state; changes are recreate-only.

## Schema

Generated schema details will be populated when `terraform-plugin-docs` is wired into the build. Until then, the authoritative schema is the corresponding Go file under [`internal/provider/`](../../internal/provider/).
