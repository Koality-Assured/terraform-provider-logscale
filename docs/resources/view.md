---
page_title: "logscale_view Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Manages a LogScale view with one or more `repository_connection` blocks.
---

# logscale_view (Resource)

Manages a LogScale view with one or more `repository_connection` blocks.

See the matching `Usage Examples` and `Important Behavior Notes` sections of [`README.md`](../../README.md) for the current schema, lifecycle constraints, and tenant-validation caveats.

## Behavior notes

- Use `repository_connection`, not `connection`. `name` and `description` currently use replace semantics.

## Schema

Generated schema details will be populated when `terraform-plugin-docs` is wired into the build. Until then, the authoritative schema is the corresponding Go file under [`internal/provider/`](../../internal/provider/).
