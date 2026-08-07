package provider

import (
	"context"
	"net/http"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type assetReference struct {
	ID   string
	Name string
}

func normalizeOptionalBoolFromAPI(current types.Bool, value bool) types.Bool {
	if !value && current.IsNull() {
		return types.BoolNull()
	}

	return types.BoolValue(value)
}

func sortedUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	sort.Strings(out)
	return out
}

func diffStringSets(current []string, desired []string) ([]string, []string) {
	current = sortedUniqueStrings(current)
	desired = sortedUniqueStrings(desired)

	currentSet := make(map[string]struct{}, len(current))
	for _, value := range current {
		currentSet[value] = struct{}{}
	}

	desiredSet := make(map[string]struct{}, len(desired))
	for _, value := range desired {
		desiredSet[value] = struct{}{}
	}

	toAdd := make([]string, 0)
	for _, value := range desired {
		if _, ok := currentSet[value]; !ok {
			toAdd = append(toAdd, value)
		}
	}

	toRemove := make([]string, 0)
	for _, value := range current {
		if _, ok := desiredSet[value]; !ok {
			toRemove = append(toRemove, value)
		}
	}

	return toAdd, toRemove
}

func stringsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}

	var values []string
	diags.Append(set.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return nil
	}

	return sortedUniqueStrings(values)
}

func stringSetValue(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	return types.SetValueFrom(ctx, types.StringType, sortedUniqueStrings(values))
}

func normalizeOptionalStringSetFromAPI(ctx context.Context, current types.Set, values []string) (types.Set, diag.Diagnostics) {
	if len(values) == 0 && current.IsNull() {
		return types.SetNull(types.StringType), nil
	}

	return stringSetValue(ctx, values)
}

func preserveCurrentReferencesIfResolvable(ctx context.Context, current types.Set, remote []assetReference) (types.Set, diag.Diagnostics) {
	currentDiags := diag.Diagnostics{}
	currentRefs := stringsFromSet(ctx, current, &currentDiags)
	if len(currentRefs) > 0 && len(currentRefs) == len(remote) {
		remaining := make(map[string]int, len(remote)*2)
		for _, ref := range remote {
			if ref.ID != "" {
				remaining[ref.ID]++
			}
			if ref.Name != "" {
				remaining[ref.Name]++
			}
		}

		matchedAll := true
		for _, currentRef := range currentRefs {
			if remaining[currentRef] == 0 {
				matchedAll = false
				break
			}
			remaining[currentRef]--
		}

		if matchedAll {
			return stringSetValue(ctx, currentRefs)
		}
	}

	remoteNames := make([]string, 0, len(remote))
	for _, ref := range remote {
		if ref.Name != "" {
			remoteNames = append(remoteNames, ref.Name)
		}
	}
	if len(remoteNames) == len(remote) {
		return stringSetValue(ctx, remoteNames)
	}

	remoteIDs := make([]string, 0, len(remote))
	for _, ref := range remote {
		if ref.ID != "" {
			remoteIDs = append(remoteIDs, ref.ID)
		}
	}

	return stringSetValue(ctx, remoteIDs)
}

func syncLabelsByID(ctx context.Context, client *http.Client, endpoint string, viewName string, id string, current []string, desired []string, addMutation string, addInputType string, removeMutation string, removeInputType string) error {
	toAdd, toRemove := diffStringSets(current, desired)

	if len(toAdd) > 0 {
		var addResult map[string]interface{}
		addQuery := "mutation AddLabels($input: " + addInputType + "!) { " + addMutation + "(input: $input) { id } }"
		if err := executeGraphQL(ctx, client, endpoint, addQuery, map[string]interface{}{
			"input": map[string]interface{}{
				"id":       id,
				"viewName": viewName,
				"labels":   toAdd,
			},
		}, &addResult); err != nil {
			return err
		}
	}

	if len(toRemove) > 0 {
		var removeResult map[string]interface{}
		removeQuery := "mutation RemoveLabels($input: " + removeInputType + "!) { " + removeMutation + "(input: $input) { id } }"
		if err := executeGraphQL(ctx, client, endpoint, removeQuery, map[string]interface{}{
			"input": map[string]interface{}{
				"id":       id,
				"viewName": viewName,
				"labels":   toRemove,
			},
		}, &removeResult); err != nil {
			return err
		}
	}

	return nil
}

func syncLabelsByFileName(ctx context.Context, client *http.Client, endpoint string, viewName string, fileName string, current []string, desired []string) error {
	toAdd, toRemove := diffStringSets(current, desired)

	if len(toAdd) > 0 {
		var addResult map[string]interface{}
		if err := executeGraphQL(ctx, client, endpoint, `
			mutation AddFileLabels($input: AddFileLabels!) {
				addFileLabels(input: $input) {
					nameAndPath {
						name
					}
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"viewName": viewName,
				"fileName": fileName,
				"labels":   toAdd,
			},
		}, &addResult); err != nil {
			return err
		}
	}

	if len(toRemove) > 0 {
		var removeResult map[string]interface{}
		if err := executeGraphQL(ctx, client, endpoint, `
			mutation RemoveFileLabels($input: RemoveFileLabels!) {
				removeFileLabels(input: $input) {
					nameAndPath {
						name
					}
				}
			}
		`, map[string]interface{}{
			"input": map[string]interface{}{
				"viewName": viewName,
				"fileName": fileName,
				"labels":   toRemove,
			},
		}, &removeResult); err != nil {
			return err
		}
	}

	return nil
}
