# AGENTS — `reference/`

This directory is the in-repo research wiki. It is documentation-only and stays outside the provider build path.

## What lives here

- `reference/README.md` — central index
- `reference/provider-expansion/README.md` — expansion-program index
- `reference/provider-expansion/<family>/README.md` — per-family overview (parsers, views, dashboards, triggers, actions, ingest-feeds, files, users, query-validation, saved-queries)
- `reference/provider-expansion/<family>/graphql-surface.md` — per-family GraphQL surface notes
- `reference/provider-expansion/cross-cutting/` — methodology review, runtime lessons, external references

## What to capture

For each family or topic, document:

- **Confirmed behavior** — verified against a real tenant you own (manual GraphQL) or against current LogScale GraphQL docs
- **Probable behavior** — strong inference from docs, not yet validated against a tenant
- **Open questions** — gaps the provider may need to close later
- **Runtime lessons** — bug fixes, tenant-specific behavior, Terraform state-model failures (in `cross-cutting/runtime-lessons.md`)
- **External dependencies** — assets a resource references but does not own (e.g. user references in group membership, action references in alerts) — see `cross-cutting/external-references.md`
- **Methodology decisions** — when scope is narrowed for practicality, record the selected first-implementation slice so deferred families remain explicitly intentional (in `cross-cutting/methodology-review.md`)

## What not to capture

- No secrets, ingest tokens, API tokens, customer data, or tenant identifiers
- No copy-paste of upstream docs that may have a different license — link to the LogScale library URL and summarize in your own words

## When to update this area

- **Before** or **alongside** implementing a new resource or data source — the wiki page should describe the GraphQL surface, schema candidates, and lifecycle constraints before the Go code lands
- After a tenant bug fix or unexpected runtime behavior — append a note to `cross-cutting/runtime-lessons.md` with date, observed behavior, root cause, and resolution
- When scope changes — update the relevant family `README.md` and the cross-cutting methodology page
- When the `## [Unreleased]` CHANGELOG entry references a methodology shift — make sure the wiki page captures the same shift in its own words

## Routing

- New resource? Read [`provider-expansion/<family>/README.md`](./provider-expansion/) first; if no folder exists for the family, create one with `README.md` + `graphql-surface.md`.
- Tenant validation note? Add to the family page under a `Tenant validation notes` heading, or to `cross-cutting/runtime-lessons.md` if the lesson is cross-cutting.
- Decision worth surfacing project-wide? Add to [`../PROMPT_SPEC.md`](../PROMPT_SPEC.md)'s `Decision Log`.
