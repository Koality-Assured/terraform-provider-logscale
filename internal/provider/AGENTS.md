# AGENTS — `internal/provider/`

This directory holds the production provider code that ships in the release binary. Treat it as the most behavior-load-bearing area in the repo.

## Conventions

- **Framework**: Terraform Plugin Framework (`hashicorp/terraform-plugin-framework`). Do not introduce SDKv2 patterns.
- **File layout**: one resource per `resource_<noun>.go`, one data source per `data_source_<noun>.go`. Tests live alongside as `<noun>_test.go` (resource/family-scoped) or `<area>_test.go` (cross-cutting helpers).
- **Shared GraphQL surface**: prefer `graphql_helpers.go`, `asset_helpers.go`, `state_normalization.go`, and `csv_helpers.go` over open-coding `http.NewRequestWithContext` + `json.Marshal` + error unmarshal blocks. Phase-0 of [`PROMPT_SPEC.md`](../../PROMPT_SPEC.md) calls out centralizing the request/error pattern further; new resources should not regress that direction.
- **Schema discipline**:
  - `Sensitive: true` on any secret / token-like attribute
  - `Computed: true` + `UseStateForUnknown()` plan modifier for resource IDs
  - `RequiresReplace()` for attributes the API can only set at create time
  - Empty-string vs null normalization handled in `state_normalization.go` — reuse those helpers rather than re-introducing `if x == ""` branches in each resource
- **Import format**: `repository_name:<id>` or `view_name:<id>` where Read is reliable; document in `README.md`.
- **Diagnostics**: prefer `resp.Diagnostics.AddError` / `AddWarning` with a one-line summary + a multi-line detail that quotes the GraphQL `errors[].message` verbatim. Do not swallow GraphQL errors.

## Honest CRUD modeling

When a LogScale resource lacks full CRUD support, follow the rules in [`PROMPT_SPEC.md#provider-behavior-rules-for-non-crud-resources`](../../PROMPT_SPEC.md#provider-behavior-rules-for-non-crud-resources):

- Create + delete only → `RequiresReplace` on mutable attributes, no fake update
- Create + update + no delete → emit a warning on destroy and detach state
- Partial Read → preserve Terraform state in `Read`, document the gap in `README.md`
- One-time-returned secrets → `Sensitive` + `Computed`, document the create-time-only behavior

Do not invent reconciliation paths the API does not support.

## Tests

- Unit tests use `_test.go` files and the standard `testing` package; no `terraform-plugin-testing` acceptance scaffolding is wired up by default.
- **Acceptance tests** that hit a real tenant must be opt-in (env-gated, e.g. `LOGSCALE_ACC=1`). They MUST NOT run from any automation without explicit operator direction.
- New resources should ship with at least minimal helper/state-handling unit tests in the same PR (see existing `aws_s3_sqs_ingest_feed_test.go`, `dashboard_test.go`, `triggers_test.go`, `detection_expansion_test.go` for the established shape).

## Tenant validation

Validate new GraphQL mutations/queries manually against a tenant you own (or against current LogScale GraphQL docs) before wiring them into provider code. Capture confirmed behavior under [`../../reference/provider-expansion/`](../../reference/provider-expansion/). Do not commit credentials, tokens, or tenant identifiers.

## When you touch this directory

- Update the resource/data-source tables and usage examples in [`../../README.md`](../../README.md)
- Update the `## [Unreleased]` section of [`../../CHANGELOG.md`](../../CHANGELOG.md)
- Add or refresh the matching [`../../reference/provider-expansion/<family>/`](../../reference/provider-expansion/) page
- Update [`../../PROMPT_SPEC.md`](../../PROMPT_SPEC.md)'s `Current progress snapshot` if scope shifted
- If you registered a new resource/data source in [`provider.go`](./provider.go), make sure the matching `New<Name>Resource` / `New<Name>DataSource` constructor exists and the matching test file compiles
