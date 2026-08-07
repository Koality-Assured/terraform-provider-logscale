# Parsers

This section captures parser-specific research for planned `logscale_parser` support.

## Documents

- [GraphQL Surface](./graphql-surface.md)
  Summary: documented create/read/update/delete/test capabilities, schema candidates, and implementation notes for parser resources.

## Current Takeaways

- parser CRUD looks strong in GraphQL for cloud, including `createParserV2`, `updateParserV2`, and `deleteParserV2`
- parser testing is first-class through `testParserV2`
- the parser datatype appears rich enough to support a detailed Terraform schema
- parser test cases are likely worth modeling natively rather than collapsing into opaque JSON
- a v1 `logscale_parser` resource is now implemented in the provider
