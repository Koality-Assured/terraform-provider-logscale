package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2"
)

var (
	_ resource.Resource                = &parserResource{}
	_ resource.ResourceWithConfigure   = &parserResource{}
	_ resource.ResourceWithImportState = &parserResource{}
)

func NewParserResource() resource.Resource {
	return &parserResource{}
}

type parserResource struct {
	client *http.Client
	config *LogScaleConfig
}

type parserResourceModel struct {
	ID                             types.String              `tfsdk:"id"`
	RepositoryName                 types.String              `tfsdk:"repository_name"`
	Name                           types.String              `tfsdk:"name"`
	Script                         types.String              `tfsdk:"script"`
	DisplayName                    types.String              `tfsdk:"display_name"`
	FieldsToTag                    types.Set                 `tfsdk:"fields_to_tag"`
	FieldsToBeRemovedBeforeParsing types.Set                 `tfsdk:"fields_to_be_removed_before_parsing"`
	IsBuiltIn                      types.Bool                `tfsdk:"is_built_in"`
	YAMLTemplate                   types.String              `tfsdk:"yaml_template"`
	TestCases                      []parserTestCaseModel     `tfsdk:"test_case"`
}

type parserTestCaseModel struct {
	EventRawString   types.String                     `tfsdk:"event_raw_string"`
	OutputAssertions []parserOutputAssertionModel     `tfsdk:"output_assertion"`
}

type parserOutputAssertionModel struct {
	OutputEventIndex types.Int64                `tfsdk:"output_event_index"`
	FieldHasValues   []parserFieldHasValueModel `tfsdk:"field_has_value"`
}

type parserFieldHasValueModel struct {
	FieldName     types.String `tfsdk:"field_name"`
	ExpectedValue types.String `tfsdk:"expected_value"`
}

type parserMutationTestCase struct {
	Event            parserMutationEvent             `json:"event"`
	OutputAssertions []parserMutationOutputAssertion `json:"outputAssertions"`
}

type parserMutationEvent struct {
	RawString string `json:"rawString"`
}

type parserMutationOutputAssertion struct {
	OutputEventIndex int64                           `json:"outputEventIndex"`
	Assertions       parserMutationAssertions        `json:"assertions"`
}

type parserMutationAssertions struct {
	FieldsHaveValues []parserMutationFieldHasValue `json:"fieldsHaveValues"`
}

type parserMutationFieldHasValue struct {
	FieldName     string `json:"fieldName"`
	ExpectedValue string `json:"expectedValue"`
}

func (r *parserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parser"
}

func (r *parserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LogScale parser.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Parser ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository_name": schema.StringAttribute{
				Description: "Repository name where the parser is created",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Parser name",
				Required:    true,
			},
			"script": schema.StringAttribute{
				Description: "Parser script",
				Required:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "Full parser display name returned by LogScale",
				Computed:    true,
			},
			"fields_to_tag": schema.SetAttribute{
				Description: "Fields to tag",
				Optional:    true,
				ElementType: types.StringType,
			},
			"fields_to_be_removed_before_parsing": schema.SetAttribute{
				Description: "Fields removed before parsing",
				Optional:    true,
				ElementType: types.StringType,
			},
			"is_built_in": schema.BoolAttribute{
				Description: "Whether the parser is built in",
				Computed:    true,
			},
			"yaml_template": schema.StringAttribute{
				Description: "YAML template returned by LogScale for recreating the parser",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"test_case": schema.ListNestedBlock{
				Description: "Parser test cases to verify parser behavior.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"event_raw_string": schema.StringAttribute{
							Description: "Raw event input passed into the parser test case.",
							Required:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"output_assertion": schema.ListNestedBlock{
							Description: "Assertions for one parsed output event.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"output_event_index": schema.Int64Attribute{
										Description: "Index of the parsed output event to validate.",
										Required:    true,
									},
								},
								Blocks: map[string]schema.Block{
									"field_has_value": schema.ListNestedBlock{
										Description: "Field/value expectations for the selected parsed output event.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"field_name": schema.StringAttribute{
													Description: "Field name to validate.",
													Required:    true,
												},
												"expected_value": schema.StringAttribute{
													Description: "Expected value for the field.",
													Required:    true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *parserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.APIToken})
	r.client = oauth2.NewClient(context.Background(), src)
}

func (r *parserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan parserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	fieldsToTag := parserStringsFromSet(ctx, plan.FieldsToTag, &resp.Diagnostics)
	fieldsToRemove := parserStringsFromSet(ctx, plan.FieldsToBeRemovedBeforeParsing, &resp.Diagnostics)
	testCases := parserTestCasesToAPI(plan.TestCases)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation CreateParser($input: CreateParserInputV2!) {
			createParserV2(input: $input) {
				id
				name
				displayName
				script
				fieldsToTag
				fieldsToBeRemovedBeforeParsing
				isBuiltIn
				yamlTemplate
				testCases {
					event {
						rawString
					}
					outputAssertions {
						outputEventIndex
						assertions {
							fieldsHaveValues {
								fieldName
								expectedValue
							}
						}
					}
				}
			}
		}
	`

	input := map[string]interface{}{
		"repositoryName":                 plan.RepositoryName.ValueString(),
		"name":                           plan.Name.ValueString(),
		"script":                         plan.Script.ValueString(),
		"fieldsToTag":                    fieldsToTag,
		"fieldsToBeRemovedBeforeParsing": fieldsToRemove,
		"testCases":                      testCases,
	}

	body, err := r.doGraphQLRequest(ctx, mutationStr, map[string]interface{}{"input": input})
	if err != nil {
		resp.Diagnostics.AddError("Error creating parser", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			CreateParserV2 parserReadResult `json:"createParserV2"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError("Error creating parser", fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body))
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError("Error creating parser", fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message))
		return
	}

	r.applyParserReadResultToState(ctx, &plan, gqlResp.Data.CreateParserV2, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *parserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state parserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	queryStr := `
		query GetParser($repositoryName: String!) {
			repository(name: $repositoryName) {
				parsers {
					id
					name
					displayName
					script
					fieldsToTag
					fieldsToBeRemovedBeforeParsing
					isBuiltIn
					yamlTemplate
					testCases {
						event {
							rawString
						}
						outputAssertions {
							outputEventIndex
							assertions {
								fieldsHaveValues {
									fieldName
									expectedValue
								}
							}
						}
					}
				}
			}
		}
	`

	body, err := r.doGraphQLRequest(ctx, queryStr, map[string]interface{}{
		"repositoryName": state.RepositoryName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading parser", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			Repository *struct {
				Parsers []parserReadResult `json:"parsers"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError("Error reading parser", fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body))
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError("Error reading parser", fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message))
		return
	}

	if gqlResp.Data.Repository == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var matched *parserReadResult
	for i := range gqlResp.Data.Repository.Parsers {
		parser := &gqlResp.Data.Repository.Parsers[i]
		if parser.ID == state.ID.ValueString() {
			matched = parser
			break
		}
	}

	if matched == nil {
		for i := range gqlResp.Data.Repository.Parsers {
			parser := &gqlResp.Data.Repository.Parsers[i]
			if parser.Name == state.Name.ValueString() {
				matched = parser
				break
			}
		}
	}

	if matched == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.applyParserReadResultToState(ctx, &state, *matched, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *parserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan parserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state parserResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	fieldsToTag := parserStringsFromSet(ctx, plan.FieldsToTag, &resp.Diagnostics)
	fieldsToRemove := parserStringsFromSet(ctx, plan.FieldsToBeRemovedBeforeParsing, &resp.Diagnostics)
	testCases := parserTestCasesToAPI(plan.TestCases)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation UpdateParser($input: UpdateParserInputV2!) {
			updateParserV2(input: $input) {
				id
				name
				displayName
				script
				fieldsToTag
				fieldsToBeRemovedBeforeParsing
				isBuiltIn
				yamlTemplate
				testCases {
					event {
						rawString
					}
					outputAssertions {
						outputEventIndex
						assertions {
							fieldsHaveValues {
								fieldName
								expectedValue
							}
						}
					}
				}
			}
		}
	`

	input := map[string]interface{}{
		"repositoryName":                 state.RepositoryName.ValueString(),
		"id":                             state.ID.ValueString(),
		"name":                           plan.Name.ValueString(),
		"script":                         map[string]interface{}{"script": plan.Script.ValueString()},
		"fieldsToTag":                    fieldsToTag,
		"fieldsToBeRemovedBeforeParsing": fieldsToRemove,
		"testCases":                      testCases,
	}

	body, err := r.doGraphQLRequest(ctx, mutationStr, map[string]interface{}{"input": input})
	if err != nil {
		resp.Diagnostics.AddError("Error updating parser", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			UpdateParserV2 parserReadResult `json:"updateParserV2"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError("Error updating parser", fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body))
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError("Error updating parser", fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message))
		return
	}

	plan.RepositoryName = state.RepositoryName
	plan.ID = state.ID
	r.applyParserReadResultToState(ctx, &plan, gqlResp.Data.UpdateParserV2, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *parserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state parserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mutationStr := `
		mutation DeleteParser($input: DeleteParserInput!) {
			deleteParserV2(input: $input)
		}
	`

	body, err := r.doGraphQLRequest(ctx, mutationStr, map[string]interface{}{
		"input": map[string]interface{}{
			"repositoryName": state.RepositoryName.ValueString(),
			"id":             state.ID.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting parser", err.Error())
		return
	}

	var gqlResp struct {
		Data struct {
			DeleteParserV2 *bool `json:"deleteParserV2"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		resp.Diagnostics.AddError("Error deleting parser", fmt.Sprintf("Could not unmarshal response: %v\nBody: %s", err, body))
		return
	}

	if len(gqlResp.Errors) > 0 {
		resp.Diagnostics.AddError("Error deleting parser", fmt.Sprintf("GraphQL error: %s", gqlResp.Errors[0].Message))
		return
	}
}

func (r *parserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			"Expected import identifier in the format repository_name:parser_id.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

type parserReadResult struct {
	ID                             string                        `json:"id"`
	Name                           string                        `json:"name"`
	DisplayName                    string                        `json:"displayName"`
	Script                         string                        `json:"script"`
	FieldsToTag                    []string                      `json:"fieldsToTag"`
	FieldsToBeRemovedBeforeParsing []string                      `json:"fieldsToBeRemovedBeforeParsing"`
	IsBuiltIn                      bool                          `json:"isBuiltIn"`
	YAMLTemplate                   string                        `json:"yamlTemplate"`
	TestCases                      []parserReadTestCaseResult    `json:"testCases"`
}

type parserReadTestCaseResult struct {
	Event struct {
		RawString string `json:"rawString"`
	} `json:"event"`
	OutputAssertions []struct {
		OutputEventIndex int64 `json:"outputEventIndex"`
		Assertions       struct {
			FieldsHaveValues []struct {
				FieldName     string `json:"fieldName"`
				ExpectedValue string `json:"expectedValue"`
			} `json:"fieldsHaveValues"`
		} `json:"assertions"`
	} `json:"outputAssertions"`
}

func (r *parserResource) doGraphQLRequest(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", r.config.Endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	return body, nil
}

func parserStringsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return []string{}
	}

	var values []string
	diags.Append(set.ElementsAs(ctx, &values, false)...)
	sort.Strings(values)
	return values
}

func parserStringSetValue(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	sort.Strings(values)
	return types.SetValueFrom(ctx, types.StringType, values)
}

func parserTestCasesToAPI(testCases []parserTestCaseModel) []parserMutationTestCase {
	apiCases := make([]parserMutationTestCase, 0, len(testCases))
	for _, tc := range testCases {
		apiAssertions := make([]parserMutationOutputAssertion, 0, len(tc.OutputAssertions))
		for _, oa := range tc.OutputAssertions {
			fieldValues := make([]parserMutationFieldHasValue, 0, len(oa.FieldHasValues))
			for _, fv := range oa.FieldHasValues {
				fieldValues = append(fieldValues, parserMutationFieldHasValue{
					FieldName:     fv.FieldName.ValueString(),
					ExpectedValue: fv.ExpectedValue.ValueString(),
				})
			}

			apiAssertions = append(apiAssertions, parserMutationOutputAssertion{
				OutputEventIndex: oa.OutputEventIndex.ValueInt64(),
				Assertions: parserMutationAssertions{
					FieldsHaveValues: fieldValues,
				},
			})
		}

		apiCases = append(apiCases, parserMutationTestCase{
			Event: parserMutationEvent{
				RawString: tc.EventRawString.ValueString(),
			},
			OutputAssertions: apiAssertions,
		})
	}

	return apiCases
}

func parserTestCasesFromAPI(apiCases []parserReadTestCaseResult, current []parserTestCaseModel) []parserTestCaseModel {
	if len(apiCases) == 0 && current == nil {
		return nil
	}

	testCases := make([]parserTestCaseModel, 0, len(apiCases))
	for _, tc := range apiCases {
		var outputAssertions []parserOutputAssertionModel
		if len(tc.OutputAssertions) == 0 {
			outputAssertions = nil
		} else {
			outputAssertions = make([]parserOutputAssertionModel, 0, len(tc.OutputAssertions))
		}

		for _, oa := range tc.OutputAssertions {
			var fieldValues []parserFieldHasValueModel
			if len(oa.Assertions.FieldsHaveValues) == 0 {
				fieldValues = nil
			} else {
				fieldValues = make([]parserFieldHasValueModel, 0, len(oa.Assertions.FieldsHaveValues))
			}

			for _, fv := range oa.Assertions.FieldsHaveValues {
				fieldValues = append(fieldValues, parserFieldHasValueModel{
					FieldName:     types.StringValue(fv.FieldName),
					ExpectedValue: types.StringValue(fv.ExpectedValue),
				})
			}

			outputAssertions = append(outputAssertions, parserOutputAssertionModel{
				OutputEventIndex: types.Int64Value(oa.OutputEventIndex),
				FieldHasValues:   fieldValues,
			})
		}

		testCases = append(testCases, parserTestCaseModel{
			EventRawString:   types.StringValue(tc.Event.RawString),
			OutputAssertions: outputAssertions,
		})
	}

	return testCases
}

func (r *parserResource) applyParserReadResultToState(ctx context.Context, state *parserResourceModel, result parserReadResult, diags *diag.Diagnostics) {
	var fieldsToTag types.Set
	if len(result.FieldsToTag) == 0 && state.FieldsToTag.IsNull() {
		fieldsToTag = types.SetNull(types.StringType)
	} else {
		var setDiags diag.Diagnostics
		fieldsToTag, setDiags = parserStringSetValue(ctx, result.FieldsToTag)
		diags.Append(setDiags...)
	}

	var fieldsToRemove types.Set
	if len(result.FieldsToBeRemovedBeforeParsing) == 0 && state.FieldsToBeRemovedBeforeParsing.IsNull() {
		fieldsToRemove = types.SetNull(types.StringType)
	} else {
		var setDiags diag.Diagnostics
		fieldsToRemove, setDiags = parserStringSetValue(ctx, result.FieldsToBeRemovedBeforeParsing)
		diags.Append(setDiags...)
	}

	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Script = types.StringValue(result.Script)
	state.DisplayName = types.StringValue(result.DisplayName)
	state.FieldsToTag = fieldsToTag
	state.FieldsToBeRemovedBeforeParsing = fieldsToRemove
	state.IsBuiltIn = types.BoolValue(result.IsBuiltIn)
	state.YAMLTemplate = types.StringValue(result.YAMLTemplate)
	state.TestCases = parserTestCasesFromAPI(result.TestCases, state.TestCases)
}
