# Trigger GraphQL Surface

## Summary

For provider purposes, "triggers" are best modeled as explicit resource families such as filter alerts, aggregate alerts, and scheduled searches rather than a single generic trigger abstraction.

Filter alerts were the first implementation target because the CRUD surface is direct, the input shape is manageable, and they connect naturally to action resources.

## Confirmed Capabilities

### Permissions And Operational Constraints

- aggregate-alert create uses the `CreateTriggers` API permission
- aggregate-alert update/delete flows are documented under the broader trigger-update permission model
- scheduled-search create uses the `CreateTriggers` API permission
- scheduled-search update/delete use the `UpdateTriggers` API permission
- aggregate-alert labels are documented as a maximum of ten labels with up to sixty characters per label
- scheduled-search labels are documented as a maximum of ten labels with up to sixty characters per label
- scheduled-search `timeZone` currently only supports UTC offsets such as `UTC`, `UTC-01`, or `UTC+12:45`
- scheduled-search `IngestTimestamp` behavior is documented as feature-flag-gated through `ScheduledSearchIngestTimestamp`

### Filter Alerts

- `createFilterAlert`
- `updateFilterAlertV2`
- `deleteFilterAlertV2`
- `addFilterAlertLabels`
- `removeFilterAlertLabels`

### Related Trigger Families

- `createAggregateAlert`
- `updateAggregateAlertV2`
- `deleteAggregateAlertV2`
- `addAggregateAlertLabels`
- `removeAggregateAlertLabels`
- `createScheduledSearchV2`
- `updateScheduledSearchV3`
- `deleteScheduledSearchV2`
- `addScheduledSearchLabels`
- `removeScheduledSearchLabels`

## Key Input/Return Shape

Confirmed filter alert inputs include:

- `viewName`
- `name`
- `description`
- `enabled`
- `queryString`
- `actionIdsOrNames`
- `queryOwnershipType`
- `runAsUserId`
- `throttleTimeSeconds`
- `throttleFields`
- `labels`

Returned filter alert fields worth tracking include:

- `id`
- `name`
- `description`
- `enabled`
- `queryString`
- `labels`
- `queryOwnership`
- `actions`
- `throttleTimeSeconds`
- `throttleFields`
- `resource`
- `yamlTemplate`

## Current Implementation Status

As of April 16, 2026:

- `logscale_filter_alert` has been added to the provider
- CRUD is implemented
- import uses `view_name:filter_alert_id`
- labels are handled directly through the filter alert create/update input surface
- `data.logscale_filter_alert` has been added for lookup by `id` or `name` within a view/repository
- `logscale_aggregate_alert` has been added to the provider
- CRUD is implemented
- import uses `view_name:aggregate_alert_id`
- `data.logscale_aggregate_alert` has been added for lookup by `id` or `name` within a view/repository
- `data.logscale_aggregate_alerts` has been added for listing aggregate alerts within a view/repository
- `logscale_scheduled_search` has been added to the provider
- CRUD is implemented
- import uses `view_name:scheduled_search_id`
- `data.logscale_scheduled_search` has been added for lookup by `id` or `name` within a view/repository
- `data.logscale_scheduled_searches` has been added for listing scheduled searches within a view/repository

## Modeling Notes

- action references are intentionally modeled as `action_ids_or_names` because the upstream API accepts both forms
- the provider attempts to preserve the user's existing reference style where it can reconcile remote actions back to current state
- query ownership is modeled explicitly because user-owned and organization-owned query execution matter operationally
- aggregate alerts and scheduled searches also preserve the user's action reference style where the remote response can be reconciled back to current state
- aggregate alert create/update documentation is not fully symmetrical: current docs still expose the legacy single `throttleField` on create while update v2 documents `throttleFields`; the provider validates this explicitly and uses the narrower create-safe path
- scheduled search docs contain a notable datatype/mutation mismatch where the datatype page shows `querystring` while the mutation examples show `queryString`; the provider uses `queryString` because that matches the mutation examples and the returned field name
- scheduled search local validation is intentionally opinionated because `EventTimestamp` and `IngestTimestamp` require different companion fields and this is cheaper to catch before apply than after a GraphQL failure
- scheduled-search `timeZone` support should be treated conservatively because the docs only promise UTC offsets, not arbitrary time zone database names
- scheduled-search `IngestTimestamp` support should still be treated as tenant-sensitive because the official docs explicitly tie it to the `ScheduledSearchIngestTimestamp` feature flag

## Follow-up Candidates

- additional action types paired with trigger resources
- tenant validation of aggregate-alert and scheduled-search behavior in LogScale cloud, especially around `IngestTimestamp` scheduled searches
- decide whether broader alert list filtering/search support is worth adding beyond simple full-view listings

## Sources

- `createFilterAlert()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-createfilteralert.html
- `updateFilterAlertV2()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updatefilteralertv2.html
- `deleteFilterAlertV2()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deletefilteralertv2.html
- `createAggregateAlert()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-createaggregatealert.html
- `updateAggregateAlertV2()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updateaggregatealertv2.html
- `deleteAggregateAlertV2()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deleteaggregatealertv2.html
- `addAggregateAlertLabels()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-addaggregatealertlabels.html
- `removeAggregateAlertLabels()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-removeaggregatealertlabels.html
- `createScheduledSearchV2()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-createscheduledsearchv2.html
- `updateScheduledSearchV3()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updatescheduledsearchv3.html
- `deleteScheduledSearchV2()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deletescheduledsearchv2.html
- `addScheduledSearchLabels()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-addscheduledsearchlabels.html
- `removeScheduledSearchLabels()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-removescheduledsearchlabels.html
