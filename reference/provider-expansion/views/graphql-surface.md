# View GraphQL Surface

## Summary

Views look viable for Terraform management, but they need slightly more careful modeling than parsers because their delete path lives in the shared search-domain API rather than a dedicated `deleteView()` mutation.

## Confirmed Capabilities

### Mutations

- `createView`
- `updateView`
- documented delete path via `deleteSearchDomain()` or `deleteSearchDomainById()`

### Query/Read Paths

- `querySearchDomains(typeFilter: Views, ...)`
- `searchDomain(name: ...)`
- `searchDomains()`

These query paths suggest the provider should treat views as a specialized case of LogScale search domains when reading/importing.

## Key Input/Return Shape

### View Inputs

Confirmed from historical probes and docs:

- `name`
- `description`
- `connections`
- `connections[].repositoryName`
- `connections[].filter`
- `connections[].languageVersion`

### View Connection Return Shape

The `ViewConnection` datatype exposes:

- `repository`
- `filter`
- `languageVersion`

This should be sufficient for a nested Terraform repository-connection block if read order is handled carefully.

## Historical Probe Notes

Earlier local GraphQL probes validated the basic request structure for:

- create (`createView` / equivalent view-create mutation)
- update (view connection update)
- delete (older delete note; no longer authoritative alone)

The existing delete note should no longer be treated as authoritative by itself, because the official docs now explicitly point to `deleteSearchDomain()` and `deleteSearchDomainById()` for deleting views.

## Terraform Modeling Notes

Recommended starting schema:

- `id` computed
- `name` required
- `description` optional
- `repository_connection` repeated nested block

Recommended nested connection fields:

- `repository_name` required
- `filter` required
- `language_version` optional

## Current Implementation Status

As of April 6, 2026:

- `logscale_view` has been added to the provider
- v1 supports create/read/update/delete
- v1 supports import by view name
- v1 models repository connections as nested Terraform blocks
- v1 currently updates connections in place
- v1 currently treats `name` and `description` as replacement fields
- `data.logscale_view` now exists for lookup by view name

As of April 9, 2026:

- the Terraform schema was corrected to use `repository_connection` instead of `connection`
- this was necessary because `connection` is a reserved Terraform root attribute/block name and caused provider schema loading failures

As of April 11, 2026:

- the provider now preserves a null Terraform `description` when the API read path returns an empty string for an omitted description
- this avoids inconsistent-result-after-apply failures for views created without an explicit description
- the provider also now preserves an omitted `repository_connection` block as null when the API read path returns no connections

Deferred for follow-up:

- `language_version`
- rename semantics through search-domain rename operations
- clearer description update semantics if the API supports them outside create
- acceptance validation against the target LogScale cloud tenant

## Import/Read Considerations

Likely import/read keys to investigate:

- by view name through `searchDomain(name: ...)`
- by search-domain ID through `querySearchDomains(...)` plus filtering if direct ID lookup proves awkward

Important modeling note:

- view names appear central to the public API surface, so name mutability and rename behavior need to be understood before finalizing import format

## Open Questions

- should view deletion use name-based or ID-based search-domain deletion in the provider?
- is rename exposed separately enough that `name` should be mutable, or should it require replacement in v1?
- can `querySearchDomains` and `searchDomain` return connection data in a stable order?

## Sources

- `createView`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-createview.html
- `updateView`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updateview.html
- `searchDomain`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-searchdomain.html
- `searchDomains`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-searchdomains.html
- `querySearchDomains`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-querysearchdomains.html
- `ViewConnectionInput`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-viewconnectioninput.html
- `ViewConnection`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-viewconnection.html
