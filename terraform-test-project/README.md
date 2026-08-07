# terraform-test-project

This directory is for local provider validation and experimentation.

## Purpose

Use this directory when you want to:

- validate a local provider build
- try a new resource or data source against a tenant
- reproduce provider bugs quickly
- keep small working examples close to provider development

For broader provider details, use the root `README.md`.

## Prerequisites

- Terraform installed
- a LogScale API token with the permissions needed for the resources you are testing
- a locally built provider binary when using dev overrides, or a released provider artifact if you are testing a published version

## Provider Address

Examples in this repository use:

```hcl
source = "local/logscale"
```

If your team later adopts a different internal namespace, update the examples accordingly.

## Local Development Override

Example `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "local/logscale" = "/path/to/your/provider/binaries"
  }
  direct {}
}
```

With dev overrides enabled:

- skip `terraform init`
- go directly to `terraform plan` or `terraform apply`

## Basic Provider Configuration

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

## Example Resources

### Repository

```hcl
resource "logscale_repository" "application_logs" {
  name           = "app-logs"
  description    = "Application log data"
  retention_days = 90
}
```

### Parser

```hcl
resource "logscale_parser" "json_example" {
  repository_name = logscale_repository.application_logs.name
  name            = "json-example"
  script          = file("${path.module}/parsers/json-example.ls")
}
```

You can also feed the parser script from YAML:

```hcl
locals {
  parser_def = yamldecode(file("${path.module}/parsers/json-example.yaml"))
}

resource "logscale_parser" "json_from_yaml" {
  repository_name = logscale_repository.application_logs.name
  name            = local.parser_def.name
  script          = local.parser_def.script
}
```

### View

```hcl
resource "logscale_view" "general" {
  name = "general"

  repository_connection {
    repository_name = logscale_repository.application_logs.name
    filter          = "*"
  }
}
```

### Users Data Source

```hcl
data "logscale_users" "all" {
  page_number = 1
  page_size   = 50
}
```

### Single User Lookup

```hcl
data "logscale_user" "analyst" {
  email = "analyst@example.com"
}
```

## Other Implemented Resources

This repo also implements older foundational resources that are useful for tenant validation:

- `logscale_group` for basic group lifecycle management
- `logscale_ingest_token` for token creation and deletion
- `logscale_organization_role_assignment` for assigning an organization role to a group
- `logscale_system_role_assignment` for assigning a system role to a group
- `logscale_group_membership` for reconciling usernames in a group, with the caveats below

Important caveats when validating those resources:

- `logscale_ingest_token` does not currently read remote token records back from LogScale; the provider keeps the created token value in Terraform state and requires recreation for changes
- `logscale_organization_role_assignment` and `logscale_system_role_assignment` currently preserve Terraform state on read instead of verifying the remote assignment from LogScale
- `logscale_system_role_assignment` often needs broader permissions than the organization-role path

## Group Membership Guidance

`logscale_group_membership` is available, but it should still be treated carefully.

What the provider does now:

- applies add/remove membership mutations
- reads back the remote membership
- warns when requested usernames do not match what LogScale returns

What you should still assume:

- username format can vary by tenant behavior
- manual verification may still be needed after apply until your tenant behavior is well understood

That means this resource is no longer in the old "never use it" state, but it is also not something we should treat as universally risk-free.

## Suggested Validation Order

When you are validating the older group and access-control resources, the safest sequence is usually:

1. Create or look up a `logscale_group`.
2. Look up the target role with `data.logscale_role` if you are testing role assignment.
3. Apply `logscale_organization_role_assignment` or `logscale_system_role_assignment`.
4. Apply `logscale_group_membership` only after the group itself is confirmed.
5. Validate the result in LogScale UI when the resource does not currently have authoritative remote readback.

## Suggested Workflow

1. Point Terraform at a local provider build or a released provider artifact.
2. Start with a small, isolated test configuration.
3. Run `terraform plan`.
4. Apply only the resource family you are actively validating.
5. Confirm the result in LogScale UI or by readback-capable Terraform resources/data sources.
6. Capture anything new we learn in the repo wiki under `reference/`.

## Files In This Directory

Typical files here include:

- `provider.tf` for provider wiring
- `main.tf` or resource-specific `.tf` files for validation scenarios
- local state and lock files generated during testing

Keep sensitive values out of version control.
