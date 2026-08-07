# LogScale Terraform Provider Expansion Prompt Spec

## Purpose

This document is the living prompt/spec for expanding this custom Terraform provider for Falcon LogScale cloud. It is intended to be revised as we learn more about the GraphQL API, confirm SaaS-specific behavior, and decide how opinionated we want the provider to be.

The near-term goal is to make the provider as complete as practical for LogScale ingestion-related management, while still supporting adjacent resources that are required to make ingestion usable in real environments.

The broader goal is to grow the provider toward a detection-as-code friendly surface area as well, including alerts, triggers, and other operational or detection-centric LogScale objects that can be safely and honestly managed through Terraform.

## Project Context

- Repository: custom Terraform provider for LogScale using the Terraform Plugin Framework and LogScale GraphQL API.
- Current provider focus: repositories, ingest tokens, groups, and role assignments.
- Important constraint: some LogScale resources are not truly CRUD via API. The provider should still do as much real management as possible and clearly warn when a change must be completed manually in the UI or by another operational process.
- Deployment target: LogScale cloud/SaaS behavior should be treated as the source of truth over broader platform documentation when those differ.

## Source Material

### Local, already validated in this repo

- Provider implementation under `internal/provider/`
- In-repo research wiki under `reference/`
- Local validation examples under `terraform-test-project/`

### External references

- GraphQL queries: https://library.humio.com/logscale-graphql-reference-queries/graphql-queries.html
- GraphQL mutations: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutations.html
- GraphQL datatypes: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-datatypes.html

## Current Baseline In This Repo

### Implemented resources

- `logscale_repository`
- `logscale_parser`
- `logscale_view`
- `logscale_dashboard`
- `logscale_saved_query`
- `logscale_lookup_file`
- `logscale_aws_s3_sqs_ingest_feed`
- `logscale_webhook_action`
- `logscale_filter_alert`
- `logscale_aggregate_alert`
- `logscale_scheduled_search`
- `logscale_ingest_token`
- `logscale_group`
- `logscale_group_membership`
- `logscale_organization_role_assignment`
- `logscale_system_role_assignment`

### Implemented data sources

- `logscale_role`
- `logscale_user`
- `logscale_users`
- `logscale_repository`
- `logscale_view`
- `logscale_saved_query`
- `logscale_saved_queries`
- `logscale_lookup_file`
- `logscale_lookup_files`
- `logscale_validate_query`
- `logscale_aws_s3_sqs_ingest_feed`
- `logscale_webhook_action`
- `logscale_webhook_actions`
- `logscale_filter_alert`
- `logscale_filter_alerts`
- `logscale_aggregate_alert`
- `logscale_aggregate_alerts`
- `logscale_scheduled_search`
- `logscale_scheduled_searches`
- `logscale_dashboard`
- `logscale_dashboards`

### Notable current limitations

- Repository update/delete are not implemented.
- Parser v1 is implemented, but currently defers `language_version` and create-only overwrite options.
- View v1 is implemented, but currently treats `name` and `description` as replacement fields while updating connections in place.
- User lifecycle is still not managed directly; only lookup/list data sources are implemented.
- Group membership now reads remote usernames, but membership mutation behavior still needs tenant validation and may require manual verification when usernames do not reconcile cleanly.
- Ingest token read/update behavior is limited.
- GraphQL request/response handling is repeated resource-by-resource rather than centralized.
- The provider still does not yet expose roles or broader ingestion assets even though local GraphQL probes show some of that API surface is usable.
- Lookup files now have a first implementation, but their whole-file replacement behavior still needs tenant validation because the upstream mutation surface is row/edit oriented.

## Confirmed/Observed API Surface So Far

The following capabilities are already evidenced either by historical GraphQL probes or by LogScale GraphQL docs and should be treated as strong candidates for implementation:

### Repositories

- `createRepository`
- `repository(name: ...)`
- No confirmed working delete path from current repo experiments

### Ingest tokens

- `addIngestTokenV3`
- `removeIngestToken`
- `assignParserToIngestTokenV2` should be investigated for parser reassignment support

### Parsers

- `createParserV2`
- `updateParserV2`
- `deleteParserV2`
- `testParserV2`

Confirmed parser-oriented input surface from docs includes:

- `repositoryName`
- `name`
- `script`
- `testCases`
- `fieldsToTag`
- `fieldsToBeRemovedBeforeParsing`
- `allowOverwritingExistingParser`
- `languageVersion`

### Views

- `createView`
- `updateView`
- delete path via search-domain mutations such as `deleteSearchDomain()` / `deleteSearchDomainById()`

Confirmed view-oriented input surface from historical probes/examples includes:

- `name`
- `description`
- `connections[]`
- `connections[].repositoryName`
- `connections[].filter`
- `connections[].languageVersion`

### Existing access control support

- Group CRUD
- Organization role assignment
- System role assignment
- Role CRUD and default-role mutation exist in historical GraphQL probes and may be worth implementing later, but they are not the first ingestion-focused priority

## Guiding Principles For Expansion

1. Prefer real API enforcement over state-only modeling.
2. When LogScale does not support full CRUD, model the resource honestly and warn clearly.
3. Optimize for cloud/SaaS behavior rather than on-prem assumptions.
4. Expose rich schema when the API supports it, especially for parser test cases and ingestion-specific settings.
5. Use Terraform semantics that make limitations obvious:
   - `RequiresReplace` when the API only supports create/delete
   - warnings when update/delete must be manual
   - sensitive handling for secrets/tokens
   - import support wherever read semantics are good enough
6. Avoid pretending drift can be reconciled when it cannot.
7. Revisit existing resource methodologies and schemas regularly to identify fields, behaviors, and modeling patterns we should add now to better support future expansion.
8. Keep the changelog current as part of normal delivery work so releases have durable, curated notes instead of relying on memory after the fact.

## Research Wiki Requirement

All meaningful research performed for this provider should be summarized in the in-repo reference wiki under `reference/`.

### Required structure

- a central index at `reference/README.md`
- major-category indexes under `reference/<category>/README.md`
- topic-specific notes in separate `.md` files under those categories
- summary text in each category README that points to deeper documents

### Required behavior

- the wiki must remain documentation-only and stay outside source/build paths
- research notes should capture both official-doc findings and local tenant-validation notes
- research notes should also capture important runtime lessons from bug fixes, tenant-specific behavior, and Terraform state-model failures
- documents should distinguish between confirmed behavior, probable behavior, and open questions
- resource families should record external dependencies or referenced assets that may not be provider-managed
- when a workstream progresses, the spec and the relevant wiki pages should both be updated
- when scope is narrowed for practicality, record the selected first implementation slice so deferred families remain explicitly intentional

## Progress Tracking Requirement

The prompt spec should track actual progress, not just desired roadmap items.

### Current progress snapshot

- parser research wiki initialized
- view research wiki initialized
- user research wiki initialized
- action research wiki initialized
- trigger research wiki initialized
- dashboard research wiki initialized
- ingest-feed research wiki initialized
- cross-cutting external-reference wiki initialized
- cross-cutting runtime-lessons wiki initialized
- cross-cutting methodology-review wiki initialized
- `logscale_parser` v1 implemented in provider code
- `logscale_view` v1 implemented in provider code
- `data.logscale_user` and `data.logscale_users` implemented in provider code
- `data.logscale_repository` and `data.logscale_view` implemented in provider code
- `logscale_dashboard` implemented in provider code
- `logscale_saved_query` implemented in provider code
- `logscale_lookup_file` implemented in provider code
- `logscale_aws_s3_sqs_ingest_feed` implemented in provider code
- `data.logscale_aws_s3_sqs_ingest_feed` implemented in provider code
- `logscale_webhook_action` implemented in provider code
- `logscale_filter_alert` implemented in provider code
- `logscale_aggregate_alert` implemented in provider code
- `logscale_scheduled_search` implemented in provider code
- dashboard lookup/list data sources implemented in provider code
- saved-query lookup/list data sources implemented in provider code
- lookup-file lookup/list data sources implemented in provider code
- query validation data source implemented in provider code
- aggregate-alert and scheduled-search lookup data sources implemented in provider code
- plural detection-oriented list data sources implemented for webhook actions, filter alerts, aggregate alerts, and scheduled searches
- AWS S3/SQS ingest feed CRUD and lookup support implemented in provider code
- parser/view follow-up work remains for richer schema coverage and acceptance validation
- dashboard CRUD is implemented through the YAML-template lifecycle, while richer nested dashboard schema modeling remains deferred
- saved-query v1 is implemented through the YAML-template lifecycle, while tenant validation and any later non-template schema refinement remain open
- lookup-file v1 is implemented as a CSV-snapshot resource, while tenant validation of whole-file replacement semantics remains open
- trigger follow-up now centers on tenant validation and any additional alert/action types rather than first-wave CRUD
- existing provider methodology review still in progress
- `logscale_view_role_assignment` implemented as the view-scoped sibling of the org/sys role-assignment resources (assign/remove + state-preserving Read + `RequiresReplace` on all three IDs); unit tests now cover the ID-format helper, schema shape, and Update-not-supported diagnostic
- `logscale_role` implemented with full CRUD via `createRole` / `updateRole` / `removeRole`; `view_permissions` modelled as a string set; import by role ID
- `logscale_default_role_assignment` implemented via `updateDefaultRole`; Read preserves state and destroy emits a warning + detaches state (no clean unset mutation upstream)
- `data.logscale_view_role_assignments` implemented for listing a group's view-scoped assignments via `group(groupId).roles`; entries without a search domain are filtered out so the surface stays view-scoped
- `tfplugindocs`-shaped `docs/` tree added with `docs/index.md`, `docs/resources/*.md`, and `docs/data-sources/*.md` covering all 19 resources and 22 data sources; the role-family additions are authored in full, the rest are stubs ready to be overwritten by `terraform-plugin-docs generate`
- agent routing layer added: `AGENTS.md`, `CLAUDE.md`, `CHANGE-HISTORY.md`, and area-scoped `AGENTS.md` files under `internal/provider/`, `reference/`, `terraform-test-project/`, and `.github/`
- GitHub Actions workflows are intentionally absent in this repository; build, test, and release are maintainer-local (see `RELEASING.md`)

## Release Notes Requirement

Changelog management is now an expected part of project maintenance.

### Required behavior

- update `CHANGELOG.md` as meaningful work lands, not only at the end of a long cycle
- ensure the target release version has a curated section in `CHANGELOG.md` before bumping `VERSION`
- keep `Unreleased` accurate between tagged versions
- reflect major provider/resource additions, lifecycle changes, workflow changes, and important operational guidance changes

### Why this matters

- releases in this repository now publish curated notes from `CHANGELOG.md`
- the changelog is part of the project’s durable delivery record
- keeping it current reduces release friction and helps future maintainers understand what changed and why

## Review And Backfill Requirement For Existing Resources

This effort should not focus only on net-new resources. We should also review the provider patterns and resource schemas already implemented in this repository to determine whether they are missing fields, metadata, validation hooks, import behavior, or lifecycle semantics that would better serve future needs.

### Review goals

- identify GraphQL fields we are currently ignoring but should capture in state or expose in schema
- identify resource attributes that should become nested blocks, sets, or richer structures rather than simple strings
- identify places where current create/read/update/delete behavior is too narrow for future use cases
- identify one-time-returned values, IDs, names, and relationships we should preserve for future cross-resource composition
- identify places where plan modifiers, warnings, or import support should be improved
- identify repeated implementation patterns that should be centralized before we add more resources

### Existing resource families that should be reviewed explicitly

- repositories
- ingest tokens
- groups
- group memberships
- organization role assignments
- system role assignments
- existing data sources

### Review lens

For each existing resource or data source, ask:

- are there API fields we should expose now so we do not paint ourselves into a corner later?
- are there identifiers or relationships that future resources will need to reference?
- are there optional settings we omitted only because the first implementation was narrow?
- are there read-path fields we should add to improve drift detection or imports?
- are there lifecycle limitations we should communicate more clearly in schema, docs, or diagnostics?
- are there patterns in historical GraphQL probes or the GraphQL docs that suggest the current implementation is under-modeled?

## External Reference Modeling Requirement

Some resources will need to reference identities or assets that are real in LogScale but are not created by this provider.

Examples:

- group memberships reference users even though users are not provider-managed here
- alerts may reference actions that may exist already
- views reference repositories through connections
- ingest tokens may reference parsers by name or ID

This must be treated as a first-class design concern in both research and implementation.

### Design expectations

- document explicitly when a resource references but does not own another object
- prefer schema names that make the reference shape clear, such as `_id`, `_name`, `_ids`, or `_ids_or_names`
- preserve stable identifiers returned by the API for future composition
- add data sources where they materially improve composition with externally managed assets
- avoid implying lifecycle ownership when the provider only manages the reference

## Provider Behavior Rules For Non-CRUD Resources

These rules should govern future implementation unless we intentionally revise them:

### If create/read/update/delete all exist and work

- Implement standard CRUD resource behavior.

### If create/read/delete exist but update does not

- Treat mutable attributes as replacement fields when practical.
- If replacement is not appropriate for user workflows, consider a documented advisory mode later, but do not silently fake updates.

### If create/read/update exist but delete does not

- Support create/read/update normally.
- On destroy, remove from Terraform state and emit a warning that manual deletion is required in LogScale.
- Document that Terraform destroy only detaches state for that resource.

### If create exists but read is partial or weak

- Persist only attributes we can reliably know.
- Mark non-readable or one-time-returned values as sensitive/computed and document that they may only be available at create time.
- Avoid inventing state from assumptions.

### If update cannot be enforced through the API but we still want config tracked

- Only do this intentionally.
- Emit a warning during apply that Terraform state is being updated but the remote object must be changed manually.
- Use this pattern sparingly and explicitly in docs.

## Recommended Implementation Shape

### Core refactor first

Before adding many more resources, build some shared infrastructure:

- centralized GraphQL request helper
- consistent GraphQL error parsing
- reusable OAuth client setup
- common diagnostics helpers for:
  - not found handling
  - unsupported delete/update warnings
  - manual-reconciliation messaging
- shared import ID parsing helpers where useful

### Resource/data source naming

Use the `logscale_*` naming pattern already present in the provider.

### Schema design preferences

- Prefer explicit nested objects/lists over opaque JSON strings unless the GraphQL datatype is too large or unstable.
- Preserve exact user intent for parser scripts and test cases.
- Treat token material as sensitive.
- Expose both stable identifiers and user-facing names where available.
- Avoid reserved Terraform root attribute/block names when designing schemas, since an invalid root name can prevent the provider schema from loading at all.

## Proposed Feature Roadmap

### Phase 0: Hardening and foundations

- Refactor common GraphQL client/request code.
- Improve diagnostics and warnings for partial CRUD behavior.
- Revisit current resources for correctness:
  - repository
  - ingest token
  - group membership
  - role assignments
- Review existing schemas and methodologies for omitted fields or structures that would better support future needs.
- Add or improve import behavior where feasible.
- Keep `reference/` and this spec updated as research and implementation progress.

### Phase 1: Core ingestion assets

Priority resources:

- `logscale_parser`
- `logscale_view`
- improved `logscale_ingest_token`
- improved `logscale_repository`

Priority data sources:

- `logscale_repository`
- `logscale_parser`
- `logscale_view`
- possibly `logscale_ingest_token` if read semantics are adequate without exposing secret material

### Phase 2: Ingestion workflow completeness

Investigate and implement if confirmed for cloud:

- parser reassignment for ingest tokens
- parser delete behavior
- additional parser metadata fields if queryable
- additional view connection management behavior
- validation-oriented data sources or helper resources where Terraform can add value
- deeper ingest-feed validation/test integration if `testAwsS3SqsIngestFeed` proves safe and useful in normal workflows

### Phase 3: Cloud ingestion integrations

Only after validation in docs plus real cloud behavior:

- ingest listeners
- AWS S3/SQS ingest feeds
- cloud-native ingest feeds/integrations
- log collector configurations
- log collector groups
- any SaaS-supported ingestion configuration objects relevant to your environment

These should be gated on actual cloud availability and manageable CRUD semantics, not just presence in the docs.

Concrete examples that should be investigated explicitly for LogScale cloud:

- AWS S3 ingest feed configurations
- AWS SQS-backed ingest feed configurations
- collector-side configuration objects
- collector grouping/fleet-management objects

### Phase 4: Detection-as-code and operational content

Investigate and implement where supported for LogScale cloud:

- alerts
- triggers
- saved searches or scheduled search constructs if applicable
- views or other reusable query/grouping objects that materially support detections
- notification or action wiring related to detections, if exposed in a Terraform-manageable way

These should be evaluated through the same lens as ingestion resources:

- cloud/SaaS support first
- honest CRUD semantics
- high-value operational use cases
- good fit for version-controlled detection content

## Detection-As-Code Candidate Resource Families

As of March 30, 2026, the official LogScale GraphQL docs indicate that the strongest detection-as-code candidates are not just a generic "trigger" resource, but several concrete trigger-style resource families with modern create/update/delete support.

### Recommended first-wave detection resources

- `logscale_filter_alert`
- `logscale_aggregate_alert`
- `logscale_scheduled_search`

These appear to be the most promising Terraform resources because the official docs show modern create/update/delete mutation paths and queryable relationships from repositories or views.

### Candidate supporting resources

- `logscale_webhook_action`
- additional action resources if they are useful in your environment, such as email, OpsGenie, VictorOps/Splunk On-Call, or Slack-style actions if supported in SaaS
- action-oriented data sources for resolving existing actions by ID or name

These are important because the modern alert and scheduled-search datatypes use action references such as `actionIdsOrNames`, which makes actions a likely prerequisite for a practical detection-as-code workflow.

Provider design assumption:

- actions and alert/scheduled-search resources should be built in tandem
- we should not assume pre-existing actions will be easy or reliable to reference by hand
- first-class Terraform action resources are preferred over a reference-only initial model

### Terminology note

In LogScale documentation, "triggers" appears to be an umbrella capability area rather than a single obvious Terraform resource shape. For provider design, we should prefer explicit resources for the underlying object types:

- filter alerts
- aggregate alerts
- scheduled searches
- actions/notifiers

Rather than beginning with a vague `logscale_trigger` abstraction, we should only add a generic trigger wrapper if the API model later proves that abstraction is both stable and helpful.

### First-pass detection resource targets

#### `logscale_filter_alert`

Why it is a good fit:

- official docs show `createFilterAlert()`
- official docs show `updateFilterAlertV2()`
- official docs show `deleteFilterAlertV2()`
- repository queries expose `filterAlerts`

Schema candidates to investigate:

- `id` computed
- `view_name` required
- `name` required
- `description` required or optional based on confirmed mutation behavior
- `query_string` required
- `enabled` optional/computed
- `labels` optional set/list(string)
- `action_ids_or_names` required set/list(string)
- `query_ownership_type` required
- `run_as_user_id` optional
- `throttle_time_seconds` optional
- `throttle_fields` optional list(string)

Design notes:

- prefer `throttle_fields` over deprecated single `throttle_field`
- treat action references carefully to avoid unnecessary diffs when order does not matter
- consider import support via `view_name:id`

#### `logscale_aggregate_alert`

Why it is a good fit:

- official docs show `createAggregateAlert()`
- official docs show `updateAggregateAlertV2()`
- official docs show `deleteAggregateAlertV2()`
- repository queries expose `aggregateAlerts`

Schema candidates to investigate:

- `id` computed
- `view_name` required
- `name` required
- `description` optional
- `query_string` required
- `enabled` required or optional/computed depending on API defaults
- `labels` optional set/list(string)
- `action_ids_or_names` required set/list(string)
- `query_ownership_type` required
- `run_as_user_id` optional
- `query_timestamp_type` required
- `search_interval_seconds` required
- `throttle_time_seconds` required
- `throttle_field` optional
- `trigger_mode` optional

Design notes:

- validate interval constraints locally if they are stable enough from docs
- watch for view-versus-repository semantics in read/import behavior
- confirm whether `throttle_field` remains singular here or whether newer multi-field support exists
- implementation note: current docs still present a create/update asymmetry here, so v1 should be conservative on create if the create mutation only accepts a single throttle field

#### `logscale_scheduled_search`

Why it is a good fit:

- official docs show `createScheduledSearchV2()`
- official docs show `updateScheduledSearchV3()`
- official docs show `deleteScheduledSearchV2()`
- scheduled searches align well with detection-as-code and operational automation

Schema candidates to investigate:

- `id` computed
- `view_name` required
- `name` required
- `description` optional
- `query_string` required
- `enabled` optional/computed
- `labels` optional set/list(string)
- `action_ids_or_names` required set/list(string)
- `query_ownership_type` required
- `run_as_user_id` optional
- `schedule` required
- `time_zone` required
- `search_interval_seconds` required
- `search_interval_offset_seconds` conditionally required
- `query_timestamp_type` required
- `backfill_limit` conditionally required
- `max_wait_time_seconds` conditionally required
- `trigger_on_empty_result` optional

Design notes:

- this resource likely needs the strongest schema validation because several fields are conditionally required based on timestamp/query mode
- if Terraform validation becomes too noisy, we can start with server-side validation plus clear diagnostics and harden local validators later
- implementation note: if docs disagree internally on `queryString` vs `querystring`, prefer the mutation example and returned field naming that work in practice
- implementation note: `IngestTimestamp` scheduled-search behavior is documented as depending on the `ScheduledSearchIngestTimestamp` feature flag, so cloud-tenant validation remains important even though the CRUD surface itself is now implemented

#### `logscale_webhook_action`

Why it is a good fit:

- official docs show `createWebhookAction()`
- webhook actions are broadly useful and easy to connect to alerts/scheduled searches
- webhook is likely the best first action type because it maps cleanly to automation systems

Schema candidates to investigate:

- `id` computed
- `view_name` required
- `name` required
- `url` required
- `method` required
- `header` repeated nested block
- `header.header` required
- `header.value` required
- `body_template` required
- `ignore_ssl` required or optional/computed depending on API defaults
- `use_proxy` required or optional/computed depending on API defaults
- any secret/sensitive fields should be marked sensitive if the API returns or accepts them

Design notes:

- action resources may need type-specific implementations rather than a single generic `logscale_action`
- if action reads do not return secret-equivalent fields exactly, plan modifiers and drift rules will need care

### Detection-oriented data sources to consider

- `logscale_filter_alert`
- `logscale_aggregate_alert`
- `logscale_scheduled_search`
- `logscale_action`
- `logscale_webhook_action`

These would help teams compose detections from partially managed objects and support migration/import workflows.

### Detection implementation priority

Recommended order after the ingestion foundation is in better shape:

1. `logscale_webhook_action` plus `logscale_filter_alert`
2. `logscale_aggregate_alert`
3. `logscale_scheduled_search`
4. additional action types based on your actual notification stack

Implementation note:

- the first alert resource should be developed alongside at least one action resource so end-to-end detection workflows are Terraform-manageable from day one
- webhook actions are the preferred first action type unless your environment depends more heavily on a different notifier

### Detection open questions

- Which of these objects are fully supported in your LogScale cloud tenant and API permission model?
- Which action types matter most in your environment after webhook support?
- Do we want to support detection packaging/template workflows later, or keep the provider focused on first-class GraphQL resources?
- Should we treat repository-backed and view-backed detections uniformly, or expose that distinction directly in schema and docs?

## Additional Candidate Ingestion Resource Families

Beyond repositories, parsers, views, and ingest tokens, the spec should explicitly include the following cloud-ingestion candidates where supported:

- AWS S3 ingest feed resources
- AWS SQS ingest feed resources
- log collector configuration resources
- log collector group resources

These should be treated as first-class candidates for later phases, especially where they help teams manage ingestion pipelines as code rather than splitting ownership between Terraform and manual UI configuration.

## Resource Specs To Target First

### `logscale_parser`

Desired capabilities:

- create
- read
- update
- delete if SaaS validation succeeds
- import if parser ID or a repo/name tuple can be resolved reliably

Desired schema candidates:

- `id` computed
- `repository_name` required
- `name` required
- `script` required
- `fields_to_tag` optional list(string)
- `fields_to_be_removed_before_parsing` optional list(string)
- `allow_overwriting_existing_parser` optional bool
- `language_version` optional nested object or string, depending on datatype complexity
- `test_case` repeated nested block

Desired test-case shape:

- `event.raw_string`
- `output_assertion.output_event_index`
- assertion structures for expected field/value pairs

Open questions:

- How much of the parser test/assertion model should be expressed natively versus simplified for v1?
- Can we query back test cases exactly enough to avoid perpetual diffs?

### `logscale_view`

Desired capabilities:

- create
- read
- update
- delete only if a real endpoint is confirmed; otherwise manual-delete warning behavior

Desired schema candidates:

- `id` computed
- `name` required
- `description` optional
- `connection` repeated nested block
- `connection.repository_name` required
- `connection.filter` required

Open questions:

- Is `name` immutable after creation in practice?
- Can all connection details be read back consistently from SaaS?

### `logscale_ingest_token`

Desired improvements:

- better read behavior
- optional parser reassignment/update if `assignParserToIngestTokenV2` is viable
- clear handling for one-time token secret value
- improved import story for non-secret metadata

Schema candidates beyond today:

- `repository_name`
- `name`
- `parser`
- `token` computed sensitive, create-time only where applicable
- possibly IDs for associated parser metadata if useful

Open questions:

- Can token metadata be queried reliably without the token secret?
- Is parser reassignment a true update or effectively replacement?

### `logscale_repository`

Desired improvements:

- strengthen read behavior
- support replacement semantics for non-updatable attributes
- support manual-delete warning flow if deletion remains unavailable
- expand schema only where cloud behavior is known and queryable

Schema candidates:

- `id`
- `name`
- `description`
- `type`
- `retention_days`

Open questions:

- Which repository attributes are mutable in SaaS, if any?
- Is there any supported API path for safe deletion in cloud?

## Documentation Requirements For Every New Resource

Each resource should document:

- what is truly managed remotely
- what is only tracked in state
- what requires manual action
- import format
- known permission requirements
- any one-time-returned values
- examples for common usage

## Validation And Testability Requirements

For every resource family we add, we should investigate whether LogScale exposes GraphQL query, validation, preview, or test capabilities that can improve Terraform safety, diagnostics, or authoring quality.

### General rule

- if a resource type has a meaningful GraphQL-backed validation or test endpoint, we should try to incorporate it into the provider design
- if it is not suitable for normal CRUD lifecycle execution, we should still consider data sources, pre-apply validation behavior, or acceptance-test utilities that use it
- if a validation/test endpoint is too expensive, too side-effectful, or too permission-sensitive for normal applies, document that and keep it out of the default CRUD path

### Examples already visible from current research

- query validation via `validateQuery`
- parser test-case support built into parser-oriented GraphQL inputs

### How to use validation capabilities in the provider

Depending on the resource type and API behavior, possible patterns include:

- plan-time validation where local schema rules are insufficient and the framework permits safe validation
- apply-time validation before create/update mutations
- dedicated validating data sources
- acceptance/manual test helpers for resources with complex behavior
- rich diagnostics that return GraphQL validation feedback directly to the user

### Resource families where validation/test support should be investigated explicitly

- parsers
- views
- ingest tokens and parser assignment flows
- alerts
- scheduled searches
- action resources
- AWS S3/SQS ingest feeds
- log collector configurations
- log collector groups

### Design guardrails

- validation features should improve confidence, not create hidden side effects
- test/preview calls should never silently mutate remote objects
- Terraform state should not depend on ephemeral test results unless the API treats them as durable resource attributes
- where test execution requires representative sample events or payloads, the schema should make that explicit rather than burying it in opaque strings

## Testing Expectations

### Unit/provider-level

- request/response parsing
- diagnostic behavior for GraphQL errors
- import ID parsing
- plan/state behavior for partial CRUD resources
- local validation logic for resource-specific rules
- handling of GraphQL-backed validation/test responses where integrated

### Acceptance/manual

- validate against LogScale cloud
- test create/read/update/delete where supported
- test destroy behavior for manual-delete resources
- test import for at least one happy path per resource
- verify no perpetual diff for stable resources
- exercise any GraphQL-backed validation/test capability that is part of the resource workflow
- explicitly verify AWS S3/SQS feed, collector, and detection resources only where the target tenant and permissions support them

## Explicit Non-Goals For The First Expansion Wave

- Covering every GraphQL mutation just because it exists
- Supporting on-prem-only features unless they also matter for cloud use
- Hiding API limitations behind misleading Terraform behavior
- Building broad non-ingestion feature families before core ingestion assets are solid

Detection-as-code resources are in scope for the overall program, but not at the expense of first making the ingestion foundation and shared provider behavior reliable.

## Working Prompt For Future Implementation Sessions

Use this repository to expand the custom Terraform provider for Falcon LogScale cloud. Prioritize ingestion-related resources and adjacent dependencies needed for practical ingestion management. Before implementing a resource, verify the GraphQL create/read/update/delete story using both local tenant validation and the LogScale GraphQL docs. Prefer real CRUD behavior, but when the API does not support full lifecycle management, implement the strongest honest Terraform behavior possible and emit clear warnings for manual steps. Keep schemas detailed where the API supports stable datatypes, especially for parser configuration and parser test cases. Refactor common GraphQL logic as needed so new resources share consistent request handling, diagnostics, imports, and partial-CRUD behavior.

Also evaluate detection-as-code style capabilities for Falcon LogScale cloud, including alerts, triggers, and other reusable detection-centric objects. Add them when they are genuinely useful, supported in SaaS, and can be modeled honestly in Terraform without hiding important lifecycle limitations.

As part of the expansion, review the methodologies and schemas already implemented in this provider to identify missing fields, metadata, relationships, validation hooks, and lifecycle behaviors that should be added now to better support future resource families and long-term maintainability.

Maintain the in-repo research wiki under `reference/` as a central knowledge base. Organize it by major category, summarize each subdocument from a category README, and capture both official-doc findings and local validation notes. Treat external references and non-provider-managed dependencies as first-class modeling concerns and keep the spec updated with actual progress as implementation advances.

Treat `CHANGELOG.md` maintenance as part of the normal workflow. Significant implementation, workflow, security, and documentation changes should update the changelog as they land so releases can publish curated notes without a last-minute reconstruction effort.

## Decision Log

Track important choices here as we refine the plan:

- Initial focus is ingestion-related resources for LogScale cloud.
- Alerts, triggers, and similar detection-as-code resources are part of the target long-term scope.
- AWS S3/SQS ingest feeds and log collector configuration/group resources are explicitly in scope for later ingestion phases.
- Partial/non-CRUD resources should be modeled honestly, with warnings rather than hidden behavior.
- Parsers and views are the first major additions after foundational cleanup.
- Dashboards, actions, and triggers are active expansion areas, with webhook actions and filter alerts already implemented and dashboard CRUD now added through the YAML-template lifecycle.
- Aggregate alerts and scheduled searches are now implemented as first-wave trigger resources, alongside lookup data sources for both families.
- A fresh official-doc sweep on April 20, 2026 reconfirmed the aggregate-alert and scheduled-search CRUD surface and also highlighted trigger-permission requirements, label limits, UTC-offset-only scheduled-search time zones, and the `ScheduledSearchIngestTimestamp` feature-flag caveat for `IngestTimestamp` scheduled searches.
- Saved queries are now implemented through the YAML-template lifecycle, following the same methodology used for dashboards where template-based CRUD is more honest than a thin partial schema.
- Lookup files are now implemented as CSV-snapshot resources and data sources, but should still be treated as an area requiring tenant validation because the upstream file-edit mutations are lower-level than the provider's whole-file model.
- Existing resources should be reviewed for missing fields and under-modeled schema before or alongside major expansion work.
- The in-repo `reference/` directory is the canonical research wiki for ongoing provider expansion work.
- As of April 6, 2026, parser/view research has started in the wiki and v1 implementations for both resources have now been added to the provider.
- `logscale_parser` currently supports CRUD, import by `repository_name:parser_id`, field/tag modeling, and parser test-case blocks, while deferring some richer parser options for later.
- `logscale_view` currently supports CRUD, import by name, and connection management, while treating `name` and `description` as replacement semantics in v1.
- `data.logscale_user` and `data.logscale_users` currently provide user lookup/list support without implying user lifecycle ownership.
- `logscale_group_membership` now reads remote group users and warns on reconciliation mismatches, but it still should be treated as an area needing real tenant validation.
- `logscale_dashboard` currently supports CRUD, import, and richer lookup behavior through the official template-based dashboard lifecycle, while explicit nested widget/section modeling remains a later enhancement.
- `logscale_aws_s3_sqs_ingest_feed` currently supports CRUD and lookup for the AWS S3/SQS + IAM-role ingest-feed path, with local validation around the documented compression/authentication/preprocessing options.
- `CHANGELOG.md` is now part of the expected maintenance workflow and should be kept current alongside meaningful changes.
- Agent routing is now codified through `AGENTS.md` (area routing entry point), `CLAUDE.md` (Claude Code MUST rules and provider conventions), area-scoped `AGENTS.md` files, and an append-only `CHANGE-HISTORY.md` provenance log so that future agent sessions can resume with full repo context without re-deriving conventions from source. Material structural / automation / cross-cutting documentation changes `MUST` add a new entry at the top of `CHANGE-HISTORY.md`.
- This repository has no GitHub Actions workflows; releases are cut manually per `RELEASING.md`.

## Open Questions To Resolve Together

- Should we keep the provider strictly honest with replace/manual-warning behavior, or do you want an optional advisory/state-tracking mode for some resources?
- How deep do you want parser test-case support in v1: full assertion model, or a smaller practical subset first?
- Do you want roles/default-role resources folded into this expansion now, or kept secondary to ingestion assets?
- Should the next implementation step after agreeing on this spec be a shared GraphQL client refactor, or go straight into `logscale_parser` first?
