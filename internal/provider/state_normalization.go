package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func normalizeOptionalStringFromAPI(current types.String, value string) types.String {
	if value == "" && current.IsNull() {
		return types.StringNull()
	}

	return types.StringValue(value)
}

func normalizeYAMLTemplateFromAPI(current types.String, value string) types.String {
	if !current.IsNull() && normalizeNewlines(current.ValueString()) == normalizeNewlines(value) {
		return current
	}

	return types.StringValue(value)
}

func normalizeNewlines(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}
