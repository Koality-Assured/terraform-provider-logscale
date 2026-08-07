# File GraphQL Surface

## Summary

Lookup files are an important detection-as-code dependency because they let queries reference managed CSV data for allowlists, blocklists, enrichment, and asset matching.

For provider purposes, the most practical v1 shape is a CSV-snapshot resource that owns the whole file content, rather than trying to model incremental row edits directly.

## Confirmed Capabilities

- `newFile`
- `updateFile`
- `deleteFile`
- `addFileLabels`
- `removeFileLabels`
- `getFileContent`
- `repository { files }`
- `view { files }`

## Key Input And Return Shape

Mutation/query inputs worth tracking include:

- `name` / `viewName`
- `fileName`
- `changedRows`
- `headers`
- `columnChanges`
- `labels`
- `filterString`
- `offset`
- `limit`

Readback fields worth tracking include:

- `nameAndPath.name`
- `nameAndPath.path`
- `headers`
- `lines`
- `totalLinesCount`
- `labels`
- `contentHash`
- `createdAt`
- `createdBy`
- `modifiedAt`
- `modifiedBy`
- `fileSizeBytes`
- `resource`

## Current Implementation Status

As of April 20, 2026:

- `logscale_lookup_file` has been added to the provider
- CRUD is implemented through `newFile`, `updateFile`, and `deleteFile`
- import uses `view_name:file_name`
- file labels are managed through `addFileLabels` / `removeFileLabels`
- `data.logscale_lookup_file` has been added for singular lookup
- `data.logscale_lookup_files` has been added for view-scoped listing

## Modeling Notes

- the provider models a lookup file as whole-file `csv_content`, including the header row
- readback uses `getFileContent`, then re-renders the CSV string into Terraform state
- update behavior currently assumes `updateFile` can be used as a snapshot-style replacement operation when given the full desired headers and rows
- because the GraphQL docs emphasize incremental concepts such as `changedRows` and `columnChanges`, tenant validation remains important for confidence in the whole-file replacement assumption
- label limits are documented as at most ten labels with a maximum of sixty characters per label

## Follow-up Candidates

- tenant validation of whole-file replacement semantics in LogScale cloud
- decide whether v2 should expose structured headers/rows in addition to `csv_content`
- consider a validation/helper data source for file-field search workflows once core lookup-file behavior is proven

## Sources

- `newFile()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-newfile.html
- `updateFile()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updatefile.html
- `deleteFile()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deletefile.html
- `addFileLabels()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-addfilelabels.html
- `removeFileLabels()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-removefilelabels.html
- `getFileContent()`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-getfilecontent.html
- `File` datatype: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-file.html
- `ColumnChange` datatype: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-columnchange.html
