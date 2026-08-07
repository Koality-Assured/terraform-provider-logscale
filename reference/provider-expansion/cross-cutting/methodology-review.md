# Methodology Review

## Purpose

This note records methodology findings discovered while expanding the provider beyond the original repository/group/token baseline.

It is not just a bug log. The goal is to capture implementation patterns we should favor, patterns we should phase out, and concrete follow-up areas for older resources.

## Current Findings

### Prefer shared GraphQL helpers over per-resource HTTP implementations

Newer resources and data sources use:

- `newConfiguredHTTPClient()`
- `executeGraphQL()`
- shared label helpers
- shared state-normalization helpers

This has been easier to maintain and has produced clearer diagnostics than the older hand-rolled request handling still present in some legacy resources.

Immediate follow-up candidate:

- `logscale_repository` still uses older bespoke HTTP logic and should be refactored to the shared helper path when we revisit it.

### Prefer template-based CRUD when the native GraphQL input graph is very large

For some LogScale asset families, the fully expanded GraphQL input model is real but very large:

- dashboards
- actions in some forms
- saved queries and other templated assets

In those cases, a template/YAML CRUD path is often the more honest and maintainable Terraform v1 shape than a huge, partially modeled nested schema.

Dashboard v1 now follows this rule:

- create: `createDashboardFromTemplateV2`
- update: `updateDashboardFromTemplate`
- delete: `deleteDashboardV3`

This gives Terraform real CRUD behavior without pretending we can safely model the entire dashboard widget/section graph yet.

Saved-query v1 now follows the same rule:

- create: `createSavedQueryFromTemplate`
- update: `updateSavedQueryFromTemplate`
- delete: `deleteSavedQueryV2`

This keeps the first implementation close to how saved queries are actually packaged and reused in detection workflows.

### Lookup data sources should expose composition-relevant fields, not only names and IDs

As the provider grows, lookup data sources are more useful when they expose:

- stable IDs
- owning view/repository information
- resource identifiers returned by LogScale
- labels
- YAML/template content when the resource family is template-driven

This matters because data sources now feed:

- import/migration workflows
- cross-resource references
- validation/lookups in downstream Terraform code

Dashboard lookups were expanded for this reason.

### Favor resource-specific validation when GraphQL docs imply conditional rules

Some LogScale resource families are not hard because the schema is large; they are hard because a smaller set of fields becomes conditionally valid based on mode flags.

Recent examples:

- scheduled searches require different companion fields for `EventTimestamp` versus `IngestTimestamp`
- aggregate alerts expose a create/update asymmetry around throttle-field input shape
- lookup files require a stronger-than-usual assumption that `updateFile()` can be used as a whole-file replacement when given the full desired snapshot

In those cases, a focused resource-specific validator is better than a premature generic abstraction. It keeps diagnostics clearer and lets us codify the exact quirks that matter for one family without spreading that complexity across the provider.

### External-reference modeling must stay explicit

Several resource families depend on assets or infrastructure the provider does not own:

- ingest feeds depend on parsers and AWS infrastructure
- alerts depend on actions
- dashboards depend on owning views/repositories and may be created from externalized YAML

Schema and documentation should keep those references explicit instead of implying the provider owns the whole dependency chain.

### State normalization needs to account for API canonicalization

We have already seen drift from:

- null vs empty string
- null vs empty collection
- reserved Terraform names

Template-bearing assets add another version of the same problem:

- newline and formatting normalization in YAML/template strings

Dashboard v1 now preserves the current YAML string when the API readback is logically equivalent but normalized differently across line endings.

## Follow-Up Backlog

### High-value cleanup

- refactor `logscale_repository` to shared GraphQL/client helpers
- revisit older resources for richer readback fields and import consistency
- add more unit coverage around state normalization and template-based assets

### Resource families where the methodology likely applies next

- saved queries
- additional action types
- files
- some trigger/alert template flows
