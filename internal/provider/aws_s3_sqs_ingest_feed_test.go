package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSelectAwsS3SqsIngestFeedByID(t *testing.T) {
	result, found, err := selectAwsS3SqsIngestFeed([]awsS3SqsIngestFeedReadResult{
		{ID: "feed-1", Name: "first"},
		{ID: "feed-2", Name: "second"},
	}, "feed-2", "")
	if err != nil {
		t.Fatalf("unexpected error selecting ingest feed: %v", err)
	}
	if !found {
		t.Fatalf("expected ingest feed to be found")
	}
	if result.Name != "second" {
		t.Fatalf("expected second, got %q", result.Name)
	}
}

func TestSelectAwsS3SqsIngestFeedByName(t *testing.T) {
	result, found, err := selectAwsS3SqsIngestFeed([]awsS3SqsIngestFeedReadResult{
		{ID: "feed-1", Name: "first"},
		{ID: "feed-2", Name: "second"},
	}, "", "first")
	if err != nil {
		t.Fatalf("unexpected error selecting ingest feed: %v", err)
	}
	if !found {
		t.Fatalf("expected ingest feed to be found")
	}
	if result.ID != "feed-1" {
		t.Fatalf("expected feed-1, got %q", result.ID)
	}
}

func TestAwsS3SqsCreateInputFromModel(t *testing.T) {
	model := awsS3SqsIngestFeedResourceModel{
		RepositoryName:     types.StringValue("security-logs"),
		Name:               types.StringValue("cloudtrail"),
		Description:        types.StringValue("CloudTrail ingest"),
		Enabled:            types.BoolValue(true),
		Parser:             types.StringValue("json-example"),
		Region:             types.StringValue("us-east-1"),
		SQSURL:             types.StringValue("https://sqs.us-east-1.amazonaws.com/123456789012/logscale"),
		Compression:        types.StringValue("Auto"),
		AuthenticationKind: types.StringValue("IamRole"),
		RoleARN:            types.StringValue("arn:aws:iam::123456789012:role/logscale"),
		PreprocessingKind:  types.StringValue("SplitAwsRecords"),
	}

	input := awsS3SqsCreateInputFromModel(model)

	if input["repositoryName"] != "security-logs" {
		t.Fatalf("expected repositoryName security-logs, got %#v", input["repositoryName"])
	}
	if input["parser"] != "json-example" {
		t.Fatalf("expected parser json-example, got %#v", input["parser"])
	}

	description, ok := input["description"].(string)
	if !ok {
		t.Fatalf("expected description string, got %#v", input["description"])
	}
	if description != "CloudTrail ingest" {
		t.Fatalf("expected description CloudTrail ingest, got %#v", description)
	}
}

func TestAwsS3SqsUpdateInputClearsDescriptionWhenNull(t *testing.T) {
	model := awsS3SqsIngestFeedResourceModel{
		RepositoryName:     types.StringValue("security-logs"),
		Name:               types.StringValue("cloudtrail"),
		Description:        types.StringNull(),
		Enabled:            types.BoolValue(true),
		Parser:             types.StringValue("json-example"),
		Region:             types.StringValue("us-east-1"),
		SQSURL:             types.StringValue("https://sqs.us-east-1.amazonaws.com/123456789012/logscale"),
		Compression:        types.StringValue("Auto"),
		AuthenticationKind: types.StringValue("IamRole"),
		RoleARN:            types.StringValue("arn:aws:iam::123456789012:role/logscale"),
		PreprocessingKind:  types.StringValue("SplitNewline"),
	}

	input := awsS3SqsUpdateInputFromModel("feed-1", model)
	if _, present := input["description"]; present {
		t.Fatalf("expected description to be absent for null input, got %#v", input["description"])
	}
}

func TestValidateAwsS3SqsIngestFeedModel(t *testing.T) {
	resource := &awsS3SqsIngestFeedResource{}
	model := awsS3SqsIngestFeedResourceModel{
		RepositoryName:     types.StringValue("security-logs"),
		Name:               types.StringValue("cloudtrail"),
		Enabled:            types.BoolValue(true),
		Parser:             types.StringValue("json-example"),
		Region:             types.StringValue("us-east-1"),
		SQSURL:             types.StringValue("https://sqs.us-east-1.amazonaws.com/123456789012/logscale"),
		Compression:        types.StringValue("Auto"),
		AuthenticationKind: types.StringValue("IamRole"),
		RoleARN:            types.StringValue("arn:aws:iam::123456789012:role/logscale"),
		PreprocessingKind:  types.StringValue("SplitAwsRecords"),
	}

	var diagnostics diag.Diagnostics
	resource.validateAwsS3SqsIngestFeedModel(model, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("expected valid model, got diagnostics: %v", diagnostics)
	}
}

func TestApplyAwsS3SqsIngestFeedReadResultToStateClearsMissingSourceFields(t *testing.T) {
	state := awsS3SqsIngestFeedResourceModel{
		Description:        types.StringNull(),
		Parser:             types.StringValue("json-example"),
		Region:             types.StringValue("us-east-1"),
		SQSURL:             types.StringValue("https://sqs.us-east-1.amazonaws.com/123456789012/logscale"),
		Compression:        types.StringValue("Auto"),
		AuthenticationKind: types.StringValue("IamRole"),
		RoleARN:            types.StringValue("arn:aws:iam::123456789012:role/logscale"),
		AWSExternalID:      types.StringValue("external-id"),
		PreprocessingKind:  types.StringValue("SplitNewline"),
	}

	resource := &awsS3SqsIngestFeedResource{}
	resource.applyAwsS3SqsIngestFeedReadResultToState(&state, awsS3SqsIngestFeedReadResult{
		ID:          "feed-1",
		Name:        "cloudtrail",
		Description: "",
		Enabled:     true,
	})

	if !state.Region.IsNull() {
		t.Fatalf("expected region to be cleared when source is missing")
	}
	// authentication_kind, role_arn, and aws_external_id are never populated from the API
	// (awsAuthentication is not queried; fields are String! non-null and require Manage Cluster).
	// They are always preserved from the incoming state/plan — including when source is nil.
	if state.AuthenticationKind.ValueString() != "IamRole" {
		t.Fatalf("expected authentication_kind to be preserved from state when source is missing, got %q", state.AuthenticationKind.ValueString())
	}
	if state.AWSExternalID.ValueString() != "external-id" {
		t.Fatalf("expected aws_external_id to be preserved from state when source is missing, got %q", state.AWSExternalID.ValueString())
	}
}

func TestSelectAwsS3SqsIngestFeedReturnsNotFound(t *testing.T) {
	_, found, err := selectAwsS3SqsIngestFeed([]awsS3SqsIngestFeedReadResult{
		{ID: "feed-1", Name: "first"},
	}, "feed-2", "")
	if err != nil {
		t.Fatalf("unexpected error selecting ingest feed: %v", err)
	}
	if found {
		t.Fatalf("expected ingest feed lookup to miss")
	}
}

func TestApplyAwsS3SqsIngestFeedReadResultPreservesCurrentParserReference(t *testing.T) {
	state := awsS3SqsIngestFeedResourceModel{
		Description: types.StringNull(),
		Parser:      types.StringValue("json-example"),
	}

	resource := &awsS3SqsIngestFeedResource{}
	resource.applyAwsS3SqsIngestFeedReadResultToState(&state, awsS3SqsIngestFeedReadResult{
		ID:          "feed-1",
		Name:        "cloudtrail",
		Description: "",
		Enabled:     true,
		Parser: &struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   "parser-1",
			Name: "json-example",
		},
	})

	if state.Parser.ValueString() != "json-example" {
		t.Fatalf("expected parser reference to preserve current value, got %q", state.Parser.ValueString())
	}
}

func TestNormalizeOptionalStringFromAPIWithNullCurrentState(t *testing.T) {
	value := normalizeOptionalStringFromAPI(types.StringNull(), "")
	if !value.IsNull() {
		t.Fatalf("expected null optional string to stay null when API returns empty string")
	}
}

func TestStringsFromSetNoop(t *testing.T) {
	setValue, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"a", "b"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building set: %v", diags)
	}

	values := stringsFromSet(context.Background(), setValue, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics reading set: %v", diags)
	}
	assertStringSlicesEqual(t, []string{"a", "b"}, values)
}
