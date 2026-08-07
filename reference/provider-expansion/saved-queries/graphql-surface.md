# Saved Query GraphQL Surface

## Summary

Saved queries are a strong fit for detection-as-code because they let teams version reusable query assets independently from alerts and scheduled searches.

For provider purposes, the most honest v1 shape is the YAML-template lifecycle rather than trying to flatten every query-option field into Terraform attributes up front.

## Confirmed Capabilities

- `createSavedQueryFromTemplate`
- `updateSavedQueryFromTemplate`
- `deleteSavedQueryV2`
- `addSavedQueryLabels`
- `removeSavedQueryLabels`
- `savedQuery(id: ...)`
- `repository { savedQueries }`
- `view { savedQueries }`

## Key Input And Return Shape

Template lifecycle inputs include:

- `viewName`
- optional `name`
- `yamlTemplate`

Saved-query readback fields worth tracking include:

- `id`
- `name`
- `displayName`
- `description`
- `labels`
- `query.queryString`
- `query.isLive`
- `query.start`
- `query.end`
- `isStarred`
- `resource`
- `yamlTemplate`

## Current Implementation Status

As of April 20, 2026:

- `logscale_saved_query` has been added to the provider
- CRUD is implemented through `createSavedQueryFromTemplate` / `updateSavedQueryFromTemplate` / `deleteSavedQueryV2`
- import uses `view_name:saved_query_id`
- saved-query labels are managed through dedicated label mutations
- `data.logscale_saved_query` has been added for lookup by `id` or `name` within a view/repository
- `data.logscale_saved_queries` has been added for listing saved queries within a view/repository

## Modeling Notes

- saved queries now follow the same template-first methodology as dashboards
- `name` is optional in the provider resource because LogScale can derive it from the YAML template
- the provider treats query details such as `query_string`, `is_live`, `query_start`, and `query_end` as readback/composition fields in v1 rather than duplicating the full template schema
- label limits are documented as at most ten labels with a maximum of sixty characters per label

## Follow-up Candidates

- tenant validation of template-based saved-query CRUD in LogScale cloud
- decide whether a future v2 should expose selected non-template attributes directly in addition to `yaml_template`
- detection-pack examples that combine saved queries with scheduled searches and alerts

## Sources

- `createSavedQueryFromTemplate()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-createsavedqueryfromtemplate.html
- `updateSavedQueryFromTemplate()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updatesavedqueryfromtemplate.html
- `deleteSavedQueryV2()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deletesavedqueryv2.html
- `addSavedQueryLabels()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-addsavedquerylabels.html
- `removeSavedQueryLabels()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-removesavedquerylabels.html
- `savedQuery()`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-savedquery.html
- `SavedQuery` datatype: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-savedquery.html
