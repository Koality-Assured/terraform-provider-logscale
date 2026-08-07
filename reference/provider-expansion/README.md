# Provider Expansion

This category tracks research and implementation notes for expanding the Terraform provider.

## Current Focus

- parsers
- views
- users
- actions
- triggers
- saved queries
- dashboards
- files
- query validation
- ingest feeds
- cross-cutting modeling concerns for external references

## Documents

- [Parsers](./parsers/README.md)
  Summary: GraphQL lifecycle, schema candidates, testability, and implementation notes for `logscale_parser`.
- [Views](./views/README.md)
  Summary: GraphQL lifecycle, schema candidates, query/update/delete behavior, and implementation notes for `logscale_view`.
- [Users](./users/README.md)
  Summary: GraphQL lookup/list behavior, permission implications, and implementation notes for user-oriented data sources.
- [Actions](./actions/README.md)
  Summary: webhook action CRUD, labels, and lookup behavior for action-oriented provider expansion.
- [Triggers](./triggers/README.md)
  Summary: filter alert, aggregate alert, and scheduled search modeling notes, including GraphQL quirks observed during implementation.
- [Saved Queries](./saved-queries/README.md)
  Summary: template-based saved-query CRUD, label handling, and lookup/list behavior for reusable query assets.
- [Dashboards](./dashboards/README.md)
  Summary: dashboard lookup support, CRUD research notes, and why full dashboard modeling remains a heavier follow-up.
- [Files](./files/README.md)
  Summary: lookup-file CRUD, content readback, and CSV-snapshot modeling notes for detection reference data.
- [Query Validation](./query-validation/README.md)
  Summary: GraphQL-backed query preflight support through `validateQuery()`.
- [Ingest Feeds](./ingest-feeds/README.md)
  Summary: AWS S3/SQS ingest-feed CRUD and lookup support, test endpoint research, and ingest-specific modeling notes.
- [External References](./cross-cutting/external-references.md)
  Summary: assets and identities that resources may need to reference even when the provider does not create them directly.
- [Runtime Lessons](./cross-cutting/runtime-lessons.md)
  Summary: bug-fix patterns and tenant/runtime lessons that should inform future resource design and state modeling.
- [Methodology Review](./cross-cutting/methodology-review.md)
  Summary: implementation-pattern findings, template-based CRUD guidance, and cleanup candidates for older resources.

## Progress Snapshot

- Parser research wiki initialized
- View research wiki initialized
- Cross-cutting external-reference guidance initialized
- `logscale_parser` v1 implemented in provider code
- `logscale_view` v1 implemented in provider code
- `data.logscale_user` and `data.logscale_users` implemented in provider code
- `data.logscale_repository` and `data.logscale_view` implemented in provider code
- `logscale_webhook_action` implemented in provider code
- `logscale_filter_alert` implemented in provider code
- `logscale_aggregate_alert` implemented in provider code
- `logscale_scheduled_search` implemented in provider code
- `logscale_saved_query` implemented in provider code
- `logscale_lookup_file` implemented in provider code
- `logscale_aws_s3_sqs_ingest_feed` implemented in provider code
- `data.logscale_aws_s3_sqs_ingest_feed` implemented in provider code
- `logscale_dashboard` implemented in provider code using the YAML-template CRUD path
- `data.logscale_saved_query`, `data.logscale_saved_queries`, `data.logscale_lookup_file`, `data.logscale_lookup_files`, and `data.logscale_validate_query` implemented in provider code
- plural detection-oriented list data sources implemented for webhook actions, filter alerts, aggregate alerts, and scheduled searches
- `data.logscale_webhook_action`, `data.logscale_filter_alert`, `data.logscale_aggregate_alert`, `data.logscale_scheduled_search`, `data.logscale_dashboard`, and `data.logscale_dashboards` implemented in provider code
- cross-cutting runtime-lessons wiki initialized for bug-fix and state-modeling guidance
- dashboard CRUD implemented with template-based lifecycle management, while richer nested schema modeling remains deferred
- ingest-feed follow-up needed for broader feed families, tenant validation, and any safe use of GraphQL test endpoints
- methodology review now documented in the wiki, with `logscale_repository` explicitly identified as a good future refactor target for shared GraphQL helper adoption
- saved-query follow-up now centers on tenant validation and deciding whether any non-template schema should be promoted into first-class writable attributes later
- lookup-file follow-up now centers on tenant validation of whole-file replacement semantics and whether structured row/header schema is worth a later refinement
- parser follow-up needed for `language_version`, create-only overwrite semantics, and acceptance validation
- view follow-up needed for `language_version`, rename/description semantics, and acceptance validation
- action follow-up needed for additional action types and plural/list data sources
- trigger follow-up now centers on tenant validation and any future aggregate/scheduled-search refinements rather than first-wave CRUD
- user follow-up needed for tenant validation and possible future group-membership integration improvements
