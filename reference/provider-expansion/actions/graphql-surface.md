# Action GraphQL Surface

## Summary

Actions are a strong foundation for detection-style Terraform management because alert and scheduled-search resources frequently reference them.

Webhook actions are the best first implementation target because their input surface is compact, the GraphQL mutations are explicit, and the returned datatype is rich enough for stable state.

## Confirmed Capabilities

### Mutations

- `createWebhookAction`
- `updateWebhookAction`
- `deleteActionV2`
- `addActionLabels`
- `removeActionLabels`

### Read Paths

- repository and view search-domain reads expose `actions`
- `WebhookAction` implements the broader `Action` interface

## Key Input/Return Shape

Confirmed webhook action input fields include:

- `viewName`
- `name`
- `url`
- `method`
- `headers`
- `bodyTemplate`
- `ignoreSSL`
- `useProxy`

Returned webhook action fields worth tracking include:

- `id`
- `name`
- `displayName`
- `labels`
- `headers`
- `url`
- `method`
- `ignoreSSL`
- `useProxy`
- `bodyTemplate`
- `resource`
- `yamlTemplate`
- `requiresOrganizationOwnedQueriesPermissionToEdit`

## Current Implementation Status

As of April 20, 2026:

- `logscale_webhook_action` has been added to the provider
- CRUD is implemented
- import uses `view_name:action_id`
- labels are managed through the dedicated action label mutations
- `data.logscale_webhook_action` has been added for lookup by `id` or `name` within a view/repository
- `data.logscale_webhook_actions` has been added for listing webhook actions within a view/repository

## Modeling Notes

- labels are not part of the create/update webhook action inputs, so the provider needs post-create/post-update label reconciliation
- action ownership is explicitly reference-based for downstream alert/trigger resources
- read paths should filter by `__typename` so webhook-action logic does not accidentally bind to another action type

## Open Questions

- which additional action types matter enough to model directly after webhook actions?
- should action list data sources grow search/filter semantics beyond full-view listings?
- should action data sources eventually expose `display_name` and other metadata once tenant validation is complete?

## Sources

- `createWebhookAction()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-createwebhookaction.html
- `updateWebhookAction()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updatewebhookaction.html
- `deleteActionV2()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deleteactionv2.html
- `CreateWebhookAction`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-createwebhookaction.html
- `UpdateWebhookAction`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-updatewebhookaction.html
- `DeleteActionV2`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-deleteactionv2.html
- `WebhookAction`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-webhookaction.html
