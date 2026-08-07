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
	_ resource.Resource                = &dashboardResource{}
	_ resource.ResourceWithConfigure   = &dashboardResource{}
	_ resource.ResourceWithImportState = &dashboardResource{}
)

func NewDashboardResource() resource.Resource {
	return &dashboardResource{}
}

type dashboardResource struct {
	client *http.Client
	config *LogScaleConfig
}

type dashboardResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ViewName         types.String `tfsdk:"view_name"`
	Name             types.String `tfsdk:"name"`
	DisplayName      types.String `tfsdk:"display_name"`
	Description      types.String `tfsdk:"description"`
	YAMLTemplate     types.String `tfsdk:"yaml_template"`
	Labels           types.Set    `tfsdk:"labels"`
	SearchDomainID   types.String `tfsdk:"search_domain_id"`
	SearchDomainName types.String `tfsdk:"search_domain_name"`
	Resource         types.String `tfsdk:"resource"`
}

func (r *dashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (r *dashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale dashboard using the YAML template CRUD path.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Dashboard ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"view_name": schema.StringAttribute{
				Description: "Repository or view that owns the dashboard.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Dashboard name used for creation. Renames currently use replacement semantics in Terraform.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Display name returned by LogScale.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Dashboard description returned by LogScale.",
				Computed:    true,
			},
			"yaml_template": schema.StringAttribute{
				Description: "Dashboard YAML template used for create/update.",
				Required:    true,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the dashboard.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"search_domain_id": schema.StringAttribute{
				Description: "Owning search domain ID.",
				Computed:    true,
			},
			"search_domain_name": schema.StringAttribute{
				Description: "Owning search domain name.",
				Computed:    true,
			},
			"resource": schema.StringAttribute{
				Description: "Provider-facing resource identifier returned by LogScale.",
				Computed:    true,
			},
		},
	}
}

func (r *dashboardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dashboardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateDashboardModel(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		CreateDashboardFromTemplateV2 dashboardListItem `json:"createDashboardFromTemplateV2"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation CreateDashboardFromTemplate($input: CreateDashboardFromTemplateV2Input!) {
			createDashboardFromTemplateV2(input: $input) {
				id
				name
				displayName
				description
				labels
				resource
				yamlTemplate
				searchDomain {
					id
					name
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"viewName":     plan.ViewName.ValueString(),
			"name":         plan.Name.ValueString(),
			"yamlTemplate": plan.YAMLTemplate.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating dashboard", err.Error())
		return
	}

	desiredLabels := stringsFromSet(ctx, plan.Labels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := syncLabelsByID(ctx, r.client, r.config.Endpoint, plan.ViewName.ValueString(), data.CreateDashboardFromTemplateV2.ID, nil, desiredLabels, "addDashboardLabels", "AddDashboardLabels", "removeDashboardLabels", "RemoveDashboardLabels"); err != nil {
		resp.Diagnostics.AddError("Error syncing dashboard labels", err.Error())
		return
	}

	readResult, found, err := lookupDashboard(ctx, r.client, r.config.Endpoint, data.CreateDashboardFromTemplateV2.ID, "")
	if err != nil {
		resp.Diagnostics.AddError("Error reading dashboard", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Dashboard disappeared after create", fmt.Sprintf("No dashboard with ID %q was found after creation.", data.CreateDashboardFromTemplateV2.ID))
		return
	}

	applyDashboardReadResultToState(ctx, &plan, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *dashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dashboardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readResult, found, err := lookupDashboard(ctx, r.client, r.config.Endpoint, state.ID.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading dashboard", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	applyDashboardReadResultToState(ctx, &state, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dashboardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state dashboardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateDashboardModel(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		UpdateDashboardFromTemplate dashboardListItem `json:"updateDashboardFromTemplate"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation UpdateDashboardFromTemplate($input: UpdateDashboardFromTemplateInput!) {
			updateDashboardFromTemplate(input: $input) {
				id
				name
				displayName
				description
				labels
				resource
				yamlTemplate
				searchDomain {
					id
					name
				}
			}
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"id":           state.ID.ValueString(),
			"viewName":     state.ViewName.ValueString(),
			"yamlTemplate": plan.YAMLTemplate.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error updating dashboard", err.Error())
		return
	}

	currentLabels := stringsFromSet(ctx, state.Labels, &resp.Diagnostics)
	desiredLabels := stringsFromSet(ctx, plan.Labels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := syncLabelsByID(ctx, r.client, r.config.Endpoint, state.ViewName.ValueString(), state.ID.ValueString(), currentLabels, desiredLabels, "addDashboardLabels", "AddDashboardLabels", "removeDashboardLabels", "RemoveDashboardLabels"); err != nil {
		resp.Diagnostics.AddError("Error syncing dashboard labels", err.Error())
		return
	}

	readResult, found, err := lookupDashboard(ctx, r.client, r.config.Endpoint, state.ID.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError("Error reading dashboard", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Dashboard disappeared after update", fmt.Sprintf("No dashboard with ID %q was found after update.", state.ID.ValueString()))
		return
	}

	plan.ViewName = state.ViewName
	plan.Name = state.Name
	applyDashboardReadResultToState(ctx, &plan, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *dashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dashboardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		DeleteDashboardV3 *bool `json:"deleteDashboardV3"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation DeleteDashboard($input: DeleteDashboard!) {
			deleteDashboardV3(input: $input)
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"id":       state.ID.ValueString(),
			"viewName": state.ViewName.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting dashboard", err.Error())
	}
}

func (r *dashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier in the format view_name:dashboard_id.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("view_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func validateDashboardModel(model dashboardResourceModel, diags *diag.Diagnostics) {
	if strings.TrimSpace(model.ViewName.ValueString()) == "" {
		diags.AddError("Invalid view_name", "view_name must not be empty.")
	}
	if strings.TrimSpace(model.Name.ValueString()) == "" {
		diags.AddError("Invalid name", "name must not be empty.")
	}
	if strings.TrimSpace(model.YAMLTemplate.ValueString()) == "" {
		diags.AddError("Invalid yaml_template", "yaml_template must not be empty.")
	}
}

func applyDashboardReadResultToState(ctx context.Context, state *dashboardResourceModel, result dashboardListItem, diags *diag.Diagnostics) {
	labels, labelDiags := normalizeOptionalStringSetFromAPI(ctx, state.Labels, result.Labels)
	diags.Append(labelDiags...)
	if diags.HasError() {
		return
	}

	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.DisplayName = types.StringValue(result.DisplayName)
	state.Description = normalizeOptionalStringFromAPI(state.Description, result.Description)
	state.YAMLTemplate = normalizeYAMLTemplateFromAPI(state.YAMLTemplate, result.YAMLTemplate)
	state.Labels = labels
	state.SearchDomainID = types.StringValue(result.SearchDomain.ID)
	state.SearchDomainName = types.StringValue(result.SearchDomain.Name)
	state.Resource = types.StringValue(result.Resource)
	if result.SearchDomain.Name != "" {
		state.ViewName = types.StringValue(result.SearchDomain.Name)
	}
}
