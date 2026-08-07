# User GraphQL Surface

## Summary

Users are a reference surface, not a lifecycle-managed resource, for this provider today. The immediate need is lookup and listing so other resources such as group memberships can compose with existing LogScale identities more safely.

## Confirmed Capabilities

### Queries

- `user(id: ...)`
- `usersPage(search:, orderBy:, pageSize:, pageNumber:)`
- `users()`

## Permission And Scope Notes

- `user(id)` is a direct exact lookup path by unique identifier
- `usersPage(...)` is organization-scoped and is the preferred practical listing/search query for cloud usage
- `users()` is cluster-wide and requires `ManageCluster`, so it should not be the default assumption for normal provider workflows

## Returned Fields Worth Using

The user/user-page surfaces expose useful composition fields such as:

- `id`
- `username`
- `displayName`
- `email`
- `isRoot`
- `isOrgRoot`

These are sufficient for Terraform data-source support that helps users reference existing LogScale identities.

## Current Implementation Status

As of April 6, 2026:

- `data.logscale_user` has been added for exact lookup by `id`, `username`, `email`, or `display_name`
- `data.logscale_users` has been added for paginated user listing and search through `usersPage(...)`
- the provider still does not manage user lifecycle directly

## Terraform Modeling Notes

Current design choices:

- use data sources instead of resources
- prefer `user(id)` for exact ID lookups
- prefer `usersPage(...)` for list/search use cases and non-ID exact matching
- avoid default dependence on `users()` because of broader permission requirements

## Immediate Relevance To Existing Resources

- group memberships currently reference usernames even though users are not provider-managed
- user data sources provide a cleaner bridge between existing identities and Terraform-managed access resources
- future alert/action ownership or permission modeling may also need user references

As of April 6, 2026:

- `logscale_group_membership` now uses the group query's `users { username }` read path to reconcile remote state more honestly
- mutation success still depends on the username format accepted by the tenant, so manual verification warnings remain appropriate when remote membership does not match the requested usernames

## Sources

- `user()`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-user.html
- `usersPage()`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-userspage.html
- `users()`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-users.html
- `User`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-user.html
- `UsersPage`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-userspage.html
