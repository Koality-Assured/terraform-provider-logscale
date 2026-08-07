# External References And Non-Code Dependencies

## Summary

Not every remote object that the provider interacts with will be created by Terraform in this repository. Some resources will need to reference users, actions, repositories, views, package assets, or other identities that already exist in LogScale or are managed elsewhere.

The spec and future implementation should model these dependencies explicitly rather than pretending every referenced object is provider-managed.

## Why This Matters

Examples:

- a group membership resource references users even though the provider does not create users
- an alert may reference actions that could be managed externally
- an ingest token references a repository and may reference a parser by name or ID
- an AWS ingest feed references a parser plus AWS-side infrastructure that Terraform may not manage from this provider
- a view connection references repositories that may already exist
- parser assignment may refer to package-scoped parser names

If these relationships are not modeled carefully, the provider can become hard to compose and hard to understand.

## Reference Categories

### Existing identities

- users
- groups
- roles
- actions

### Existing assets

- repositories
- views
- parsers
- saved queries
- lookup files
- ingest tokens
- alerts
- scheduled searches

### External or pre-existing namespaces

- package-scoped parser names
- SaaS-managed built-in assets
- objects provisioned outside Terraform
- cloud infrastructure objects such as AWS IAM roles and SQS queues

## Implementation Guidance

- document clearly when a resource references an external object without managing its lifecycle
- prefer schema fields that make reference type obvious, such as `_id`, `_name`, or `_ids_or_names`
- preserve stable identifiers in state when the API returns them
- use data sources where practical to resolve externally managed objects
- do not imply ownership of assets the provider cannot create, update, or delete

## Immediate Relevance To Current Work

For parsers and views:

- views reference repositories through connections
- parsers are currently modeled against a repository namespace in provider v1
- ingest tokens may later reference parsers created by the new parser resource

For ingest feeds:

- AWS S3/SQS ingest feeds reference parsers by name or ID rather than owning parser lifecycle
- AWS S3/SQS ingest feeds also depend on external AWS infrastructure, especially IAM roles and SQS queue URLs
- the provider should model those fields clearly as references/configuration, not as owned AWS resources

For current existing resources:

- group membership references users that are not provider-managed
- role assignments reference groups and roles that may have mixed ownership
- alerts and scheduled searches can now reference actions, saved queries, and lookup files that may be managed in the same Terraform stack or resolved from existing LogScale assets

As of April 6, 2026:

- user lookup/list data sources now exist to help bridge that gap for existing identities
- user lifecycle itself remains outside provider ownership
- group membership now reads remote usernames from the group query and warns when requested usernames do not reconcile to what LogScale returns

These patterns should be called out directly in the prompt spec and resource docs as implementation begins.
