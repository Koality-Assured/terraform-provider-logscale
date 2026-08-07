# Users

This section captures user lookup and listing research for provider composition and external-reference workflows.

## Documents

- [GraphQL Surface](./graphql-surface.md)
  Summary: documented user lookup/list capabilities, permission implications, and current Terraform data source design.

## Current Takeaways

- users are an external identity surface that the provider currently references but does not create
- `user(id)` is the strongest exact-lookup query
- `usersPage(...)` is the most practical listing/search query for organization-scoped use
- `users()` exists, but it is cluster-wide and requires broader permissions than we want to assume for normal cloud workflows
- v1 user lookup/list is better represented as data sources than resources
