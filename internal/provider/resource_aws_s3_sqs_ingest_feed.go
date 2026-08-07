package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &awsS3SqsIngestFeedResource{}
	_ resource.ResourceWithConfigure   = &awsS3SqsIngestFeedResource{}
	_ resource.ResourceWithImportState = &awsS3SqsIngestFeedResource{}
)

func NewAwsS3SqsIngestFeedResource() resource.Resource {
	return &awsS3SqsIngestFeedResource{}
}

type awsS3SqsIngestFeedResource struct {
	client *http.Client
	config *LogScaleConfig
}

type awsS3SqsIngestFeedResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	RepositoryName     types.String `tfsdk:"repository_name"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	Parser             types.String `tfsdk:"parser"`
	Region             types.String `tfsdk:"region"`
	SQSURL             types.String `tfsdk:"sqs_url"`
	Compression        types.String `tfsdk:"compression"`
	AuthenticationKind types.String `tfsdk:"authentication_kind"`
	RoleARN            types.String `tfsdk:"role_arn"`
	AWSExternalID      types.String `tfsdk:"aws_external_id"`
	PreprocessingKind types.String `tfsdk:"preprocessing_kind"`
	CreatedAt          types.Int64  `tfsdk:"created_at"`
	ForceStopped       types.Bool   `tfsdk:"force_stopped"`
	StatusProblem      types.String `tfsdk:"status_problem"`
	StatusCause        types.String `tfsdk:"status_cause"`
	StatusTimestamp    types.Int64  `tfsdk:"status_timestamp"`
}

type awsS3SqsIngestFeedReadResult struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Enabled      bool   `json:"enabled"`
	CreatedAt    int64  `json:"createdAt"`
	ForceStopped bool   `json:"forceStopped"`
	Parser       *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"parser"`
	// ExecutionInfo omitted: querying executionInfo requires the "Manage Cluster"
	// permission in LogScale. Removing it from both the mutation response and the
	// list/read query avoids the "Manage cluster not allowed" GraphQL error that
	// would otherwise cause Create and Read to fail even though the resource
	// exists. status_problem, status_cause, status_timestamp are always null.
	Source *struct {
		TypeName      string `json:"__typename"`
		Region        string `json:"region"`
		SQSURL        string `json:"sqsUrl"`
		Compression   string `json:"compression"`
		Preprocessing *struct {
			TypeName string `json:"__typename"`
			Kind     string `json:"kind"`
		} `json:"preprocessing"`
		// AwsAuthentication omitted: all fields (roleArn, externalId) are String! non-null.
		// When the token lacks Manage Cluster permission the server nulls them, which cascades
		// through non-null awsAuthentication -> source -> IngestFeed, returning null for the
		// entire feed object. role_arn and authentication_kind are preserved from Terraform
		// state/plan instead of being populated from the API.
	} `json:"source"`
}

func (r *awsS3SqsIngestFeedResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_s3_sqs_ingest_feed"
}

func (r *awsS3SqsIngestFeedResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale AWS S3/SQS ingest feed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Ingest feed ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository_name": schema.StringAttribute{
				Description: "Repository that owns the ingest feed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Ingest feed name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional ingest feed description.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether ingest from the feed is enabled.",
				Required:    true,
			},
			"parser": schema.StringAttribute{
				Description: "Parser name or ID used by the ingest feed.",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: "AWS region used by the ingest feed.",
				Required:    true,
			},
			"sqs_url": schema.StringAttribute{
				Description: "AWS SQS queue URL.",
				Required:    true,
			},
			"compression": schema.StringAttribute{
				Description: "Compression mode, for example Auto, Gzip, or None.",
				Required:    true,
			},
			"authentication_kind": schema.StringAttribute{
				Description: "AWS authentication kind. Currently IamRole is the documented option.",
				Required:    true,
			},
			"role_arn": schema.StringAttribute{
				Description: "IAM role ARN used for AWS authentication.",
				Required:    true,
			},
			"aws_external_id": schema.StringAttribute{
				Description: "External ID returned by LogScale for the IAM role authentication.",
				Computed:    true,
			},
			"preprocessing_kind": schema.StringAttribute{
				Description: "Preprocessing mode, for example SplitNewline or SplitAwsRecords.",
				Required:    true,
			},
			"created_at": schema.Int64Attribute{
				Description: "Unix timestamp for when the ingest feed was created.",
				Computed:    true,
			},
			"force_stopped": schema.BoolAttribute{
				Description: "Whether the ingest feed is force stopped.",
				Computed:    true,
			},
			"status_problem": schema.StringAttribute{
				Description: "Latest ingest feed status problem returned by LogScale.",
				Computed:    true,
			},
			"status_cause": schema.StringAttribute{
				Description: "Latest ingest feed status cause returned by LogScale.",
				Computed:    true,
			},
			"status_timestamp": schema.Int64Attribute{
				Description: "Timestamp for the latest ingest feed status message.",
				Computed:    true,
			},
		},
	}
}

func (r *awsS3SqsIngestFeedResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.config = config
	r.client = newConfiguredHTTPClient(config)
}

func (r *awsS3SqsIngestFeedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan awsS3SqsIngestFeedResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.validateAwsS3SqsIngestFeedModel(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	input := awsS3SqsCreateInputFromModel(plan)

	var mutationData struct {
		CreateAwsS3SqsIngestFeed struct {
			ID string `json:"id"`
		} `json:"createAwsS3SqsIngestFeed"`
	}

	warnings, err := executeGraphQLLenient(ctx, r.client, r.config.Endpoint, awsS3SqsIngestFeedSelectionMutation("createAwsS3SqsIngestFeed", "CreateAwsS3SqsIngestFeed"), map[string]interface{}{
		"input": input,
	}, &mutationData)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AWS S3/SQS ingest feed", err.Error())
		return
	}
	if mutationData.CreateAwsS3SqsIngestFeed.ID == "" {
		resp.Diagnostics.AddError("Error creating AWS S3/SQS ingest feed", "mutation returned no ID: "+strings.Join(warnings, "; "))
		return
	}
	for _, w := range warnings {
		resp.Diagnostics.AddWarning("LogScale API warning during ingest feed create", w)
	}

	// Populate state directly from the plan. Reading back after create requires
	// Manage Cluster permission (to resolve ingestFeeds); all values are already
	// known from the input, so no read is needed.
	plan.ID = types.StringValue(mutationData.CreateAwsS3SqsIngestFeed.ID)
	plan.AWSExternalID = types.StringNull()
	plan.CreatedAt = types.Int64Null()
	plan.ForceStopped = types.BoolNull()
	plan.StatusProblem = types.StringNull()
	plan.StatusCause = types.StringNull()
	plan.StatusTimestamp = types.Int64Null()
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *awsS3SqsIngestFeedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state awsS3SqsIngestFeedResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, found, err := r.readAwsS3SqsIngestFeed(ctx, state.RepositoryName.ValueString(), state.ID.ValueString(), state.Name.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Manage cluster not allowed") {
			resp.Diagnostics.AddWarning(
				"LogScale ingest feed read skipped — insufficient permissions",
				"The API token lacks the Manage Cluster permission required to list ingest feeds. "+
					"Existing state is preserved. Grant Manage Cluster to enable drift detection.",
			)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
		resp.Diagnostics.AddError("Error reading AWS S3/SQS ingest feed", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	r.applyAwsS3SqsIngestFeedReadResultToState(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *awsS3SqsIngestFeedResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan awsS3SqsIngestFeedResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state awsS3SqsIngestFeedResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.validateAwsS3SqsIngestFeedModel(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	input := awsS3SqsUpdateInputFromModel(state.ID.ValueString(), plan)

	var mutationData struct {
		UpdateAwsS3SqsIngestFeed struct {
			ID string `json:"id"`
		} `json:"updateAwsS3SqsIngestFeed"`
	}

	warnings, err := executeGraphQLLenient(ctx, r.client, r.config.Endpoint, awsS3SqsIngestFeedSelectionMutation("updateAwsS3SqsIngestFeed", "UpdateAwsS3SqsIngestFeed"), map[string]interface{}{
		"input": input,
	}, &mutationData)
	if err != nil {
		resp.Diagnostics.AddError("Error updating AWS S3/SQS ingest feed", err.Error())
		return
	}
	if mutationData.UpdateAwsS3SqsIngestFeed.ID == "" {
		resp.Diagnostics.AddError("Error updating AWS S3/SQS ingest feed", "mutation returned no ID: "+strings.Join(warnings, "; "))
		return
	}
	for _, w := range warnings {
		resp.Diagnostics.AddWarning("LogScale API warning during ingest feed update", w)
	}

	// Populate state directly from the plan — same reasoning as Create.
	plan.RepositoryName = state.RepositoryName
	plan.ID = types.StringValue(mutationData.UpdateAwsS3SqsIngestFeed.ID)
	plan.AWSExternalID = types.StringNull()
	plan.CreatedAt = types.Int64Null()
	plan.ForceStopped = types.BoolNull()
	plan.StatusProblem = types.StringNull()
	plan.StatusCause = types.StringNull()
	plan.StatusTimestamp = types.Int64Null()
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *awsS3SqsIngestFeedResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state awsS3SqsIngestFeedResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		DeleteIngestFeed *bool `json:"deleteIngestFeed"`
	}

	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation DeleteIngestFeed($input: DeleteIngestFeed!) {
			deleteIngestFeed(input: $input)
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"repositoryName": state.RepositoryName.ValueString(),
			"id":             state.ID.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting AWS S3/SQS ingest feed", err.Error())
	}
}

func (r *awsS3SqsIngestFeedResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier in the format repository_name:ingest_feed_id.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func awsS3SqsIngestFeedSelectionMutation(fieldName string, inputType string) string {
	// Request only id from the mutation response. After a successful mutation the
	// provider populates state from the plan values rather than reading back from the
	// API — listing ingestFeeds requires the Manage Cluster permission that is not
	// available on standard tokens.
	return `
		mutation AwsS3SqsIngestFeedMutation($input: ` + inputType + `!) {
			` + fieldName + `(input: $input) {
				id
			}
		}
	`
}

func awsS3SqsCreateInputFromModel(model awsS3SqsIngestFeedResourceModel) map[string]interface{} {
	input := map[string]interface{}{
		"repositoryName": model.RepositoryName.ValueString(),
		"name":           model.Name.ValueString(),
		"enabled":        model.Enabled.ValueBool(),
		"parser":         model.Parser.ValueString(),
		"region":         model.Region.ValueString(),
		"sqsUrl":         model.SQSURL.ValueString(),
		"compression":    model.Compression.ValueString(),
		"authentication": map[string]interface{}{
			"kind":    model.AuthenticationKind.ValueString(),
			"roleArn": model.RoleARN.ValueString(),
		},
		"preprocessing": map[string]interface{}{
			"kind": model.PreprocessingKind.ValueString(),
		},
	}

	if !model.Description.IsNull() {
		input["description"] = model.Description.ValueString()
	}

	return input
}

func awsS3SqsUpdateInputFromModel(id string, model awsS3SqsIngestFeedResourceModel) map[string]interface{} {
	input := map[string]interface{}{
		"id":             id,
		"repositoryName": model.RepositoryName.ValueString(),
		"name":           model.Name.ValueString(),
		"enabled":        model.Enabled.ValueBool(),
		"parser":         model.Parser.ValueString(),
		"region":         model.Region.ValueString(),
		"sqsUrl":         model.SQSURL.ValueString(),
		"compression":    model.Compression.ValueString(),
		"authentication": map[string]interface{}{
			"kind":    model.AuthenticationKind.ValueString(),
			"roleArn": model.RoleARN.ValueString(),
		},
		"preprocessing": map[string]interface{}{
			"kind": model.PreprocessingKind.ValueString(),
		},
	}

	if !model.Description.IsNull() {
		input["description"] = model.Description.ValueString()
	}

	return input
}

func (r *awsS3SqsIngestFeedResource) readAwsS3SqsIngestFeed(ctx context.Context, repositoryName string, id string, name string) (awsS3SqsIngestFeedReadResult, bool, error) {
	var data struct {
		Repository *struct {
			IngestFeeds struct {
				Results []awsS3SqsIngestFeedReadResult `json:"results"`
			} `json:"ingestFeeds"`
		} `json:"repository"`
	}

	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		query GetAwsS3SqsIngestFeeds($repositoryName: String!) {
			repository(name: $repositoryName) {
				ingestFeeds(typeFilter: AwsS3Sqs, limit: 200) {
					results {
						id
						name
						description
						enabled
						createdAt
						forceStopped
						parser {
							id
							name
						}
						source {
							__typename
							... on IngestFeedS3SqsSource {
								region
								sqsUrl
								compression
								preprocessing {
									__typename
									... on IngestFeedPreprocessingSplitNewline {
										kind
									}
									... on IngestFeedPreprocessingSplitAwsRecords {
										kind
									}
								}
								# awsAuthentication omitted: roleArn and externalId are String! non-null.
								# Resolving either without Manage Cluster nulls the field, cascading through
								# non-null awsAuthentication -> source -> IngestFeed -> entire results list.
								# role_arn is preserved from Terraform state instead.
							}
						}
					}
				}
			}
		}
	`, map[string]interface{}{
		"repositoryName": repositoryName,
	}, &data)
	if err != nil {
		return awsS3SqsIngestFeedReadResult{}, false, err
	}

	if data.Repository == nil {
		return awsS3SqsIngestFeedReadResult{}, false, nil
	}

	return selectAwsS3SqsIngestFeed(data.Repository.IngestFeeds.Results, id, name)
}

func selectAwsS3SqsIngestFeed(feeds []awsS3SqsIngestFeedReadResult, id string, name string) (awsS3SqsIngestFeedReadResult, bool, error) {
	if id != "" {
		for _, feed := range feeds {
			if feed.ID == id {
				return feed, true, nil
			}
		}
	}

	if name != "" {
		for _, feed := range feeds {
			if feed.Name == name {
				return feed, true, nil
			}
		}
	}

	return awsS3SqsIngestFeedReadResult{}, false, nil
}

func (r *awsS3SqsIngestFeedResource) validateAwsS3SqsIngestFeedModel(model awsS3SqsIngestFeedResourceModel, diags *diag.Diagnostics) {
	if strings.TrimSpace(model.RepositoryName.ValueString()) == "" {
		diags.AddError("Invalid repository_name", "repository_name must not be empty.")
	}
	if strings.TrimSpace(model.Name.ValueString()) == "" {
		diags.AddError("Invalid name", "name must not be empty.")
	}
	if strings.TrimSpace(model.Parser.ValueString()) == "" {
		diags.AddError("Invalid parser", "parser must not be empty.")
	}
	if strings.TrimSpace(model.Region.ValueString()) == "" {
		diags.AddError("Invalid region", "region must not be empty.")
	}
	if strings.TrimSpace(model.SQSURL.ValueString()) == "" {
		diags.AddError("Invalid sqs_url", "sqs_url must not be empty.")
	}
	if !isOneOf(model.Compression.ValueString(), "Auto", "Gzip", "None") {
		diags.AddError("Invalid compression", "compression must be one of Auto, Gzip, or None.")
	}
	if !isOneOf(model.AuthenticationKind.ValueString(), "IamRole") {
		diags.AddError("Invalid authentication_kind", "authentication_kind must currently be IamRole.")
	}
	if strings.TrimSpace(model.RoleARN.ValueString()) == "" {
		diags.AddError("Invalid role_arn", "role_arn must not be empty.")
	}
	if !isOneOf(model.PreprocessingKind.ValueString(), "SplitNewline", "SplitAwsRecords") {
		diags.AddError("Invalid preprocessing_kind", "preprocessing_kind must be one of SplitNewline or SplitAwsRecords.")
	}
}

func isOneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}

	return false
}

func (r *awsS3SqsIngestFeedResource) applyAwsS3SqsIngestFeedReadResultToState(state *awsS3SqsIngestFeedResourceModel, result awsS3SqsIngestFeedReadResult) {
	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Description = normalizeOptionalStringFromAPI(state.Description, result.Description)
	state.Enabled = types.BoolValue(result.Enabled)
	if result.Parser != nil {
		current := state.Parser.ValueString()
		if current != "" && (current == result.Parser.ID || current == result.Parser.Name) {
			state.Parser = types.StringValue(current)
		} else if result.Parser.Name != "" {
			state.Parser = types.StringValue(result.Parser.Name)
		} else {
			state.Parser = types.StringValue(result.Parser.ID)
		}
	}
	if result.Source != nil {
		state.Region = types.StringValue(result.Source.Region)
		state.SQSURL = types.StringValue(result.Source.SQSURL)
		state.Compression = types.StringValue(result.Source.Compression)
		if result.Source.Preprocessing != nil {
			state.PreprocessingKind = types.StringValue(result.Source.Preprocessing.Kind)
		} else {
			state.PreprocessingKind = types.StringNull()
		}
		// role_arn, authentication_kind, and aws_external_id are preserved from the
		// incoming state/plan. awsAuthentication is not queried because its fields
		// are String! non-null and require Manage Cluster to resolve; querying them
		// causes GraphQL null propagation that nulls the entire feed result.
	} else {
		state.Region = types.StringNull()
		state.SQSURL = types.StringNull()
		state.Compression = types.StringNull()
		state.PreprocessingKind = types.StringNull()
		// role_arn, authentication_kind, aws_external_id preserved from state/plan
	}
	state.CreatedAt = types.Int64Value(result.CreatedAt)
	state.ForceStopped = types.BoolValue(result.ForceStopped)
	// executionInfo removed from queries — requires Manage Cluster permission.
	// Status fields are not useful enough to gate the entire resource on that permission.
	state.StatusProblem = types.StringNull()
	state.StatusCause = types.StringNull()
	state.StatusTimestamp = types.Int64Null()
}
