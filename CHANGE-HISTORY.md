# Repository change history (high level)

This file is an **append-only, high-level** log of **material repository changes** driven or executed with AI assistance. It is **not** a full commit log, release notes, or research-wiki changelog.

## Who maintains this

Anyone may add entries. **AI agents** working in this repository **`MUST`** add a new entry at the **top** of the entries list when they complete work that **meaningfully** changes structure, navigation, automation, security posture of docs, or large cross-cutting documentation — see [`AGENTS.md`](./AGENTS.md).

## Provenance (include for each entry)

| Field | Guidance |
| --- | --- |
| **Date** | UTC date (or date-time) when the work finished |
| **Requesting user** | Human who asked for the work (account or name as known in session) |
| **AI agent** | Tool and agent identity (e.g. product name, model if explicitly shown) |
| **User request** | Short paraphrase of what the user asked — no transcript |
| **Summary** | Bullet list: **what changed** at a high level only |

Do **not** paste secrets, credentials, customer data, or long prompts.

---

## Entries (newest first)

### 2026-10-09 - Repository beautification with vector identity assets, live badges, and MIT license

| Field | Value |
| --- | --- |
| Date | 2026-10-09 (UTC) |
| Requesting user | Robbie |
| AI agent | Antigravity (Terraform Provider LogScale Beautification Specialist) |
| User request | Beautify repository with vector branding assets, live badges, MIT license, and polished hero section. |
| Summary | • **Vector identity**: Designed and created `assets/terraform-provider-logscale-logo.svg` (512x512) and `assets/terraform-provider-logscale-banner.svg` (1200x360) featuring Terraform purple isometric blocks, Falcon LogScale amber streaming telemetry waveforms, and dark obsidian backings<br>• **License**: Added standard MIT `LICENSE` file for Koality-Assured in the repository root<br>• **README Polish**: Replaced basic header with centered visual banner, logo emblem, tagline, live badges (Terraform Registry, Go 1.21+, Version, MIT License, Conventional Commits), and expanded layout documentation<br>• **Changelog & Documentation**: Updated `CHANGELOG.md` with unreleased branding and docs entries |

---

### 2026-08-07 - Published scrubbed provider under Koality-Assured (no Actions, no aex)


| Field | Value |
| --- | --- |
| Date | 2026-08-07 (UTC) |
| Requesting user | Robbie |
| AI agent | Cursor Grok 4.5 |
| User request | Generalize the provider (remove org-specific references), drop the AEX probe tree, and sync to a new public LogScale Terraform repo on the Koality Assured team with workflows that do not run. |
| Summary | • **New home** [`Koality-Assured/terraform-provider-logscale`](https://github.com/Koality-Assured/terraform-provider-logscale) with a fresh git history<br>• **Removed** the entire `aex/` probe tree and all agent/docs pointers to it<br>• **Removed** all `.github/workflows/` (CI, CodeQL, release, Gemini PR summary) so nothing auto-runs; `.github/` retains badges only<br>• **Rewrote** `RELEASING.md` as a manual tag + GitHub Release procedure<br>• **Scrubbed** org-specific names, emails, runner labels, WIF bindings, and consumer-repo secret references from agent docs, changelog provenance, and research notes |

---

### 2026-05-20 - v0.3.3 + v0.3.4: fix two `logscale_aws_s3_sqs_ingest_feed` GraphQL errors

| Field | Value |
| --- | --- |
| Date | 2026-05-20 (UTC) |
| Requesting user | robert.ketron |
| AI agent | Claude Code (claude-sonnet-4-6) |
| User request | Fix provider errors blocking first-ever SQS ingest feed creation |
| Summary | • **v0.3.3 (`cause` sub-selection):** `IngestFeedStatusCause` changed from scalar to complex type on the live API. Removed `cause` from `statusMessage` query in both mutation response and list/read query. `status_cause` attribute now returns null. See `reference/provider-expansion/ingest-feeds/graphql-surface.md` "Known API Schema Drift" for full context and restoration path.<br>• **v0.3.4 (`description` nested object):** `description` input was constructed as `{"description": "..."}` instead of a plain string, causing a GraphQL type validation error on every Create/Update with a description. Fixed to pass the string directly.<br>• Both fixes documented in `reference/provider-expansion/ingest-feeds/graphql-surface.md` for future maintainers. |

---

### 2026-05-15 - Prepared v0.3.2 release: cut `CHANGELOG.md` section + bumped `VERSION`

| Field | Value |
| --- | --- |
| Date | 2026-05-15 (UTC) |
| Requesting user | robert.ketron |
| AI agent | Claude Code (claude-opus-4-7) |
| User request | Version increment and prepare for release. |
| Summary | • **CHANGELOG**: moved the curated `## [Unreleased]` block (role-family resources, view-role-assignments data source, agent routing, `docs/` tree, view-role-assignment unit tests) into a new `## [0.3.2]` section. Inserted a fresh empty `## [Unreleased]` heading so post-release work has a place to land<br>• **VERSION**: bumped `0.3.1` → `0.3.2`. Patch increment matches the project's established cadence; the new release adds `logscale_role`, `logscale_default_role_assignment`, `data.logscale_view_role_assignments`, the agent-routing layer, and the `docs/` tree<br>• **Release notes**: curated `## [0.3.2]` section is what maintainers publish when tagging the release (see `RELEASING.md`) |

### 2026-05-15 - Closed evaluation punch-list: `logscale_role` + default-role + view-role-assignments data source + tests + `tfplugindocs`-shaped `docs/`

| Field | Value |
| --- | --- |
| Date | 2026-05-15 (UTC) |
| Requesting user | robert.ketron |
| AI agent | Claude Code (claude-opus-4-7) |
| User request | Close the evaluation gaps from the prior pass (skip central GraphQL client and linter): add view-role-assignment tests + list data source, add `logscale_role` and the default-role mutation surface, and build a `tfplugindocs`-shaped `docs/` tree. |
| Summary | • **New resource** [`logscale_role`](./internal/provider/resource_role.go) — full CRUD via `createRole` / `updateRole` / `removeRole`, `view_permissions` modelled as a string set, import by role ID. Resource ships with a local `gqlCall` helper as a stepping stone toward eventual centralization without disrupting the open-coded pattern in existing resources<br>• **New resource** [`logscale_default_role_assignment`](./internal/provider/resource_default_role_assignment.go) — configures a group's default role via `updateDefaultRole`. `group_id` is `RequiresReplace`, `role_id` updates in place. Read preserves state (matches org/sys/view role-assignment pattern). Destroy emits a warning and detaches state because the upstream API has no clean unset path<br>• **New data source** [`data.logscale_view_role_assignments`](./internal/provider/data_source_view_role_assignments.go) — lists a group's view-scoped role assignments via `group(groupId).roles`. Filters out entries without a search domain (those are organization-/system-scoped grants on the sibling resources). Each entry exposes the same synthetic ID format (`group_id:view_id:role_id`) used by `logscale_view_role_assignment`<br>• **New tests** [`internal/provider/view_role_assignment_test.go`](./internal/provider/view_role_assignment_test.go) — covers the extracted `viewRoleAssignmentID` helper, full schema introspection (Required / Computed / `RequiresReplace` / `UseStateForUnknown` plan modifiers via Description-text matching since `stringplanmodifier` types are unexported), and the documented Update-not-supported diagnostic<br>• **New docs tree** [`docs/`](./docs/) — `tfplugindocs`-shaped layout (`index.md`, `resources/*.md`, `data-sources/*.md`) covering all 19 resources and 22 data sources. Role, default-role-assignment, and view-role-assignments pages authored in full; remaining 38 stubs include frontmatter, behavior notes, and a pointer back to the README and the Go source, ready to be overwritten by `terraform-plugin-docs generate` once the tool is wired into the build<br>• **Updated** [`README.md`](./README.md) resource/data-source tables, "Important Behavior Notes" section (new "Role" + "Default Role Assignment" subsections), and usage examples; [`provider.go`](./internal/provider/provider.go) registrations updated; [`provider_test.go`](./internal/provider/provider_test.go) expected-typename lists updated; [`CHANGELOG.md`](./CHANGELOG.md) `## [Unreleased]` and `### Known gaps` updated to reflect what was closed and what was explicitly deferred (central GraphQL client refactor — risk to working CRUD)<br>• **Verified** `go build ./...` and `go test ./internal/provider/...` both green |

### 2026-05-15 - Established agent routing layer

| Field | Value |
| --- | --- |
| Date | 2026-05-15 (UTC) |
| Requesting user | robert.ketron |
| AI agent | Claude Code (claude-opus-4-7) |
| User request | Evaluate the provider for capability/validation/documentation gaps, then adopt an agent routing methodology for the repository. |
| Summary | • **New** [`AGENTS.md`](./AGENTS.md) — repo-level agent entry point. Defines area routing to nested `AGENTS.md` files, repo rules (honest CRUD modeling, untrusted-input handling, changelog discipline, change-history requirement), and workflow triggers tied to existing surfaces (`PROMPT_SPEC.md`, `CHANGELOG.md`, `reference/`)<br>• **New** [`CLAUDE.md`](./CLAUDE.md) — Claude Code entry point. Eight MUST rules adapted to the provider's threat surface (untrusted GraphQL responses, no tokens in commits, no fabricated remote state, opt-in acceptance tests). Provider-specific conventions section codifies the Plugin-Framework / `logscale_<noun>` / `Sensitive`+`Computed` / `RequiresReplace` / state-preservation patterns already established in `internal/provider/`<br>• **New** [`CHANGE-HISTORY.md`](./CHANGE-HISTORY.md) — this file. Append-only provenance log<br>• **New** area `AGENTS.md` files under [`internal/provider/`](./internal/provider/AGENTS.md), [`reference/`](./reference/AGENTS.md), [`terraform-test-project/`](./terraform-test-project/AGENTS.md), and [`.github/`](./.github/AGENTS.md)<br>• **Updated** [`CHANGELOG.md`](./CHANGELOG.md) `## [Unreleased]` section with the routing additions, and added a follow-up note for the `logscale_view_role_assignment` tests gap |
