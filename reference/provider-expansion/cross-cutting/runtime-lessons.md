# Runtime Lessons And Bug-Fix Patterns

## Summary

This document captures implementation lessons that came from real provider failures, tenant-specific behavior, and Terraform state-model edge cases.

These are not just historical notes. They are design inputs for future resources and should be reviewed when adding new schema fields, read paths, or GraphQL mutations.

## Current Lessons

### Prefer cloud-validated read paths over narrower documented fields

Official GraphQL documentation is useful, but LogScale cloud behavior should win when the two differ.

Confirmed example:

- parser reads originally used a narrower `repository.parser(parserid: ...)` style lookup
- at least one cloud tenant rejected that shape at runtime
- the provider now reads parsers through `repository { parsers { ... } }` and matches client-side by parser ID, then name

Implication:

- when choosing between a direct lookup and a broader list/read path, favor the shape that is proven to work in the target cloud tenant if drift and performance remain acceptable

## Terraform State-Normalization Lessons

### Preserve null when the API normalizes omitted values to empty values

Several provider bugs came from Terraform plan/state saying "unset" while the API read path returned an empty string, empty list, or empty set.

Confirmed examples:

- repository `description`
- view `description`
- parser `fields_to_tag`
- parser `fields_to_be_removed_before_parsing`
- parser `test_case` collections
- view `repository_connection` collections

Implication:

- future optional strings, lists, sets, and nested collections should be reviewed for null-vs-empty behavior before release
- shared normalization helpers are preferred over one-off fixes where practical

### Optional nested blocks need the same care as optional attributes

The same drift pattern does not stop at strings or sets.

Confirmed examples:

- omitted `repository_connection` can come back as an empty API array
- omitted parser test-case/assertion structures can come back as empty API arrays

Implication:

- whenever a resource adds optional nested blocks, check whether an omitted block should remain null when the API returns zero elements

### Template-bearing assets need newline/canonicalization awareness

Template-driven resources can drift even when the logical content is unchanged if the API normalizes line endings or formatting.

Confirmed example:

- dashboard YAML templates may be authored with Windows line endings locally while the API readback may normalize them to Unix line endings

Implication:

- when a resource uses YAML/template strings as a primary content field, preserve the current value when the remote content is logically equivalent but only newline-normalized

## Schema Design Lessons

### Avoid reserved Terraform root names

Terraform framework/schema rules can reject root attribute or block names even before resource logic runs.

Confirmed example:

- `connection` was rejected as a reserved root name for views
- the provider now uses `repository_connection`

Implication:

- schema review should include Terraform-reserved-name checks before shipping a new resource

## GraphQL Mutation Lessons

### Use the documented input type names exactly

Small GraphQL type-name mistakes can survive code review and only fail at runtime.

Confirmed example:

- parser create originally used `CreateParserV2Input`
- the correct type name was `CreateParserInputV2`

Implication:

- mutation definitions should be checked against official datatype docs during implementation
- this is a good area for future regression tests

### Prefer mutation examples and successful return shapes when docs disagree internally

Not every GraphQL datatype page and mutation page is perfectly aligned.

Confirmed examples:

- scheduled-search update docs currently show `querystring` on one datatype page while mutation examples and return fields use `queryString`
- aggregate-alert docs still show a legacy single `throttleField` on create while update v2 documents `throttleFields`
- scheduled-search docs also document `IngestTimestamp` support behind the `ScheduledSearchIngestTimestamp` feature flag, even while the core mutation surface is long-term and otherwise stable
- saved-query CRUD now has both classic field-based mutations and a preview YAML-template lifecycle, and the template path maps better to Terraform-managed reusable query assets

Implication:

- when docs disagree, prefer the shape that is consistent with the mutation examples and the returned object fields
- record the mismatch in the resource-family research notes so we do not re-learn the same quirk later
- add local validation or conservative input shaping when a create and update path are not symmetrical
- treat feature-flag notes in the docs as real tenant-validation requirements, even when the surrounding mutation family looks stable enough for implementation
- when a template lifecycle and a flatter legacy lifecycle coexist, prefer the lifecycle that matches how the asset is actually versioned and composed

## Operational Guidance Lessons

### Experimental resources should warn, not bluff

When a resource is usable but still tenant-sensitive, the provider and docs should say so directly.

Confirmed example:

- group membership now performs remote readback and warns when requested usernames do not reconcile cleanly
- this is stronger and more honest than the earlier state-only behavior
- lookup-file v1 uses a whole-file CSV snapshot model over the lower-level file edit surface, so tenant validation should confirm that the update semantics match that assumption in practice

Implication:

- for partially reliable resource families, prefer remote readback plus warnings over optimistic local-state assumptions

## What To Check On New Resources

Before considering a new resource family stable, verify:

- the chosen read path is proven against the target cloud tenant
- optional attributes do not drift from null to empty values after apply
- optional nested blocks do not drift from null to empty collections after apply
- GraphQL mutation input type names exactly match the docs
- schema field names do not use Terraform-reserved root names
- docs and wiki notes reflect any tenant-specific caveats discovered during implementation
