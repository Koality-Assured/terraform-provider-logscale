---
page_title: "logscale_lookup_file Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Manages a LogScale lookup file as a whole-file CSV snapshot.
---

# logscale_lookup_file (Resource)

Manages a LogScale lookup file as a whole-file CSV snapshot.

See the matching `Usage Examples` and `Important Behavior Notes` sections of [`README.md`](../../README.md) for the current schema, lifecycle constraints, and tenant-validation caveats.

## Behavior notes

- Import format is `view_name:file_name`. `csv_content` must include the header row. Tenant validation recommended.

## Schema

Generated schema details will be populated when `terraform-plugin-docs` is wired into the build. Until then, the authoritative schema is the corresponding Go file under [`internal/provider/`](../../internal/provider/).
