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
	_ resource.Resource                = &webhookActionResource{}
	_ resource.ResourceWithConfigure   = &webhookActionResource{}
	_ resource.ResourceWithImportState = &webhookActionResource{}
)

func NewWebhookActionResource() resource.Resource {
	return &webhookActionResource{}
}

type webhookActionResource struct {
	client *http.Client
	config *LogScaleConfig
}

type webhookActionResourceModel struct {
	ID                                      types.String               `tfsdk:"id"`
	ViewName                                types.String               `tfsdk:"view_name"`
	Name                                    types.String               `tfsdk:"name"`
	URL                                     types.String               `tfsdk:"url"`
	Method                                  types.String               `tfsdk:"method"`
	IgnoreSSL                               types.Bool                 `tfsdk:"ignore_ssl"`
	UseProxy                                types.Bool                 `tfsdk:"use_proxy"`
	BodyTemplate                            types.String               `tfsdk:"body_template"`
	Labels                                  types.Set                  `tfsdk:"labels"`
	Resource                                types.String               `tfsdk:"resource"`
	YAMLTemplate                            types.String               `tfsdk:"yaml_template"`
	RequiresOrgOwnedQueriesPermissionToEdit types.Bool                 `tfsdk:"requires_org_owned_queries_permission_to_edit"`
	Headers                                 []webhookActionHeaderModel `tfsdk:"header"`
}

type webhookActionHeaderModel struct {
	Header types.String `tfsdk:"header"`
	Value  types.String `tfsdk:"value"`
}

type webhookActionReadResult struct {
	TypeName                                string                   `json:"__typename"`
	ID                                      string                   `json:"id"`
	Name                                    string                   `json:"name"`
	URL                                     string                   `json:"url"`
	Method                                  string                   `json:"method"`
	IgnoreSSL                               bool                     `json:"ignoreSSL"`
	UseProxy                                bool                     `json:"useProxy"`
	BodyTemplate                            string                   `json:"bodyTemplate"`
	Labels                                  []string                 `json:"labels"`
	Resource                                string                   `json:"resource"`
	YAMLTemplate                            string                   `json:"yamlTemplate"`
	RequiresOrgOwnedQueriesPermissionToEdit bool                     `json:"requiresOrgOwnedQueriesPermissionToEdit"`
	Headers                                 []webhookActionHeaderAPI `json:"headers"`
}

type webhookActionHeaderAPI struct {
	Header string `json:"header"`
	Value  string `json:"value"`
}

func (r *webhookActionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_action"
}

func (r *webhookActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale webhook action.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Webhook action ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"view_name": schema.StringAttribute{
				Description: "Repository or view that owns the action.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Action name.",
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: "Webhook target URL.",
				Required:    true,
			},
			"method": schema.StringAttribute{
				Description: "HTTP method used by the action.",
				Required:    true,
			},
			"ignore_ssl": schema.BoolAttribute{
				Description: "Whether to ignore SSL validation errors.",
				Required:    true,
			},
			"use_proxy": schema.BoolAttribute{
				Description: "Whether to use the configured outbound proxy.",
				Required:    true,
			},
			"body_template": schema.StringAttribute{
				Description: "Webhook request body template.",
				Required:    true,
			},
			"labels": schema.SetAttribute{
				Description: "Labels attached to the action.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"resource": schema.StringAttribute{
				Description: "Provider-facing resource identifier returned by LogScale.",
				Computed:    true,
			},
			"yaml_template": schema.StringAttribute{
				Description: "YAML template returned by LogScale.",
				Computed:    true,
			},
			"requires_org_owned_queries_permission_to_edit": schema.BoolAttribute{
				Description: "Whether editing this action requires organization-owned query permissions.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"header": schema.ListNestedBlock{
				Description: "HTTP headers attached to the webhook request.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"header": schema.StringAttribute{
							Description: "Header name.",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Description: "Header value.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *webhookActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *webhookActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"viewName":     plan.ViewName.ValueString(),
		"name":         plan.Name.ValueString(),
		"url":          plan.URL.ValueString(),
		"method":       plan.Method.ValueString(),
		"ignoreSSL":    plan.IgnoreSSL.ValueBool(),
		"useProxy":     plan.UseProxy.ValueBool(),
		"bodyTemplate": plan.BodyTemplate.ValueString(),
		"headers":      webhookHeadersToAPI(plan.Headers),
	}

	var data struct {
		CreateWebhookAction webhookActionReadResult `json:"createWebhookAction"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation CreateWebhookAction($input: CreateWebhookAction!) {
			createWebhookAction(input: $input) {
				__typename
				id
				name
				labels
				resource
				... on WebhookAction {
					url
					method
					ignoreSSL
					useProxy
					bodyTemplate
					yamlTemplate
					requiresOrgOwnedQueriesPermissionToEdit
					headers {
						header
						value
					}
				}
			}
		}
	`, map[string]interface{}{"input": input}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook action", err.Error())
		return
	}

	desiredLabels := stringsFromSet(ctx, plan.Labels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := syncLabelsByID(ctx, r.client, r.config.Endpoint, plan.ViewName.ValueString(), data.CreateWebhookAction.ID, nil, desiredLabels, "addActionLabels", "AddActionLabels", "removeActionLabels", "RemoveActionLabels"); err != nil {
		resp.Diagnostics.AddError("Error syncing webhook action labels", err.Error())
		return
	}

	readResult, found, err := r.readWebhookAction(ctx, plan.ViewName.ValueString(), data.CreateWebhookAction.ID, "")
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook action", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Webhook action disappeared after create", fmt.Sprintf("No webhook action with ID %q was found after creation.", data.CreateWebhookAction.ID))
		return
	}

	r.applyWebhookActionReadResultToState(ctx, &plan, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *webhookActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readResult, found, err := r.readWebhookAction(ctx, state.ViewName.ValueString(), state.ID.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook action", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	r.applyWebhookActionReadResultToState(ctx, &state, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state webhookActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]interface{}{
		"viewName":     state.ViewName.ValueString(),
		"id":           state.ID.ValueString(),
		"name":         plan.Name.ValueString(),
		"url":          plan.URL.ValueString(),
		"method":       plan.Method.ValueString(),
		"ignoreSSL":    plan.IgnoreSSL.ValueBool(),
		"useProxy":     plan.UseProxy.ValueBool(),
		"bodyTemplate": plan.BodyTemplate.ValueString(),
		"headers":      webhookHeadersToAPI(plan.Headers),
	}

	var data struct {
		UpdateWebhookAction webhookActionReadResult `json:"updateWebhookAction"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation UpdateWebhookAction($input: UpdateWebhookAction!) {
			updateWebhookAction(input: $input) {
				__typename
				id
				name
				labels
				resource
				... on WebhookAction {
					url
					method
					ignoreSSL
					useProxy
					bodyTemplate
					yamlTemplate
					requiresOrgOwnedQueriesPermissionToEdit
					headers {
						header
						value
					}
				}
			}
		}
	`, map[string]interface{}{"input": input}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook action", err.Error())
		return
	}

	currentLabels := stringsFromSet(ctx, state.Labels, &resp.Diagnostics)
	desiredLabels := stringsFromSet(ctx, plan.Labels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := syncLabelsByID(ctx, r.client, r.config.Endpoint, state.ViewName.ValueString(), state.ID.ValueString(), currentLabels, desiredLabels, "addActionLabels", "AddActionLabels", "removeActionLabels", "RemoveActionLabels"); err != nil {
		resp.Diagnostics.AddError("Error syncing webhook action labels", err.Error())
		return
	}

	readResult, found, err := r.readWebhookAction(ctx, state.ViewName.ValueString(), state.ID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook action", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Webhook action disappeared after update", fmt.Sprintf("No webhook action with ID %q was found after update.", state.ID.ValueString()))
		return
	}

	plan.ViewName = state.ViewName
	r.applyWebhookActionReadResultToState(ctx, &plan, readResult, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *webhookActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data struct {
		DeleteActionV2 *bool `json:"deleteActionV2"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		mutation DeleteWebhookAction($input: DeleteActionV2!) {
			deleteActionV2(input: $input)
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"viewName": state.ViewName.ValueString(),
			"id":       state.ID.ValueString(),
		},
	}, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting webhook action", err.Error())
	}
}

func (r *webhookActionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Unexpected Import Identifier", "Expected import identifier in the format view_name:action_id.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("view_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *webhookActionResource) readWebhookAction(ctx context.Context, viewName string, id string, name string) (webhookActionReadResult, bool, error) {
	var data struct {
		SearchDomain *struct {
			TypeName string                    `json:"__typename"`
			Actions  []webhookActionReadResult `json:"actions"`
		} `json:"searchDomain"`
	}
	err := executeGraphQL(ctx, r.client, r.config.Endpoint, `
		query GetWebhookActions($viewName: String!) {
			searchDomain(name: $viewName) {
				__typename
				... on Repository {
					actions {
						__typename
						id
						name
						labels
						resource
						... on WebhookAction {
							url
							method
							ignoreSSL
							useProxy
							bodyTemplate
							yamlTemplate
							requiresOrgOwnedQueriesPermissionToEdit
							headers {
								header
								value
							}
						}
					}
				}
				... on View {
					actions {
						__typename
						id
						name
						labels
						resource
						... on WebhookAction {
							url
							method
							ignoreSSL
							useProxy
							bodyTemplate
							yamlTemplate
							requiresOrgOwnedQueriesPermissionToEdit
							headers {
								header
								value
							}
						}
					}
				}
			}
		}
	`, map[string]interface{}{"viewName": viewName}, &data)
	if err != nil {
		return webhookActionReadResult{}, false, err
	}

	if data.SearchDomain == nil {
		return webhookActionReadResult{}, false, nil
	}

	return selectWebhookAction(data.SearchDomain.Actions, id, name)
}

func selectWebhookAction(actions []webhookActionReadResult, id string, name string) (webhookActionReadResult, bool, error) {
	if id != "" {
		for _, action := range actions {
			if action.TypeName == "WebhookAction" && action.ID == id {
				return action, true, nil
			}
		}
	}

	if name != "" {
		for _, action := range actions {
			if action.TypeName == "WebhookAction" && action.Name == name {
				return action, true, nil
			}
		}
	}

	return webhookActionReadResult{}, false, nil
}

func webhookHeadersToAPI(headers []webhookActionHeaderModel) []map[string]string {
	apiHeaders := make([]map[string]string, 0, len(headers))
	for _, header := range headers {
		apiHeaders = append(apiHeaders, map[string]string{
			"header": header.Header.ValueString(),
			"value":  header.Value.ValueString(),
		})
	}

	return apiHeaders
}

func webhookHeadersFromAPI(headers []webhookActionHeaderAPI) []webhookActionHeaderModel {
	out := make([]webhookActionHeaderModel, 0, len(headers))
	for _, header := range headers {
		out = append(out, webhookActionHeaderModel{
			Header: types.StringValue(header.Header),
			Value:  types.StringValue(header.Value),
		})
	}

	return out
}

func (r *webhookActionResource) applyWebhookActionReadResultToState(ctx context.Context, state *webhookActionResourceModel, result webhookActionReadResult, diags *diag.Diagnostics) {
	labels, labelDiags := normalizeOptionalStringSetFromAPI(ctx, state.Labels, result.Labels)
	diags.Append(labelDiags...)
	if diags.HasError() {
		return
	}

	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.URL = types.StringValue(result.URL)
	state.Method = types.StringValue(result.Method)
	state.IgnoreSSL = types.BoolValue(result.IgnoreSSL)
	state.UseProxy = types.BoolValue(result.UseProxy)
	state.BodyTemplate = types.StringValue(result.BodyTemplate)
	state.Labels = labels
	state.Resource = types.StringValue(result.Resource)
	state.YAMLTemplate = types.StringValue(result.YAMLTemplate)
	state.RequiresOrgOwnedQueriesPermissionToEdit = types.BoolValue(result.RequiresOrgOwnedQueriesPermissionToEdit)
	state.Headers = webhookHeadersFromAPI(result.Headers)
}
