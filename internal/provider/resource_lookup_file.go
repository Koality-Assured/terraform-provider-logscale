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
	_ resource.Resource                = &lookupFileResource{}
	_ resource.ResourceWithConfigure   = &lookupFileResource{}
	_ resource.ResourceWithImportState = &lookupFileResource{}
)

func NewLookupFileResource() resource.Resource {
	return &lookupFileResource{}
}

type lookupFileResource struct {
	client *http.Client
	config *LogScaleConfig
}

type lookupFileResourceModel struct {
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

type lookupFileMetadata struct {
	NameAndPath struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"nameAndPath"`
	Labels        []string `json:"labels"`
	ContentHash   string   `json:"contentHash"`
	CreatedAt     string   `json:"createdAt"`
	CreatedBy     string   `json:"createdBy"`
	ModifiedAt    string   `json:"modifiedAt"`
	ModifiedBy    string   `json:"modifiedBy"`
	FileSizeBytes *int64   `json:"fileSizeBytes"`
	Resource      string   `json:"resource"`
}

type uploadedFileSnapshot struct {
	NameAndPath struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"nameAndPath"`
	Headers         []string   `json:"headers"`
	Lines           [][]string `json:"lines"`
	TotalLinesCount int64      `json:"totalLinesCount"`
	Resource        string     `json:"resource"`
}

type lookupFileReadResult struct {
	Metadata lookupFileMetadata
	Content  uploadedFileSnapshot
}

func (r *lookupFileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lookup_file"
}

func (r *lookupFileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale lookup file using the file upload and update GraphQL surface.",
		Attributes: map[string]schema.Attribute{
			"view_name": schema.StringAttribute{
				Description: "Repository or view that owns the lookup file.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file_name": schema.StringAttribute{
				Description: "Lookup file name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"csv_content": schema.StringAttribute{
				Description: "CSV content for the lookup file, including the header row.",
				Required:    true,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the lookup file.",
				Optional:    true,
				ElementType: types.StringType,
			},
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

func (r *lookupFileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*LogScaleConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *LogScaleConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	r.config = config
	r.client = newConfiguredHTTPClient(config)
}

func (r *lookupFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan lookupFileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	snapshot, ok := validateLookupFilePlan(plan, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}

	var created struct {
		NewFile uploadedFileSnapshot `json:"newFile"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation NewFile($viewName: String!, $fileName: String!) {
			newFile(name: $viewName, fileName: $fileName) {
				nameAndPath { name path }
				headers
				lines
				totalLinesCount
				resource
			}
		}
	`, map[string]interface{}{
		"viewName": plan.ViewName.ValueString(),
		"fileName": plan.FileName.ValueString(),
	}, &created)
	if err != nil {
		resp.Diagnostics.AddError("Error creating lookup file", err.Error())
		return
	}

	if err := r.replaceLookupFileContent(ctx, plan.ViewName.ValueString(), plan.FileName.ValueString(), nil, snapshot); err != nil {
		resp.Diagnostics.AddError("Error uploading lookup file content", err.Error())
		return
	}

	desiredLabels := stringsFromSet(ctx, plan.Labels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := syncLabelsByFileName(ctx, r.client, r.config.Endpoint, plan.ViewName.ValueString(), plan.FileName.ValueString(), nil, desiredLabels); err != nil {
		resp.Diagnostics.AddError("Error syncing lookup file labels", err.Error())
		return
	}

	readResult, found, err := r.readLookupFile(ctx, plan.ViewName.ValueString(), plan.FileName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading lookup file", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Lookup file disappeared after create", fmt.Sprintf("No lookup file named %q was found after creation.", plan.FileName.ValueString()))
		return
	}
	applyLookupFileReadResultToState(ctx, &plan, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *lookupFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state lookupFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	readResult, found, err := r.readLookupFile(ctx, state.ViewName.ValueString(), state.FileName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading lookup file", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	applyLookupFileReadResultToState(ctx, &state, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *lookupFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan lookupFileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state lookupFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	desiredSnapshot, ok := validateLookupFilePlan(plan, &resp.Diagnostics)
	if !ok || resp.Diagnostics.HasError() {
		return
	}
	currentSnapshot, err := parseCSVContent(state.CSVContent.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading current lookup file state", err.Error())
		return
	}
	if err := r.replaceLookupFileContent(ctx, state.ViewName.ValueString(), state.FileName.ValueString(), &currentSnapshot, desiredSnapshot); err != nil {
		resp.Diagnostics.AddError("Error updating lookup file", err.Error())
		return
	}
	currentLabels := stringsFromSet(ctx, state.Labels, &resp.Diagnostics)
	desiredLabels := stringsFromSet(ctx, plan.Labels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := syncLabelsByFileName(ctx, r.client, r.config.Endpoint, state.ViewName.ValueString(), state.FileName.ValueString(), currentLabels, desiredLabels); err != nil {
		resp.Diagnostics.AddError("Error syncing lookup file labels", err.Error())
		return
	}
	readResult, found, err := r.readLookupFile(ctx, state.ViewName.ValueString(), state.FileName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading lookup file", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Lookup file disappeared after update", fmt.Sprintf("No lookup file named %q was found after update.", state.FileName.ValueString()))
		return
	}
	plan.ViewName = state.ViewName
	plan.FileName = state.FileName
	applyLookupFileReadResultToState(ctx, &plan, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *lookupFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state lookupFileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var data struct {
		DeleteFile *bool `json:"deleteFile"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation DeleteFile($viewName: String!, $fileName: String!) {
			deleteFile(name: $viewName, fileName: $fileName)
		}
	`, map[string]interface{}{
		"viewName": state.ViewName.ValueString(),
		"fileName": state.FileName.ValueString(),
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting lookup file", err.Error())
	}
}

func (r *lookupFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier in the format view_name:file_name.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("view_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("file_name"), parts[1])...)
}

func validateLookupFilePlan(model lookupFileResourceModel, diags *diag.Diagnostics) (csvSnapshot, bool) {
	if strings.TrimSpace(model.ViewName.ValueString()) == "" {
		diags.AddError("Invalid view_name", "view_name must not be empty.")
	}
	if strings.TrimSpace(model.FileName.ValueString()) == "" {
		diags.AddError("Invalid file_name", "file_name must not be empty.")
	}
	if strings.TrimSpace(model.CSVContent.ValueString()) == "" {
		diags.AddError("Invalid csv_content", "csv_content must not be empty.")
	}
	if diags.HasError() {
		return csvSnapshot{}, false
	}
	snapshot, err := parseCSVContent(model.CSVContent.ValueString())
	if err != nil {
		diags.AddError("Invalid csv_content", err.Error())
		return csvSnapshot{}, false
	}
	return snapshot, true
}

func (r *lookupFileResource) replaceLookupFileContent(ctx context.Context, viewName string, fileName string, current *csvSnapshot, desired csvSnapshot) error {
	changedRows, err := csvRowsToUpdatePayload(desired.Rows)
	if err != nil {
		return err
	}
	currentHeaders := []string{}
	if current != nil {
		currentHeaders = current.Headers
	}
	var data struct {
		UpdateFile uploadedFileSnapshot `json:"updateFile"`
	}
	return executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation UpdateFile($viewName: String!, $fileName: String!, $changedRows: [String!]!, $headers: [String!]!, $columnChanges: [ColumnChange!]!) {
			updateFile(name: $viewName, fileName: $fileName, changedRows: $changedRows, headers: $headers, columnChanges: $columnChanges) {
				nameAndPath { name path }
				headers
				lines
				totalLinesCount
				resource
			}
		}
	`, map[string]interface{}{
		"viewName":      viewName,
		"fileName":      fileName,
		"changedRows":   changedRows,
		"headers":       desired.Headers,
		"columnChanges": computeColumnChanges(currentHeaders, desired.Headers),
	}, &data)
}

func (r *lookupFileResource) readLookupFile(ctx context.Context, viewName string, fileName string) (lookupFileReadResult, bool, error) {
	metadataList, err := lookupFiles(ctx, r.client, r.config.Endpoint, viewName)
	if err != nil {
		return lookupFileReadResult{}, false, err
	}
	var metadata *lookupFileMetadata
	for _, candidate := range metadataList {
		if candidate.NameAndPath.Name == fileName {
			copied := candidate
			metadata = &copied
			break
		}
	}
	if metadata == nil {
		return lookupFileReadResult{}, false, nil
	}

	var contentData struct {
		GetFileContent uploadedFileSnapshot `json:"getFileContent"`
	}
	err = executeGraphQL(ctx, r.client, r.config.Endpoint, `
		query GetFileContent($viewName: String!, $fileName: String!) {
			getFileContent(name: $viewName, fileName: $fileName) {
				nameAndPath { name path }
				headers
				lines
				totalLinesCount
				resource
			}
		}
	`, map[string]interface{}{
		"viewName": viewName,
		"fileName": fileName,
	}, &contentData)
	if err != nil {
		return lookupFileReadResult{}, false, err
	}

	return lookupFileReadResult{
		Metadata: *metadata,
		Content:  contentData.GetFileContent,
	}, true, nil
}

func applyLookupFileReadResultToState(ctx context.Context, state *lookupFileResourceModel, result lookupFileReadResult, diags *diag.Diagnostics) {
	labels, labelDiags := normalizeOptionalStringSetFromAPI(ctx, state.Labels, result.Metadata.Labels)
	diags.Append(labelDiags...)
	if diags.HasError() {
		return
	}
	csvContent, err := renderCSVContent(result.Content.Headers, result.Content.Lines)
	if err != nil {
		diags.AddError("Error rendering lookup file content", err.Error())
		return
	}

	state.FileName = types.StringValue(result.Metadata.NameAndPath.Name)
	state.Labels = labels
	state.CSVContent = types.StringValue(csvContent)
	state.ContentHash = types.StringValue(result.Metadata.ContentHash)
	state.CreatedAt = normalizeOptionalStringFromAPI(state.CreatedAt, result.Metadata.CreatedAt)
	state.CreatedBy = normalizeOptionalStringFromAPI(state.CreatedBy, result.Metadata.CreatedBy)
	state.ModifiedAt = normalizeOptionalStringFromAPI(state.ModifiedAt, result.Metadata.ModifiedAt)
	state.ModifiedBy = normalizeOptionalStringFromAPI(state.ModifiedBy, result.Metadata.ModifiedBy)
	if result.Metadata.FileSizeBytes == nil {
		state.FileSizeBytes = types.Int64Null()
	} else {
		state.FileSizeBytes = types.Int64Value(*result.Metadata.FileSizeBytes)
	}
	state.TotalLinesCount = types.Int64Value(result.Content.TotalLinesCount)
	state.Resource = types.StringValue(result.Metadata.Resource)
}

func lookupFiles(ctx context.Context, client *http.Client, endpoint string, viewName string) ([]lookupFileMetadata, error) {
	var data struct {
		SearchDomain *struct {
			TypeName string               `json:"__typename"`
			Files    []lookupFileMetadata `json:"files"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, client, endpoint, `
		query GetLookupFiles($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository {
					files {
						nameAndPath { name path }
						labels
						contentHash
						createdAt
						createdBy
						modifiedAt
						modifiedBy
						fileSizeBytes
						resource
					}
				}
				... on View {
					files {
						nameAndPath { name path }
						labels
						contentHash
						createdAt
						createdBy
						modifiedAt
						modifiedBy
						fileSizeBytes
						resource
					}
				}
			}
		}
	`, map[string]interface{}{"viewName": viewName}, &data)
	if err != nil {
		return nil, err
	}
	if data.SearchDomain == nil {
		return nil, nil
	}
	return data.SearchDomain.Files, nil
}
