package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &awsS3SqsIngestFeedDataSource{}
	_ datasource.DataSourceWithConfigure = &awsS3SqsIngestFeedDataSource{}
)

func NewAwsS3SqsIngestFeedDataSource() datasource.DataSource {
	return &awsS3SqsIngestFeedDataSource{}
}

type awsS3SqsIngestFeedDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type awsS3SqsIngestFeedDataSourceModel struct {
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
	PreprocessingKind  types.String `tfsdk:"preprocessing_kind"`
	CreatedAt          types.Int64  `tfsdk:"created_at"`
	ForceStopped       types.Bool   `tfsdk:"force_stopped"`
	StatusProblem      types.String `tfsdk:"status_problem"`
	StatusCause        types.String `tfsdk:"status_cause"`
	StatusTimestamp    types.Int64  `tfsdk:"status_timestamp"`
}

func (d *awsS3SqsIngestFeedDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_s3_sqs_ingest_feed"
}

func (d *awsS3SqsIngestFeedDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an AWS S3/SQS ingest feed by ID or name within a repository.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Ingest feed ID. Specify exactly one of id or name.",
				Optional:    true,
				Computed:    true,
			},
			"repository_name": schema.StringAttribute{
				Description: "Repository that owns the ingest feed.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Ingest feed name. Specify exactly one of id or name.",
				Optional:    true,
				Computed:    true,
			},
			"description": schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{Computed: true},
			"parser": schema.StringAttribute{Computed: true},
			"region": schema.StringAttribute{Computed: true},
			"sqs_url": schema.StringAttribute{Computed: true},
			"compression": schema.StringAttribute{Computed: true},
			"authentication_kind": schema.StringAttribute{Computed: true},
			"role_arn": schema.StringAttribute{Computed: true},
			"aws_external_id": schema.StringAttribute{Computed: true},
			"preprocessing_kind": schema.StringAttribute{Computed: true},
			"created_at": schema.Int64Attribute{Computed: true},
			"force_stopped": schema.BoolAttribute{Computed: true},
			"status_problem": schema.StringAttribute{Computed: true},
			"status_cause": schema.StringAttribute{Computed: true},
			"status_timestamp": schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *awsS3SqsIngestFeedDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.config = config
	d.client = newConfiguredHTTPClient(config)
}

func (d *awsS3SqsIngestFeedDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data awsS3SqsIngestFeedDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := strings.TrimSpace(data.ID.ValueString())
	name := strings.TrimSpace(data.Name.ValueString())

	if (id == "") == (name == "") {
		resp.Diagnostics.AddError("Invalid AWS S3/SQS ingest feed lookup", "Specify exactly one of id or name.")
		return
	}

	resource := &awsS3SqsIngestFeedResource{
		client: d.client,
		config: d.config,
	}
	result, found, err := resource.readAwsS3SqsIngestFeed(ctx, data.RepositoryName.ValueString(), id, name)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AWS S3/SQS ingest feed", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("AWS S3/SQS ingest feed not found", "No AWS S3/SQS ingest feed matched the supplied lookup criteria.")
		return
	}

	state := awsS3SqsIngestFeedResourceModel{
		ID:             data.ID,
		RepositoryName: data.RepositoryName,
		Name:           data.Name,
	}
	resource.applyAwsS3SqsIngestFeedReadResultToState(&state, result)

	data.ID = state.ID
	data.RepositoryName = state.RepositoryName
	data.Name = state.Name
	data.Description = state.Description
	data.Enabled = state.Enabled
	data.Parser = state.Parser
	data.Region = state.Region
	data.SQSURL = state.SQSURL
	data.Compression = state.Compression
	data.AuthenticationKind = state.AuthenticationKind
	data.RoleARN = state.RoleARN
	data.AWSExternalID = state.AWSExternalID
	data.PreprocessingKind = state.PreprocessingKind
	data.CreatedAt = state.CreatedAt
	data.ForceStopped = state.ForceStopped
	data.StatusProblem = state.StatusProblem
	data.StatusCause = state.StatusCause
	data.StatusTimestamp = state.StatusTimestamp

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
