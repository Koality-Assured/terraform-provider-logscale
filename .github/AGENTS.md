# AGENTS — `.github/`

This directory holds GitHub-surface files for the provider repository. **There are no GitHub Actions workflows in this repository** — CI, CodeQL, release packaging, and PR-summary automation are intentionally absent so nothing auto-runs.

## Contents

| Path | Purpose |
| --- | --- |
| `badges/version.svg` | Version badge referenced from `README.md`; keep it consistent with [`../VERSION`](../VERSION) |

## Hard rules

- **Do not add workflow YAML under `workflows/`** unless a human explicitly asks to reintroduce automation for this repository.
- **No secrets in committed files.** Never place API tokens, ingest tokens, or long-lived cloud credentials in this tree.
- **No tenant-mutating automation.** Acceptance tests that hit a real LogScale tenant must remain opt-in and operator-triggered.

## When you touch this directory

- Keep the badge under `badges/version.svg` consistent with the value in [`../VERSION`](../VERSION) when cutting a release (see [`../RELEASING.md`](../RELEASING.md))
- Add a `## [Unreleased]` entry to [`../CHANGELOG.md`](../CHANGELOG.md) for material GitHub-surface changes
- Add a [`../CHANGE-HISTORY.md`](../CHANGE-HISTORY.md) provenance entry for material additions/removals
