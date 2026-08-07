# Query Validation GraphQL Surface

## Summary

Query validation is one of the highest-value preflight capabilities for detection-as-code because it lets teams catch query compilation failures before a saved query, alert, or scheduled search reaches apply-time API errors.

## Confirmed Capabilities

- `validateQuery`

## Key Input And Return Shape

Validation inputs include:

- `queryString`
- `version`
- `isLive`
- `arguments[].name`
- `arguments[].value`

Validation results include:

- `isValid`
- `diagnostics[].severity`
- `diagnostics[].message`
- `diagnostics[].code`

## Current Implementation Status

As of April 20, 2026:

- `data.logscale_validate_query` has been added to the provider
- the data source exposes `query_string`, `version`, `is_live`, `argument`, `is_valid`, and returned diagnostics

## Modeling Notes

- the provider currently validates language version locally against the documented enum values `legacy`, `xdr1`, and `federated1`
- `validateQuery()` is intentionally modeled as a data source rather than being silently called from every CRUD resource
- this keeps validation explicit, side-effect-free, and reusable across saved queries, alerts, and scheduled searches

## Follow-up Candidates

- consider optional example modules that pair `data.logscale_validate_query` with saved-query and alert resources
- consider richer preflight helpers if `analyzeQuery()` becomes useful and stable enough for provider use later

## Sources

- `validateQuery()`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-validatequery.html
- `QueryArgument` datatype: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-queryargument.html
- `QueryValidationInfo` datatype: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-queryvalidationinfo.html
- `QueryDiagnostic` datatype: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-querydiagnostic.html
