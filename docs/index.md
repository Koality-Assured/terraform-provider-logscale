---
page_title: "logscale Provider"
subcategory: ""
description: |-
  Terraform provider for managing Falcon LogScale resources through the GraphQL API.
---

# logscale Provider

A Terraform provider for managing Falcon LogScale resources through the GraphQL API.

This repository is for provider implementation, documentation, research, and automated releases. Normal consumers should use the published release artifacts from this repo. Manual local builds are for provider development and debugging.

## Example Usage

```terraform
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

## Schema

### Required

- `api_url` (String) LogScale GraphQL endpoint, for example `https://tenant.logscale.us-1.crowdstrike.com/graphql`.
- `api_token` (String, Sensitive) API token used for GraphQL authentication.

## Documentation surface

These docs follow the `tfplugindocs`-style directory layout (`docs/index.md`, `docs/resources/*.md`, `docs/data-sources/*.md`). Pages are authored manually today; once `terraform-plugin-docs` is wired into the build, regenerating from schema will overwrite them in place. Until then, treat these docs as canonical and keep the in-repo `README.md` in sync.

For deeper behavior notes, partial-CRUD semantics, and tenant-validation caveats, see [`README.md`](../README.md) (the "Important Behavior Notes" section) and [`PROMPT_SPEC.md`](../PROMPT_SPEC.md).
