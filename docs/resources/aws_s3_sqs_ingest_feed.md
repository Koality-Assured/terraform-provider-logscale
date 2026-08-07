---
page_title: "logscale_aws_s3_sqs_ingest_feed Resource - terraform-provider-logscale"
subcategory: ""
description: |-
  Manages a LogScale AWS S3/SQS ingest feed with IAM role authentication.
---

# logscale_aws_s3_sqs_ingest_feed (Resource)

Manages a LogScale AWS S3/SQS ingest feed with IAM role authentication.

See the matching `Usage Examples` and `Important Behavior Notes` sections of [`README.md`](../../README.md) for the current schema, lifecycle constraints, and tenant-validation caveats.

## Behavior notes

- Import format is `repository_name:ingest_feed_id`. v1 models the IAM-role + `SplitNewline`/`SplitAwsRecords` paths.

## Schema

Generated schema details will be populated when `terraform-plugin-docs` is wired into the build. Until then, the authoritative schema is the corresponding Go file under [`internal/provider/`](../../internal/provider/).
