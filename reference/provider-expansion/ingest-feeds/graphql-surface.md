# AWS S3/SQS Ingest Feeds GraphQL Surface

## Scope

This note tracks the current provider-facing understanding of LogScale AWS S3/SQS ingest feeds for the cloud/SaaS tenant model.

## Confirmed Official-Doc Surface

The official LogScale GraphQL docs currently expose the following AWS feed operations:

- `createAwsS3SqsIngestFeed`
- `updateAwsS3SqsIngestFeed`
- `deleteIngestFeed`
- `testAwsS3SqsIngestFeed`

The documented lookup/query surface also shows ingest feeds being queried from repositories, including filtering by the AWS S3/SQS feed type.

## Current Provider Slice

Implemented in provider code:

- `logscale_aws_s3_sqs_ingest_feed`
- `data.logscale_aws_s3_sqs_ingest_feed`

Current scope of the Terraform resource:

- repository-scoped CRUD
- import by `repository_name:ingest_feed_id`
- lookup by repository plus ID or name
- modeled around the documented AWS IAM-role authentication path
- modeled around the documented preprocessing kinds `SplitNewline` and `SplitAwsRecords`

## Current Schema Model

Current resource/data-source fields:

- `id`
- `repository_name`
- `name`
- `description`
- `enabled`
- `parser`
- `region`
- `sqs_url`
- `compression`
- `authentication_kind`
- `role_arn`
- `aws_external_id`
- `preprocessing_kind`
- `created_at`
- `force_stopped`
- `status_problem`
- `status_cause`
- `status_timestamp`

## Runtime/Modeling Notes

- The provider currently treats parser selection as an external reference rather than provider-managed ownership.
- The resource preserves a supplied parser reference when the API readback still resolves to the same parser by ID or name.
- Optional description state is normalized so an omitted Terraform description is less likely to drift when the API returns an empty string.
- The current implementation uses the repository ingest-feed list/read path and filters client-side by ID or name instead of relying on a narrower feed-specific query shape.
- The provider applies local guardrails for the documented enum-like fields:
  - `compression`: `Auto`, `Gzip`, `None`
  - `authentication_kind`: currently `IamRole`
  - `preprocessing_kind`: `SplitNewline`, `SplitAwsRecords`

## Validation/Test Endpoint Notes

The official docs expose `testAwsS3SqsIngestFeed`, which is valuable research-wise because it suggests the API can validate feed configuration without creating the resource.

Current provider decision:

- document the test capability in the research wiki
- do not call the test mutation automatically during normal CRUD applies yet

Why it is deferred for now:

- we have not yet validated the tenant permissions and side-effect profile strongly enough to make it part of default apply behavior
- the provider should avoid surprising users with extra network-side validation calls until the tradeoffs are clearer

## Known API Schema Drift (discovered in production, fixed in provider)

These are confirmed mismatches between the provider's original implementation and the live LogScale API schema. Document here so future provider work doesn't reintroduce them.

### `statusMessage.cause` — complex type, not scalar (fixed v0.3.3)

The provider originally queried `cause` as a scalar string inside `statusMessage { problem cause statusTimestamp }`. The live API returns `cause` as a complex object type `IngestFeedStatusCause`, causing:

```
GraphQL error: Field 'cause' of type 'IngestFeedStatusCause' must have a sub selection.
```

**Fix:** removed `cause` from both the mutation response query and the list/read query. The `status_cause` Terraform attribute now always returns `null`. When the correct sub-field structure of `IngestFeedStatusCause` is confirmed via schema introspection, restore the query with proper fragment selection.

### `description` in Create/Update input — plain string, not nested object (fixed v0.3.4)

The provider was constructing the `description` input field as:
```go
input["description"] = map[string]interface{}{"description": model.Description.ValueString()}
```
instead of:
```go
input["description"] = model.Description.ValueString()
```

The live API expects `description` as a plain string in `CreateAwsS3SqsIngestFeed` and `UpdateAwsS3SqsIngestFeed`. The nested form caused:

```
GraphQL error: Variable '$input' expected value of type 'CreateAwsS3SqsIngestFeed!' ... 'description' Invalid value
```

This affected any feed that had a `description` set — feeds without a description (null) were unaffected since the null path skipped the field entirely.

## Permission Gate — "Manage Cluster" error on ingest feed operations

The `createAwsS3SqsIngestFeed` and/or `updateAwsS3SqsIngestFeed` mutations return `GraphQL error: Manage cluster not allowed.` with certain API tokens. Observed behavior:

**Create:** The feed appears to be **created successfully** on the LogScale side (resource is visible in the UI with the correct name and description) but the provider reports the operation as failed. This suggests the error occurs during the post-create state read (the `GetAwsS3SqsIngestFeeds` list query), not during the create mutation itself. Terraform marks the resource as tainted/failed even though it exists.

**Workaround for an existing feed:** import it into Terraform state:
```
terraform import 'module.sqsfeeds.logscale_aws_s3_sqs_ingest_feed.<label>[0]' \
  <repository_name>:<feed_id>
```
Feed ID is visible in the LogScale UI on the feed's settings page.

**If the read also fails after import**, the token needs the Manage Cluster role. Resolution:
1. Generate a LogScale Personal Access Token (PAT) with the Manage Cluster role
2. Update the `LOGSCALE_API_TOKEN` (or equivalent) secret in the consumer Terraform / CI environment that calls the provider

## Open Questions

- whether `testAwsS3SqsIngestFeed` should become a validating data source, acceptance helper, or optional preflight flow later
- whether additional AWS auth or preprocessing variants exist in your target SaaS tenant beyond the currently documented narrow slice
- whether broader ingest-feed families should share a common helper/model layer before adding Azure or collector-backed feed types

## Official References

- `createAwsS3SqsIngestFeed`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-createawss3sqsingestfeed.html
- `updateAwsS3SqsIngestFeed`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-updateawss3sqsingestfeed.html
- `deleteIngestFeed`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-deleteingestfeed.html
- `testAwsS3SqsIngestFeed`: https://library.humio.com/logscale-graphql-reference-mutations/graphql-mutation-field-testawss3sqsingestfeed.html
- `CreateAwsS3SqsIngestFeed`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-createawss3sqsingestfeed.html
- `UpdateAwsS3SqsIngestFeed`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-updateawss3sqsingestfeed.html
- `DeleteIngestFeed`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-deleteingestfeed.html
- `TestAwsS3SqsIngestFeed`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-testawss3sqsingestfeed.html
- `IngestFeedAwsAuthenticationInput`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-input-ingestfeedawsauthenticationinput.html
- `IngestFeedAwsAuthenticationKind`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-enum-ingestfeedawsauthenticationkind.html
- `IngestFeedCompression`: https://library.humio.com/logscale-graphql-reference-datatypes/graphql-enum-ingestfeedcompression.html
