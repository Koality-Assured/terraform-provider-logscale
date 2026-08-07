# Views

This section captures view-specific research for planned `logscale_view` support.

## Documents

- [GraphQL Surface](./graphql-surface.md)
  Summary: documented create/read/update/delete behavior, schema candidates, and implementation notes for view resources.

## Current Takeaways

- view create/update are clearly documented
- delete does appear to exist in the broader search-domain API, even though older probe notes assumed otherwise
- views are likely better modeled through the search-domain query family than through one-off assumptions
- view connections have stable input and output shapes suitable for nested Terraform blocks
- a v1 `logscale_view` resource is now implemented in the provider
- the provider schema must use `repository_connection` rather than `connection`, because `connection` is a reserved Terraform root block name
