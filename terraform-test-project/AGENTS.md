# AGENTS — `terraform-test-project/`

This directory is the in-repo Terraform configuration used to validate the provider end-to-end with a `dev_overrides` pointer at a locally built binary.

## Hard rules

- **No real tenant URLs, no real tokens, no real customer identifiers** in committed `.tf` files. The provider/test project uses placeholder variables (`var.logscale_api_url`, `var.logscale_api_token`); supply real values from a local `terraform.tfvars` that is gitignored, or from the operator's shell environment via `TF_VAR_logscale_api_url` / `TF_VAR_logscale_api_token`.
- **No `.terraform/` artifacts committed.** The `.terraform/` directory under this folder is local state from `terraform init` runs and must stay untracked. If a stray binary or lockfile appears under `.terraform/`, do not commit it.
- **Provider address pinned to `local/logscale`.** The examples here intentionally use the private address documented in `README.md` so that `dev_overrides` works without `terraform init` hitting the public registry.

## What this project is for

- Exercising new resources during development against either a sandbox tenant or a deliberately constrained tenant the operator owns
- Catching schema regressions before they ship
- Demonstrating end-to-end usage patterns that mirror the `README.md` examples

## What this project is not for

- Acceptance tests — those live under [`../internal/provider/`](../internal/provider/) as `_test.go` files, env-gated
- Customer-facing reference configuration — the canonical examples are in `README.md`

## When you touch this project

- Update [`README.md`](./README.md) in this folder if you add a new example resource pattern
- Keep `provider.tf` aligned with the variables documented in the root `README.md` (currently `api_url` + `api_token`)
- If a new resource needs configuration patterns that materially diverge from the root README examples, add a short note in [`README.md`](./README.md) explaining the divergence and why

## State files

If a `terraform.tfstate` ever appears in this directory tree, treat it as untrusted local-only output:

- It may contain tenant identifiers
- It may contain sensitive token material (the `api_token` provider arg is sensitive, but resource state for `logscale_ingest_token` retains the token value by design)
- Never commit it

The repo's `.gitignore` should keep these excluded; double-check before any `git add -A`.
