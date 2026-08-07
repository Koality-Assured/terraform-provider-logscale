# LogScale Terraform Provider

[![Version](./.github/badges/version.svg)](./VERSION)

A Terraform provider for managing Falcon LogScale resources through the GraphQL API.

This repository is for provider implementation, documentation, research, and releases. Normal consumers should use the published release artifacts from this repo. Manual local builds are for provider development and debugging.

## Overview

This repository is where we implement, document, test, and release the LogScale Terraform provider itself.

What this repo is for:

- provider implementation
- provider release automation
- provider documentation and research wiki
- local validation examples

What this repo is not for:

- the primary production Terraform code that manages LogScale SaaS for your environment

Release notes are tracked in `CHANGELOG.md`. Release process details are in `RELEASING.md`. Ongoing research and API notes live under `reference/`.

## Current Provider Surface

### Resources

| Resource | Status | Notes |
|----------|--------|-------|
| `logscale_repository` | Partial CRUD | Create/read supported. Update/delete are limited by API behavior. |
| `logscale_parser` | Working (v1) | CRUD supported. Includes optional parser test cases. |
| `logscale_view` | Working (v1) | CRUD supported. Uses `repository_connection` blocks. |
| `logscale_dashboard` | Working (v1) | CRUD supported through the official YAML-template dashboard lifecycle. |
| `logscale_saved_query` | Working (v1) | CRUD supported through the YAML-template saved-query lifecycle. |
| `logscale_lookup_file` | Working (v1) | CRUD supported through the file upload/edit surface with whole-file CSV modeling. |
| `logscale_aws_s3_sqs_ingest_feed` | Working (v1) | CRUD supported for AWS S3/SQS ingest feeds with IAM role authentication. |
| `logscale_webhook_action` | Working (v1) | CRUD supported for webhook actions. |
| `logscale_filter_alert` | Working (v1) | CRUD supported for filter alerts. |
| `logscale_aggregate_alert` | Working (v1) | CRUD supported for aggregate alerts. |
| `logscale_scheduled_search` | Working (v1) | CRUD supported for scheduled searches with timestamp-mode validation. |
| `logscale_group` | Working | CRUD supported. |
| `logscale_group_membership` | Experimental | Remote readback and warnings are implemented, but tenant validation is still important before broad production use. |
| `logscale_role` | Working (v1) | CRUD supported via `createRole` / `updateRole` / `removeRole`. Models view-level permissions as a string set; import by role ID. |
| `logscale_ingest_token` | Working with caveats | Create/delete supported. Read currently preserves Terraform state because the provider does not have remote token readback; updates require recreation. |
| `logscale_organization_role_assignment` | Working with caveats | Assign/remove supported. Read currently preserves Terraform state rather than verifying the remote assignment. |
| `logscale_system_role_assignment` | Working with caveats | Assign/remove supported, often needs elevated permissions, and read currently preserves Terraform state rather than verifying the remote assignment. |
| `logscale_view_role_assignment` | Experimental | Assigns a role to a group on a specific view or repository through `assignRoleToGroup` / `removeRoleFromGroup`. Read currently preserves Terraform state rather than verifying the remote assignment. Tenant validation recommended before broad production use. |
| `logscale_default_role_assignment` | Experimental | Configures a group's default role through `updateDefaultRole`. Read preserves state; destroy emits a warning and detaches state because the upstream API does not expose a clean unset path. |

### Data Sources

| Data Source | Status | Notes |
|-------------|--------|-------|
| `data.logscale_role` | Working | Look up existing roles by display name or ID. |
| `data.logscale_user` | Working | Look up a single user by ID, username, email, or display name. |
| `data.logscale_users` | Working | List/search users with pagination. |
| `data.logscale_repository` | Working | Look up a repository by name. |
| `data.logscale_view` | Working | Look up a view by name. |
| `data.logscale_saved_query` | Working | Look up a saved query by ID or name within a repository/view. |
| `data.logscale_saved_queries` | Working | List saved queries within a repository/view. |
| `data.logscale_lookup_file` | Working | Look up a lookup file by file name within a repository/view. |
| `data.logscale_lookup_files` | Working | List lookup files within a repository/view. |
| `data.logscale_validate_query` | Working | Validate query text through GraphQL before wiring it into saved queries or triggers. |
| `data.logscale_aws_s3_sqs_ingest_feed` | Working | Look up an AWS S3/SQS ingest feed by ID or name within a repository. |
| `data.logscale_webhook_action` | Working | Look up a webhook action by ID or name within a repository/view. |
| `data.logscale_webhook_actions` | Working | List webhook actions within a repository/view. |
| `data.logscale_filter_alert` | Working | Look up a filter alert by ID or name within a repository/view. |
| `data.logscale_filter_alerts` | Working | List filter alerts within a repository/view. |
| `data.logscale_aggregate_alert` | Working | Look up an aggregate alert by ID or name within a repository/view. |
| `data.logscale_aggregate_alerts` | Working | List aggregate alerts within a repository/view. |
| `data.logscale_scheduled_search` | Working | Look up a scheduled search by ID or name within a repository/view. |
| `data.logscale_scheduled_searches` | Working | List scheduled searches within a repository/view. |
| `data.logscale_dashboard` | Working | Look up a dashboard by ID or name. |
| `data.logscale_dashboards` | Working | List dashboards with paginated search. |
| `data.logscale_view_role_assignments` | Experimental | List view-scoped role assignments held by a group via `group(groupId).roles`. Tenant validation recommended before broad production use. |

## Installation

### Preferred Path

Use the provider binaries published through this repository's GitHub Releases (see `RELEASING.md`).

- maintainers cut releases manually when `VERSION` is bumped
- Linux and Windows artifacts are published through GitHub Releases
- release notes come from `CHANGELOG.md`

### Local Development Path

If you are working on the provider itself, use a Terraform dev override that points at your locally built binary.

Example `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "local/logscale" = "/path/to/your/provider/binaries"
  }
  direct {}
}
```

With dev overrides, skip `terraform init` and go directly to `terraform plan` or `terraform apply`.

## Provider Configuration

Examples in this repository use the internal/private provider address `local/logscale`.

```hcl
terraform {
  required_providers {
    logscale = {
      source = "local/logscale"
    }
  }
}

provider "logscale" {
  api_url   = var.logscale_api_url
  api_token = var.logscale_api_token
}

variable "logscale_api_url" {
  type = string
}

variable "logscale_api_token" {
  type      = string
  sensitive = true
}
```

Provider arguments:

| Argument | Required | Description |
|----------|----------|-------------|
| `api_url` | Yes | LogScale GraphQL endpoint, for example `https://tenant.logscale.us-1.crowdstrike.com/graphql` |
| `api_token` | Yes | API token used for GraphQL authentication |

## Usage Examples

### Repository

```hcl
resource "logscale_repository" "security_logs" {
  name           = "security-logs"
  description    = "Security event logs"
  retention_days = 365
}
```

### Parser

```hcl
resource "logscale_parser" "json_example" {
  repository_name = logscale_repository.security_logs.name
  name            = "json-example"
  script          = file("${path.module}/parsers/json-example.ls")

  fields_to_tag = [
    "host",
    "service",
  ]

  test_case {
    event_raw_string = "{\"host\":\"web-01\",\"service\":\"api\"}"

    output_assertion {
      output_event_index = 0

      field_has_value {
        field_name     = "host"
        expected_value = "web-01"
      }
    }
  }
}
```

### View

```hcl
resource "logscale_view" "security_ops" {
  name        = "security-ops"
  description = "Security-focused cross-repository view"

  repository_connection {
    repository_name = logscale_repository.security_logs.name
    filter          = "*"
  }
}
```

### Webhook Action

```hcl
resource "logscale_webhook_action" "pagerduty" {
  view_name     = logscale_repository.security_logs.name
  name          = "pagerduty-webhook"
  url           = "https://events.example.invalid"
  method        = "POST"
  ignore_ssl    = false
  use_proxy     = false
  body_template = "{\"message\":\"{name}\"}"

  labels = ["critical", "webhook"]

  header {
    header = "Content-Type"
    value  = "application/json"
  }
}
```

### Dashboard

```hcl
resource "logscale_dashboard" "security_overview" {
  view_name     = logscale_view.security_ops.name
  name          = "security-overview"
  yaml_template = file("${path.module}/dashboards/security-overview.yaml")
  labels        = ["security", "overview"]
}
```

### Saved Query

```hcl
resource "logscale_saved_query" "rare_admin_query" {
  view_name     = logscale_view.security_ops.name
  name          = "rare-admin-query"
  yaml_template = file("${path.module}/saved-queries/rare-admin-query.yaml")
  labels        = ["detections", "reusable"]
}
```

### Lookup File

```hcl
resource "logscale_lookup_file" "vip_hosts" {
  view_name   = logscale_view.security_ops.name
  file_name   = "vip-hosts.csv"
  csv_content = file("${path.module}/lookup-files/vip-hosts.csv")
  labels      = ["detections", "lookup"]
}
```

### Aggregate Alert

```hcl
resource "logscale_aggregate_alert" "rare_admin_activity" {
  view_name               = logscale_view.security_ops.name
  name                    = "rare-admin-activity"
  description             = "Detect uncommon administrative activity"
  enabled                 = true
  query_string            = "#event_simpleName=AdminAction"
  action_ids_or_names     = [logscale_webhook_action.pagerduty.name]
  query_ownership_type    = "Organization"
  query_timestamp_type    = "EventTimestamp"
  search_interval_seconds = 300
  throttle_time_seconds   = 900
  trigger_mode            = "ImmediateMode"
  labels                  = ["security", "critical"]
}
```

### Scheduled Search

```hcl
resource "logscale_scheduled_search" "daily_health_check" {
  view_name                      = logscale_view.security_ops.name
  name                           = "daily-health-check"
  description                    = "Run a daily platform health query"
  enabled                        = true
  query_string                   = "#type=health"
  action_ids_or_names            = [logscale_webhook_action.pagerduty.name]
  labels                         = ["ops", "daily"]
  query_ownership_type           = "Organization"
  schedule                       = "0 6 * * *"
  time_zone                      = "UTC"
  search_interval_seconds        = 3600
  query_timestamp_type           = "EventTimestamp"
  search_interval_offset_seconds = 0
  backfill_limit                 = 24
  trigger_on_empty_result        = false
}
```

### AWS S3/SQS Ingest Feed

```hcl
resource "logscale_aws_s3_sqs_ingest_feed" "cloudtrail" {
  repository_name     = logscale_repository.security_logs.name
  name                = "aws-cloudtrail"
  description         = "CloudTrail ingest feed"
  enabled             = true
  parser              = logscale_parser.json_example.name
  region              = "us-east-1"
  sqs_url             = "https://sqs.us-east-1.amazonaws.com/123456789012/logscale-cloudtrail"
  compression         = "Auto"
  authentication_kind = "IamRole"
  role_arn            = "arn:aws:iam::123456789012:role/logscale-cloudtrail-reader"
  preprocessing_kind  = "SplitAwsRecords"
}
```

### Filter Alert

```hcl
resource "logscale_filter_alert" "auth_failures" {
  view_name            = logscale_repository.security_logs.name
  name                 = "auth-failures"
  enabled              = true
  query_string         = "#event_simpleName=AuthenticationFailed"
  query_ownership_type = "Organization"
  action_ids_or_names  = ["pagerduty-webhook"]
  labels               = ["authentication", "critical"]
}
```

### Role

```hcl
resource "logscale_role" "analyst" {
  display_name = "Analyst"
  view_permissions = [
    "ReadAccess",
    "ChangeDashboards",
  ]
}
```

### Default Role Assignment

```hcl
resource "logscale_default_role_assignment" "secops_default" {
  group_id = logscale_group.secops.id
  role_id  = logscale_role.analyst.id
}
```

### View Role Assignment

Grants a group a role on a specific view or repository. In LogScale a repository
is a kind of view, so `view_id` accepts either a view ID or a repository ID.

```hcl
resource "logscale_group" "secops" {
  display_name = "SecOps"
}

data "logscale_role" "analyst" {
  display_name = "Analyst"
}

resource "logscale_view_role_assignment" "secops_security_logs_analyst" {
  group_id = logscale_group.secops.id
  view_id  = logscale_repository.security_logs.id
  role_id  = data.logscale_role.analyst.id
}
```

### View Role Assignments Data Source

Lists the view-scoped role assignments a group currently holds, in the same
shape used by `logscale_view_role_assignment` (`group_id:view_id:role_id`).

```hcl
data "logscale_view_role_assignments" "secops" {
  group_id = logscale_group.secops.id
}

output "secops_view_assignments" {
  value = data.logscale_view_role_assignments.secops.assignments
}
```

### Users Data Source

```hcl
data "logscale_users" "all" {
  page_number = 1
  page_size   = 50
}

output "usernames" {
  value = [for u in data.logscale_users.all.users : u.username]
}
```

### Repository And View Data Sources

```hcl
data "logscale_repository" "security_logs" {
  name = "security-logs"
}

data "logscale_view" "security_ops" {
  name = "security-ops"
}
```

### AWS S3/SQS Ingest Feed Data Source

```hcl
data "logscale_aws_s3_sqs_ingest_feed" "cloudtrail" {
  repository_name = logscale_repository.security_logs.name
  name            = "aws-cloudtrail"
}
```

### Dashboard Data Sources

```hcl
data "logscale_dashboards" "all" {
  page_number = 1
  page_size   = 25
}

data "logscale_dashboard" "hosts" {
  name = "Hosts"
}
```

### Saved Query And Lookup File Data Sources

```hcl
data "logscale_saved_query" "rare_admin_query" {
  view_name = logscale_view.security_ops.name
  name      = "rare-admin-query"
}

data "logscale_saved_queries" "all_queries" {
  view_name = logscale_view.security_ops.name
}

data "logscale_lookup_file" "vip_hosts" {
  view_name = logscale_view.security_ops.name
  file_name = "vip-hosts.csv"
}

data "logscale_lookup_files" "all_lookup_files" {
  view_name = logscale_view.security_ops.name
}
```

### Trigger Data Sources

```hcl
data "logscale_webhook_actions" "all_actions" {
  view_name = logscale_view.security_ops.name
}

data "logscale_filter_alerts" "all_filter_alerts" {
  view_name = logscale_view.security_ops.name
}

data "logscale_aggregate_alert" "rare_admin_activity" {
  view_name = logscale_view.security_ops.name
  name      = "rare-admin-activity"
}

data "logscale_aggregate_alerts" "all_aggregate_alerts" {
  view_name = logscale_view.security_ops.name
}

data "logscale_scheduled_search" "daily_health_check" {
  view_name = logscale_view.security_ops.name
  name      = "daily-health-check"
}

data "logscale_scheduled_searches" "all_scheduled_searches" {
  view_name = logscale_view.security_ops.name
}
```

### Query Validation Data Source

```hcl
data "logscale_validate_query" "preflight" {
  query_string = "#event_simpleName=AdminAction"
  version      = "legacy"
  is_live      = false
}
```

## Important Behavior Notes

### Repository

- `description` is optional in the provider schema
- repository update/delete behavior is constrained by the upstream API
- plan repository naming and lifecycle carefully

### Parser

- import format is `repository_name:parser_id`
- optional `fields_to_tag`, `fields_to_be_removed_before_parsing`, and `test_case` blocks preserve omitted/null behavior where possible
- parser scripts can be supplied inline or through Terraform functions such as `file()`, `templatefile()`, or `yamldecode(file(...)).script`

### View

- use `repository_connection`, not `connection`
- `name` and `description` currently use replace semantics in the Terraform schema
- connections update in place

### Saved Query

- import format is `view_name:saved_query_id`
- saved-query CRUD uses the YAML-template lifecycle rather than flattening the full query asset schema into many Terraform attributes
- labels are managed through dedicated saved-query label mutations

### Lookup File

- import format is `view_name:file_name`
- `csv_content` must include the header row
- file CRUD currently assumes the upstream file-edit surface can be used as a whole-file replacement path, so tenant validation is still recommended before broad production use

### AWS S3/SQS Ingest Feed

- import format is `repository_name:ingest_feed_id`
- current v1 models the documented IAM role authentication path via `authentication_kind = "IamRole"`
- current v1 models the documented preprocessing kinds `SplitNewline` and `SplitAwsRecords`
- the official GraphQL API also exposes `testAwsS3SqsIngestFeed`; we have documented that research, but test execution is not wired into normal CRUD applies yet
- treat parser references as external dependencies: the provider stores the parser reference you supplied when it can still be reconciled from the API readback

### Group Membership

- this resource is still best treated as experimental
- the provider now reads back remote membership and warns on mismatch
- some tenants may still require manual verification of username format and post-apply results

### Ingest Token

- create/delete are implemented
- the provider does not currently read back ingest tokens from LogScale, so `Read` preserves Terraform state instead of reconciling against a remote token record
- the sensitive `token` value is only returned at creation time and is then retained in state
- updates are not supported; changes that affect the token should be handled by recreation

### Organization, System, And View Role Assignments

- all three role-assignment resources create and delete assignments, but current `Read` behavior preserves Terraform state instead of querying LogScale to verify the remote assignment still exists
- changes are recreate-only rather than in-place updates
- `logscale_system_role_assignment` often needs elevated permissions beyond what is required for the organization-role path
- `logscale_view_role_assignment` is the view-scoped equivalent: it grants a group a role on a specific view or repository through the `assignRoleToGroup` / `removeRoleFromGroup` GraphQL mutations; in LogScale a repository is a kind of view, so `view_id` accepts either a view ID or a repository ID
- the `view_id`, `group_id`, and `role_id` attributes on `logscale_view_role_assignment` are all `RequiresReplace`; changing any of them recreates the assignment
- for validation-sensitive workflows, confirm assignments in LogScale UI or with tenant-specific operational checks after apply
- `data.logscale_view_role_assignments` lists a group's current view-scoped assignments via `group(groupId).roles`; entries without a search domain (organization-/system-scoped grants) are filtered out

### Role

- create / update / delete via `createRole` / `updateRole` / `removeRole`
- `view_permissions` is a set of permission identifiers — the exact set accepted by your tenant is defined by the LogScale permissions enumeration
- import by role ID
- `display_name` and `view_permissions` are updated in place

### Default Role Assignment

- create / update via `updateDefaultRole`; `group_id` is `RequiresReplace`, `role_id` can be updated in place
- read preserves Terraform state (matches the org/sys/view role-assignment pattern)
- destroy emits a warning and detaches state only — LogScale does not expose a clean "unset default role" mutation, so if you intend to remove the default role entirely do so manually in the LogScale UI

### Dashboard Support

- dashboard lookup/list data sources are implemented
- dashboard CRUD is implemented through the official YAML-template lifecycle
- `name` currently uses replacement semantics in Terraform to keep rename behavior explicit
- `yaml_template` is the content source of truth for dashboard updates
- richer fully nested widget/section schema modeling is still deferred in favor of the safer template-based CRUD path

## Troubleshooting

### Wrong Provider Address

If Terraform is trying to install `hashicorp/logscale` or another public registry address, update your configuration to use the internal/private address you actually distribute. This repository's examples use:

```hcl
source = "local/logscale"
```

### Provider Schema Error About `connection`

Older configs may still use:

```hcl
connection {
  repository_name = "repo"
  filter          = "*"
}
```

Current provider versions require:

```hcl
repository_connection {
  repository_name = "repo"
  filter          = "*"
}
```

### Permission Errors

Some LogScale operations depend on the token type and granted permissions. Ingest token and system-role operations are the most common places to hit permission-related failures.

## Repository Layout

Key paths in this repository:

- `internal/provider/` - provider implementation
- `reference/` - research wiki and API notes
- `.github/badges/` - version badge (no Actions workflows in this repo)
- `terraform-test-project/` - local validation examples
- `CHANGELOG.md` - release notes
- `RELEASING.md` - release procedure

## Additional References

- [CHANGELOG.md](./CHANGELOG.md)
- [RELEASING.md](./RELEASING.md)
- [reference/README.md](./reference/README.md)
- [terraform-test-project/README.md](./terraform-test-project/README.md)
