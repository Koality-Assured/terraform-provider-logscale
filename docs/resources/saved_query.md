---
page_title: "logscale_saved_query Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Manages a LogScale saved query through the YAML-template lifecycle.
---

# logscale_saved_query (Resource)

Manages a LogScale saved query through the YAML-template lifecycle.

See the matching `Usage Examples` and `Important Behavior Notes` sections of [`README.md`](../../README.md) for the current schema, lifecycle constraints, and tenant-validation caveats.

## Behavior notes

- Import format is `view_name:saved_query_id`. Labels are managed through dedicated mutations.

## Schema

Generated schema details will be populated when `terraform-plugin-docs` is wired into the build. Until then, the authoritative schema is the corresponding Go file under [`internal/provider/`](../../internal/provider/).
