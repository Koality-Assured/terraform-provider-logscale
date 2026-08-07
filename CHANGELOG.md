# Changelog

All notable changes to this project should be documented in this file.

The format is based on Keep a Changelog, adapted for this repository's manual release flow.

## [Unreleased]

### Changed

- published this repository under Koality-Assured as a scrubbed public provider home: removed the `aex/` probe tree, removed all GitHub Actions workflows (releases are manual — see `RELEASING.md`), and generalized org-specific agent/docs references

## [0.3.12]

### Fixed

- `logscale_ingest_token`: changing the `parser` field now triggers resource replacement (destroy + create) instead of an in-place update, which the LogScale API does not support. Previously, any `parser` change caused `terraform apply` to fail with `Ingest Token Update Not Supported`.

## [0.3.11]

### Fixed

- `logscale_aws_s3_sqs_ingest_feed`: Read no longer fails hard when the API token lacks the "Manage Cluster" permission. Listing `ingestFeeds` requires that permission; previously this caused `terraform plan` to fail with `GraphQL error: Manage cluster not allowed.` for any feed already in state. Now, if the read returns a Manage Cluster error, existing state is preserved and a warning is emitted. Drift detection requires a token with Manage Cluster; without it, plan and apply will succeed using the locally-known state.

## [0.3.10]

### Fixed

- `logscale_aws_s3_sqs_ingest_feed`: Create and Update no longer call `readAwsS3SqsIngestFeed` after a successful mutation. Listing `ingestFeeds` requires the "Manage Cluster" permission that is not available on standard tokens, causing every create/update to fail even though the feed was applied correctly. State is now populated directly from the plan values plus the ID returned by the mutation. The Read lifecycle (used for `terraform refresh` and drift detection) is unaffected.

## [0.3.9]

### Fixed

- `logscale_aws_s3_sqs_ingest_feed`: removed the entire `awsAuthentication` block from the read/list GraphQL query. All fields in `IngestFeedAwsAuthenticationIamRole` (`roleArn: String!`, `externalId: String!`) are non-null in the LogScale schema. When the token lacks "Manage Cluster" either field being unresolvable cascades null through `awsAuthentication: IngestFeedAwsAuthentication!` → `source: IngestFeedSource!` → `IngestFeed` → the entire `results: [IngestFeed!]!` list, making all feeds invisible to the provider. `role_arn` and `authentication_kind` are now preserved from Terraform state/plan rather than populated from the API. `aws_external_id` remains null.

## [0.3.8]

### Fixed

- `logscale_aws_s3_sqs_ingest_feed`: removed `externalId` from the list/read GraphQL query. `IngestFeedAwsAuthenticationIamRole.externalId` is `String!` (non-null) in the LogScale schema; when the token lacks the "Manage Cluster" permission the server nulls it, which cascades through non-null `awsAuthentication` → `source` → the entire feed result. This caused "Error reading AWS S3/SQS ingest feed after create" even though the v0.3.7 mutation fix succeeded in obtaining an ID. `aws_external_id` attribute is always null for tokens without Manage Cluster.

## [0.3.7]

### Fixed

- `logscale_aws_s3_sqs_ingest_feed`: mutation response selection reduced to `id` only. Fields like `awsAuthentication.externalId` in the mutation response require the Manage Cluster permission and trigger GraphQL error propagation that nulls the entire response object — causing the provider to see an empty ID and report failure even though the feed was created. Create and Update now get only the `id` from the mutation, then call the read path to populate full state.

## [0.3.6]

### Fixed

- `logscale_aws_s3_sqs_ingest_feed`: Create and Update now use a lenient GraphQL execution path that tolerates ancillary permission errors when the mutation returns a valid resource ID. Previously, a `GraphQL error: Manage cluster not allowed.` returned alongside a successfully created feed caused Terraform to discard the data and report failure, leaving the feed orphaned in LogScale state on every apply. Now: if the mutation returns a valid ID, the resource is written to state and the permission error is surfaced as a Terraform warning instead of a hard error. Only fails hard if no ID is returned (genuinely failed create/update).

## [0.3.5]

### Fixed

- `logscale_aws_s3_sqs_ingest_feed`: removed `executionInfo` entirely from both the mutation response and the list/read query. Querying `executionInfo` requires the "Manage Cluster" permission in LogScale; its presence caused a `GraphQL error: Manage cluster not allowed.` that made Create and Read fail even though the feed resource was created successfully. The `status_problem`, `status_cause`, and `status_timestamp` computed attributes are now always null — they were diagnostic-only and do not affect feed management.

## [0.3.4]

### Fixed

- `logscale_aws_s3_sqs_ingest_feed`: fixed `description` field being sent as a nested object `{"description": "..."}` in both Create and Update input payloads instead of a plain string. This caused a `Variable '$input' expected value of type 'CreateAwsS3SqsIngestFeed!'` GraphQL validation error on any feed with a description set.

## [0.3.3]

### Fixed

- `logscale_aws_s3_sqs_ingest_feed`: removed `cause` from the `statusMessage` GraphQL sub-selection. The `IngestFeedStatusCause` type became a complex object on the LogScale API, causing a `must have a sub selection` error on every Create and Read. `status_cause` attribute now returns null until the correct type fragment is determined. All other feed CRUD operations are unaffected.
- `logscale_aws_s3_sqs_ingest_feed`: fixed `description` field being sent as a nested object `{"description": "..."}` in both Create and Update input payloads instead of a plain string. This caused a `Variable '$input' expected value of type 'CreateAwsS3SqsIngestFeed!'` GraphQL validation error on any feed with a description set.

## [0.3.2]

### Added

- added `logscale_view_role_assignment` for granting a group a role on a specific view or repository through the `assignRoleToGroup` / `removeRoleFromGroup` GraphQL mutations; mirrors the assign/remove + state-preserving Read behavior of `logscale_organization_role_assignment` and `logscale_system_role_assignment`, with `RequiresReplace` semantics on `group_id`, `view_id`, and `role_id`
- added `logscale_role` resource with full CRUD via `createRole` / `updateRole` / `removeRole`, including `view_permissions` as a string set and import by role ID
- added `logscale_default_role_assignment` resource for configuring a group's default role via `updateDefaultRole`; Read preserves state and destroy emits a warning + detaches state because the upstream API has no clean unset path
- added `data.logscale_view_role_assignments` for listing a group's view-scoped role assignments via `group(groupId).roles`; entries without a search domain (organization-/system-scoped grants) are filtered out
- added `internal/provider/view_role_assignment_test.go` covering the ID-format helper, schema attributes (Required / Computed / `RequiresReplace` / `UseStateForUnknown` plan modifiers), and the documented Update-not-supported diagnostic
- added agent routing layer: root `AGENTS.md` (area routing entry point), `CLAUDE.md` (Claude Code MUST rules and provider-specific conventions), `CHANGE-HISTORY.md` (append-only AI-assisted change provenance), and area-scoped `AGENTS.md` files under `internal/provider/`, `reference/`, `terraform-test-project/`, and `.github/`
- added a `tfplugindocs`-shaped `docs/` tree (`docs/index.md`, `docs/resources/*.md`, `docs/data-sources/*.md`) covering all 19 resources and 22 data sources; the role, default-role-assignment, and view-role-assignments pages are authored in full, the remainder are stubs that point back to the README's example/caveat sections and will be overwritten when `terraform-plugin-docs` is wired into the build

### Documentation

- documented `logscale_view_role_assignment` in the README resource table, the role-assignment caveats section (renamed to "Organization, System, And View Role Assignments"), and the usage examples
- clarified README caveats for `logscale_ingest_token`, `logscale_organization_role_assignment`, and `logscale_system_role_assignment` so the documented read/update behavior matches the current implementation
- expanded `terraform-test-project/README.md` to cover the implemented group, ingest-token, and role-assignment resources plus their current validation caveats

### Known gaps (tracked, not yet addressed)

- `internal/provider/` still open-codes the GraphQL request/response/error pattern in most resources rather than going through `graphql_helpers.go` / `asset_helpers.go`; Phase 0 of `PROMPT_SPEC.md` still calls for centralization. Explicitly deferred for this release cycle because folding existing resources into a shared client risks regressing working CRUD behavior; the new `logscale_role` resource uses a local `gqlCall` helper as a stepping stone toward eventual centralization.
- `docs/` tree currently uses authored stubs for existing resources/data sources (with frontmatter, behavior notes, and a pointer back to README); full schema-derived content depends on wiring `terraform-plugin-docs` into the build.
- acceptance tests against a real tenant are not yet env-gated/parameterized; `view_role_assignment` and `default_role_assignment` Read paths still preserve state by design, so drift detection follow-up remains open.

## [0.3.1]

### Added

- added `logscale_dashboard` with CRUD and import support through the official dashboard YAML-template lifecycle
- expanded `data.logscale_dashboard` to expose richer dashboard fields needed for composition and template-driven management
- added dashboard-focused unit tests for selection and YAML state-normalization behavior
- added `logscale_aggregate_alert` with CRUD and import support
- added `logscale_scheduled_search` with CRUD and import support
- added `data.logscale_aggregate_alert` and `data.logscale_scheduled_search`
- added trigger-focused unit tests for aggregate-alert and scheduled-search helper behavior
- added `logscale_saved_query` with CRUD and import support through the YAML-template lifecycle
- added `logscale_lookup_file` with CRUD and import support through whole-file CSV modeling
- added `data.logscale_saved_query`, `data.logscale_saved_queries`, `data.logscale_lookup_file`, `data.logscale_lookup_files`, and `data.logscale_validate_query`
- added plural detection-oriented list data sources for webhook actions, filter alerts, aggregate alerts, and scheduled searches
- added CSV helper unit tests for lookup-file content shaping and file-update payload generation

### Changed

- dashboard CRUD now uses `createDashboardFromTemplateV2`, `updateDashboardFromTemplate`, `deleteDashboardV3`, and dedicated dashboard label mutations instead of deferring CRUD entirely
- documented a methodology-review pass in the research wiki, including the decision to prefer template-based CRUD when the native GraphQL input graph is too large for an honest Terraform v1 schema
- scheduled-search validation now enforces the documented `EventTimestamp` versus `IngestTimestamp` companion-field rules before apply
- aggregate-alert create handling now uses the more conservative documented single-field throttle path when create/update docs are not symmetrical
- saved-query CRUD now follows the template-based asset methodology already established for dashboards
- lookup-file v1 now models files as whole CSV snapshots over the lower-level file-edit GraphQL surface
- action and trigger coverage now includes plural list/read data sources to support migration and broader detection composition

### Documentation

- updated the dashboard research notes and root README to reflect the new template-based dashboard resource
- added a cross-cutting methodology-review note to capture implementation patterns and older-resource cleanup candidates
- updated trigger research notes, runtime lessons, the prompt spec, and the root README to reflect aggregate-alert and scheduled-search support plus the GraphQL quirks discovered during implementation
- refreshed the trigger research notes against the latest official docs and captured additional details around trigger permissions, label limits, scheduled-search UTC-offset time zones, and the `ScheduledSearchIngestTimestamp` feature-flag caveat
- added saved-query, files, and query-validation research pages
- updated the expansion spec, methodology notes, runtime lessons, external-reference notes, and root README to reflect saved-query support, lookup-file support, query validation, and the new detection-oriented list data sources

## [0.3.0]

### Added

- added `logscale_webhook_action` with CRUD and import support
- added `logscale_filter_alert` with CRUD and import support
- added `logscale_aws_s3_sqs_ingest_feed` with CRUD and import support
- added `data.logscale_repository` and `data.logscale_view`
- added `data.logscale_aws_s3_sqs_ingest_feed`
- added `data.logscale_webhook_action`, `data.logscale_filter_alert`, `data.logscale_dashboard`, and `data.logscale_dashboards`
- added unit tests for provider registration, shared asset helper logic, and AWS S3/SQS ingest-feed helpers and state handling

### Changed

- added local validation guardrails for AWS S3/SQS ingest feeds so documented compression, authentication, and preprocessing values fail early with clearer diagnostics
- aligned AWS S3/SQS ingest-feed read behavior with the broader repository ingest-feed query shape and narrowed lookup client-side by ID or name

### Documentation

- refreshed the root README to match the current private-provider workflow, implemented resources, and current caveats
- refreshed `terraform-test-project/README.md` so local validation guidance uses `local/logscale` and no longer contradicts current provider behavior
- added a cross-cutting runtime-lessons reference page so recent provider bug-fix patterns are captured in the in-repo wiki
- removed tenant/company-specific references from shared docs and example configuration where practical
- added action, trigger, dashboard, and ingest-feed research pages to document the narrowed expansion workstream and current implementation status
- updated the expansion spec and reference wiki to track AWS S3/SQS ingest-feed progress, external references, and the current decision to document rather than automatically run GraphQL test mutations for feeds

## [0.2.4]

### Changed

- Removed the separate GitHub Actions PR comment workflow so CI no longer attempts to post pass/fail comments on pull requests.
- `logscale_parser` now reads repository parsers through the broader `parsers` field and filters client-side by ID or name, avoiding parser read failures on tenants where `repository.parser(parserid: ...)` is rejected.

### Fixed

- Added broader state-normalization safeguards so omitted optional values are less likely to drift into empty-string or empty-collection state during readback.
- `logscale_repository` now preserves a null optional `description` when LogScale returns an empty string for an omitted description.
- `logscale_view` now preserves an omitted `repository_connection` block as null when the API returns no connections.
- `logscale_parser` now preserves omitted `test_case` structures as null when the API returns no test cases or nested assertions.

## [0.2.3]

### Changed

- The README version badge now comes from a repo-local SVG generated from `VERSION`, making it work reliably in the private repository without depending on Shields' public release metadata.

### Fixed

- `logscale_parser` now preserves null optional set attributes when LogScale returns empty arrays for omitted `fields_to_tag` or `fields_to_be_removed_before_parsing`, preventing inconsistent-result-after-apply failures.

## [0.2.2]

### Fixed

- `logscale_view` now preserves a null `description` in Terraform state when LogScale normalizes an omitted description to an empty string, preventing inconsistent-result-after-apply failures.
- `logscale_parser` now uses the correct `CreateParserInputV2` GraphQL input type during create operations, fixing parser creation failures caused by the invalid `CreateParserV2Input` type name.

## [0.2.1]

### Changed

- GitHub Actions PR validation comments now post from a separate `workflow_run`-based workflow so pull request builds can still report pass/fail details without hitting `Resource not accessible by integration` writeback failures.
- `logscale_view` now uses `repository_connection` instead of the invalid root block name `connection`, fixing provider schema loading for clients upgrading to the newest version.

## [0.2.0]

### Added

- Automated GitHub Actions workflows for CI, release packaging, and CodeQL scanning on the private self-hosted runner pool.
- Automated release flow based on `VERSION` changes on `master`, with Linux and Windows provider artifacts and SHA256 checksum publishing.
- In-repo research wiki under `reference/`, with parser, view, user, and cross-cutting external-reference documentation.
- `logscale_parser` resource with CRUD support, import by `repository_name:parser_id`, field/tag modeling, and nested parser test-case blocks.
- `logscale_view` resource with CRUD support, import by name, and nested connection modeling.
- `data.logscale_user` for exact user lookup by `id`, `username`, `email`, or `display_name`.
- `data.logscale_users` for paginated user listing and search.

### Changed

- `logscale_group_membership` now reads remote usernames from the group query and warns when requested usernames do not reconcile with what LogScale returns.
- Repository documentation, release documentation, and project guidance were refocused on provider development and release automation.
- Release notes are now tracked in-repo through this changelog and published as curated GitHub release notes.

### Security

- Upgraded `google.golang.org/grpc` to `v1.79.3` and refreshed related Go module dependencies to remediate the critical repository vulnerability reported against the prior gRPC dependency chain.

## [0.1.0]

### Added

- Initial LogScale Terraform provider baseline with repository, group, ingest token, organization role assignment, and system role assignment support.
- Initial role data source support for looking up existing LogScale roles.
- Initial provider documentation and local development/test-project material.

### Changed

- `logscale_group_membership` existed as an early experimental resource, but was not yet reliably reconciling remote membership.
- Local/manual provider build workflows and dev overrides were the primary documented consumption path before automated release packaging was introduced.
