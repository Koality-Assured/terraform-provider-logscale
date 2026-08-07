---
page_title: "logscale_ingest_token Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Manages a LogScale ingest token.
---

# logscale_ingest_token (Resource)

Manages a LogScale ingest token.

See the matching `Usage Examples` and `Important Behavior Notes` sections of [`README.md`](../../README.md) for the current schema, lifecycle constraints, and tenant-validation caveats.

## Behavior notes

- Read preserves Terraform state because the provider has no remote token readback. The `token` value is sensitive and only returned at create time.

## Schema

Generated schema details will be populated when `terraform-plugin-docs` is wired into the build. Until then, the authoritative schema is the corresponding Go file under [`internal/provider/`](../../internal/provider/).
