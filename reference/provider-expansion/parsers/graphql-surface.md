# Parser GraphQL Surface

## Summary

Parsers are one of the strongest early candidates for provider expansion. The official GraphQL surface supports create, update, delete, and dedicated test execution, and the returned parser datatype is rich enough to support a Terraform resource with meaningful drift detection.

## Confirmed Capabilities

### Mutations

- `createParserV2`
- `updateParserV2`
- `deleteParserV2`
- `testParserV2`

### Query/Read Paths

- repository-level parser lookup appears available through `repositories()` and `repository()` result fields such as `parsers`
- parser details are represented by the `Parser` datatype

## Key Input/Return Shape

### Create/Update Inputs

Confirmed from the official docs:

- `repositoryName`
- `name`
- `fieldsToTag`
- `fieldsToBeRemovedBeforeParsing`
- `testCases`
- `allowOverwritingExistingParser` on create
- `script` as a nested update object for update
- language version support through `languageVersion` / `LanguageVersionInputType`

### Test Cases

Confirmed input building blocks include:

- `ParserTestCaseInput`
- `ParserTestEventInput`
- `ParserTestCaseAssertionsForOutputInput`
- `ParserTestCaseOutputAssertionsInput`
- `FieldHasValueInput`

This suggests Terraform can model parser test cases as explicit nested blocks instead of hiding them in a JSON string.

### Returned Parser Fields Worth Considering

The `Parser` datatype exposes fields including:

- `id`
- `name`
- `displayName`
- `script`
- `fieldsToTag`
- `fieldsToBeRemovedBeforeParsing`
- `languageVersion`
- `testCases`
- `yamlTemplate`
- `description`
- `isBuiltIn`
- `isOverridden`
- `originDisplayString`
- creation/modification metadata

Not all of these need to be in v1, but they show that parser modeling can grow well beyond the early probe examples.

## Historical Probe Notes

Earlier local GraphQL probes validated the basic request structure for parser create/update and confirmed that test cases can be submitted in a practical shape.

## Terraform Modeling Notes

Recommended starting schema:

- `id` computed
- `repository_name` required
- `name` required
- `script` required
- `fields_to_tag` optional list(string)
- `fields_to_be_removed_before_parsing` optional list(string)
- `allow_overwriting_existing_parser` optional bool
- `test_case` repeated nested block

Recommended early computed/read-only candidates:

- `display_name`
- `language_version`
- `is_built_in`
- `yaml_template`

## Current Implementation Status

As of April 6, 2026:

- `logscale_parser` has been added to the provider
- v1 supports create/read/update/delete
- v1 supports import with `repository_name:parser_id`
- v1 models parser test cases as nested Terraform blocks
- v1 currently exposes `display_name`, `is_built_in`, and `yaml_template` as computed values

As of April 11, 2026:

- the provider create mutation now uses the documented `CreateParserInputV2` input type name
- earlier provider builds incorrectly used `CreateParserV2Input`, which caused parser create operations to fail at runtime
- the provider now preserves null Terraform set values for omitted optional parser field lists when the API returns empty arrays
- this avoids inconsistent-result-after-apply failures for `fields_to_tag` and `fields_to_be_removed_before_parsing`
- the provider also now preserves omitted parser test-case collections as null when the API returns no test cases or nested assertions
- parser reads now use `repository { parsers { ... } }` with client-side matching by ID or name because at least one cloud tenant rejected the narrower `parser(parserid: ...)` field arguments despite the official docs still listing them

Deferred for follow-up:

- `language_version`
- create-only overwrite behavior such as `allowOverwritingExistingParser`
- acceptance validation against the target LogScale cloud tenant

## Validation/Testability

Parsers have unusually strong validation potential because `testParserV2` is a dedicated mutation for exercising parser behavior.

Possible provider patterns:

- validate parser test cases before create/update
- expose parser test execution through diagnostics during apply
- add a validating data source later if full CRUD integration becomes too noisy

## Open Questions

- can parser test cases be read back in a stable enough order to avoid perpetual diffs?
- should `script` be a simple string in v1, with language-version details modeled separately?
- should built-in/package parser fields be exposed now or deferred until package/template work begins?

## Sources

- `createParserV2`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-createparserv2.html
- `updateParserV2`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updateparserv2.html
- `deleteParserV2`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deleteparserv2.html
- `testParserV2`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-testparserv2.html
- `CreateParserInputV2`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-createparserinputv2.html
- `UpdateParserInputV2`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-updateparserinputv2.html
- `Parser`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-parser.html
- `ParserTestCaseInput`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-parsertestcaseinput.html
- `ParserTestEventInput`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-parsertesteventinput.html
- `FieldHasValueInput`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-fieldhasvalueinput.html
