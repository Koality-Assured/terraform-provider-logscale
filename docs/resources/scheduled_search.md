---
page_title: "logscale_scheduled_search Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Manages a LogScale scheduled search.
---

# logscale_scheduled_search (Resource)

Manages a LogScale scheduled search.

See the matching `Usage Examples` and `Important Behavior Notes` sections of [`README.md`](../../README.md) for the current schema, lifecycle constraints, and tenant-validation caveats.

## Behavior notes

- Several fields are conditionally required based on timestamp/query mode. `IngestTimestamp` mode depends on the `ScheduledSearchIngestTimestamp` feature flag.

## Schema

Generated schema details will be populated when `terraform-plugin-docs` is wired into the build. Until then, the authoritative schema is the corresponding Go file under [`internal/provider/`](../../internal/provider/).
