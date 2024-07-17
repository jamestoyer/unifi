package utils

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringValue returns a types.StringNull if the string is empty; else the value.
// This exists as the client today omitempty for string values. Ideally it would support pointers to indicate that a
// value is explicitly unset or just send the empty value as a default.
func StringValue(v string) types.String {
	if v == "" {
		return types.StringNull()
	}

	return types.StringValue(v)
}
