package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &lookupFileDataSource{}
	_ datasource.DataSourceWithConfigure = &lookupFileDataSource{}
	_ datasource.DataSource              = &lookupFilesDataSource{}
	_ datasource.DataSourceWithConfigure = &lookupFilesDataSource{}
)

func NewLookupFileDataSource() datasource.DataSource {
	return &lookupFileDataSource{}
}

func NewLookupFilesDataSource() datasource.DataSource {
	return &lookupFilesDataSource{}
}

type lookupFileDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type lookupFilesDataSource struct {
	client *http.Client
	config *LogScaleConfig
}

type lookupFileDataSourceModel struct {
	ViewName        types.String `tfsdk:"view_name"`
	FileName        types.String `tfsdk:"file_name"`
	CSVContent      types.String `tfsdk:"csv_content"`
	Labels          types.Set    `tfsdk:"labels"`
	ContentHash     types.String `tfsdk:"content_hash"`
	CreatedAt       types.String `tfsdk:"created_at"`
	CreatedBy       types.String `tfsdk:"created_by"`
	ModifiedAt      types.String `tfsdk:"modified_at"`
	ModifiedBy      types.String `tfsdk:"modified_by"`
	FileSizeBytes   types.Int64  `tfsdk:"file_size_bytes"`
	TotalLinesCount types.Int64  `tfsdk:"total_lines_count"`
	Resource        types.String `tfsdk:"resource"`
}

type lookupFilesDataSourceModel struct {
	ID       types.String             `tfsdk:"id"`
	ViewName types.String             `tfsdk:"view_name"`
	Files    []lookupFileSummaryModel `tfsdk:"files"`
}

type lookupFileSummaryModel struct {
	FileName      types.String `tfsdk:"file_name"`
	ContentHash   types.String `tfsdk:"content_hash"`
	ModifiedAt    types.String `tfsdk:"modified_at"`
	FileSizeBytes types.Int64  `tfsdk:"file_size_bytes"`
	Resource      types.String `tfsdk:"resource"`
}

func (d *lookupFileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lookup_file"
}

func (d *lookupFilesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lookup_files"
}

func (d *lookupFileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a LogScale lookup file by file name within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"view_name":         schema.StringAttribute{Description: "Repository or view that owns the lookup file.", Required: true},
			"file_name":         schema.StringAttribute{Description: "Lookup file name.", Required: true},
			"csv_content":       schema.StringAttribute{Description: "Full CSV content returned by LogScale.", Computed: true},
			"labels":            schema.SetAttribute{Description: "Labels attached to the lookup file.", Computed: true, ElementType: types.StringType},
			"content_hash":      schema.StringAttribute{Description: "Content hash returned by LogScale.", Computed: true},
			"created_at":        schema.StringAttribute{Description: "Creation time returned by LogScale.", Computed: true},
			"created_by":        schema.StringAttribute{Description: "Creating user returned by LogScale.", Computed: true},
			"modified_at":       schema.StringAttribute{Description: "Last modification time returned by LogScale.", Computed: true},
			"modified_by":       schema.StringAttribute{Description: "Last modifying user returned by LogScale.", Computed: true},
			"file_size_bytes":   schema.Int64Attribute{Description: "File size in bytes returned by LogScale.", Computed: true},
			"total_lines_count": schema.Int64Attribute{Description: "Total number of data rows returned by LogScale.", Computed: true},
			"resource":          schema.StringAttribute{Description: "Provider-facing resource identifier returned by LogScale.", Computed: true},
		},
	}
}

func (d *lookupFilesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists LogScale lookup files within a repository/view.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Description: "Synthetic identifier for this lookup-file listing request.", Computed: true},
			"view_name": schema.StringAttribute{Description: "Repository or view whose lookup files should be returned.", Required: true},
			"files": schema.ListNestedAttribute{
				Description: "Lookup files returned for the supplied view.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"file_name":       schema.StringAttribute{Computed: true},
						"content_hash":    schema.StringAttribute{Computed: true},
						"modified_at":     schema.StringAttribute{Computed: true},
						"file_size_bytes": schema.Int64Attribute{Computed: true},
						"resource":        schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *lookupFileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	d.config = config
	d.client = newConfiguredHTTPClient(config)
}

func (d *lookupFilesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	d.config = config
	d.client = newConfiguredHTTPClient(config)
}

func (d *lookupFileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data lookupFileDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resource := &lookupFileResource{client: d.client, config: d.config}
	result, found, err := resource.readLookupFile(ctx, data.ViewName.ValueString(), data.FileName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading lookup file", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Lookup file not found", "No lookup file matched the supplied lookup criteria.")
		return
	}
	state := lookupFileResourceModel{
		ViewName: data.ViewName,
		FileName: data.FileName,
		Labels:   types.SetNull(types.StringType),
	}
	applyLookupFileReadResultToState(ctx, &state, result, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	data.CSVContent = state.CSVContent
	data.Labels = state.Labels
	data.ContentHash = state.ContentHash
	data.CreatedAt = state.CreatedAt
	data.CreatedBy = state.CreatedBy
	data.ModifiedAt = state.ModifiedAt
	data.ModifiedBy = state.ModifiedBy
	data.FileSizeBytes = state.FileSizeBytes
	data.TotalLinesCount = state.TotalLinesCount
	data.Resource = state.Resource
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *lookupFilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data lookupFilesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	files, err := lookupFiles(ctx, d.client, d.config.Endpoint, data.ViewName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading lookup files", err.Error())
		return
	}
	summaries := make([]lookupFileSummaryModel, 0, len(files))
	for _, item := range files {
		fileSize := types.Int64Null()
		if item.FileSizeBytes != nil {
			fileSize = types.Int64Value(*item.FileSizeBytes)
		}
		summaries = append(summaries, lookupFileSummaryModel{
			FileName:      types.StringValue(item.NameAndPath.Name),
			ContentHash:   types.StringValue(item.ContentHash),
			ModifiedAt:    normalizeOptionalStringFromAPI(types.StringNull(), item.ModifiedAt),
			FileSizeBytes: fileSize,
			Resource:      types.StringValue(item.Resource),
		})
	}
	data.ID = types.StringValue(fmt.Sprintf("view=%s", data.ViewName.ValueString()))
	data.Files = summaries
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
