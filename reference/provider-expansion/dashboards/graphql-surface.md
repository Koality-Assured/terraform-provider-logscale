# Dashboard GraphQL Surface

## Summary

Dashboards are in scope, and the provider now supports a practical v1 CRUD path based on the official YAML-template mutations.

The native dashboard input graph is still significantly larger and more nested than actions or filter alerts, so the provider currently favors the template lifecycle over trying to model the full widget/section schema in Terraform.

## Confirmed Capabilities

### Queries

- `dashboardsPage`

### Mutations

- `createDashboard`
- `createDashboardFromTemplateV2`
- `updateDashboard`
- `updateDashboardFromTemplate`
- `deleteDashboardV3`
- legacy `deleteDashboard` is deprecated
- `addDashboardLabels`
- `removeDashboardLabels`

## Key Input/Return Shape

Confirmed dashboard query fields include:

- `id`
- `name`
- `displayName`
- `searchDomain`

The full native dashboard mutation surface is much richer and includes nested sections such as:

- parameters
- widgets
- update frequency
- view/search-domain association

This is the main reason the provider does not try to model the full nested dashboard graph in v1.

The template-oriented input surface is much smaller and better suited for Terraform v1:

- `CreateDashboardFromTemplateV2Input`
- `UpdateDashboardFromTemplateInput`

Those inputs focus on:

- `viewName`
- `name` for create
- `id` for update
- `yamlTemplate`

## Current Implementation Status

As of April 16, 2026:

- `data.logscale_dashboard` has been added for single-dashboard lookup
- `data.logscale_dashboards` has been added for paginated dashboard listing through `dashboardsPage`
- `logscale_dashboard` has been added with CRUD and import support
- dashboard lookup now exposes richer fields needed for composition and template-driven management, including labels, resource identifiers, and `yaml_template`

## Modeling Notes

- the provider uses the YAML-template lifecycle for dashboard CRUD:
  - create: `createDashboardFromTemplateV2`
  - update: `updateDashboardFromTemplate`
  - delete: `deleteDashboardV3`
- `dashboardsPage` is still a good fit for both data sources and read-path lookups because it provides pagination and a broad dashboard return type
- `deleteDashboardV3` should be preferred over the deprecated `deleteDashboard`
- dashboard labels are managed through the dedicated `addDashboardLabels` / `removeDashboardLabels` mutations
- `name` uses replacement semantics in the Terraform schema for now, even though the template may contain a name, because the create input and update input are asymmetric and we want Terraform behavior to stay explicit
- `yaml_template` is treated as the source of truth for content updates, and the provider preserves equivalent current YAML when the API only normalizes line endings

## Sources

- `dashboardsPage()`: https://library.humio.com/logscale-graphql-reference-queries/graphql-query-field-dashboardspage.html
- `createDashboardFromTemplateV2()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-createdashboardfromtemplatev2.html
- `updateDashboardFromTemplate()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updatedashboardfromtemplate.html
- `updateDashboard()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updatedashboard.html
- `deleteDashboardV3()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deletedashboardv3.html
- `deleteDashboard()`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deletedashboard.html
- `CreateDashboardFromTemplateV2Input`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-datatype-createdashboardfromtemplatev2input.html
- `UpdateDashboardFromTemplateInput`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-updatedashboardfromtemplateinput.html
- `AddDashboardLabels`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-adddashboardlabels.html
- `RemoveDashboardLabels`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-removedashboardlabels.html
- `Dashboard`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-dashboard.html
- `DashboardPage`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-type-dashboardpage.html
