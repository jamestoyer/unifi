package customtype

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"net"
	"strings"
)

var (
	_ basetypes.StringValuable       = (*Mac)(nil)
	_ xattr.ValidateableAttribute    = (*Mac)(nil)
	_ function.ValidateableParameter = (*Mac)(nil)
)

// Mac represents a valid mac address.
type Mac struct {
	basetypes.StringValue
}

// Type returns an MacType.
func (v Mac) Type(_ context.Context) attr.Type {
	return MacType{}
}

// Equal returns true if the given value is equivalent.
func (v Mac) Equal(o attr.Value) bool {
	other, ok := o.(Mac)

	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

// ValidateAttribute implements attribute value validation. This type requires the value provided to be a String
// value that is a valid mac address.
func (v Mac) ValidateAttribute(ctx context.Context, req xattr.ValidateAttributeRequest, resp *xattr.ValidateAttributeResponse) {
	if v.IsUnknown() || v.IsNull() {
		return
	}

	s := v.ValueString()
	sanitised := strings.ReplaceAll(strings.ToLower(v.StringValue.ValueString()), "-", ":")
	if sanitised != s {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Mac String Value",
			"A string value that container upper case characters or dashes was provided.\n\n"+
				"Given Value: "+s+"\n"+
				"Expected Value: "+sanitised+"\n",
		)
	}

	if _, err := net.ParseMAC(v.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Mac String Value",
			"A string value was provided that is not valid Mac string format .\n\n"+
				"Given Value: "+v.ValueString()+"\n"+
				"Error: "+err.Error(),
		)

		return
	}

}

// ValidateParameter implements provider-defined function parameter value validation. This type requires the value
// provided to be a String value that is a valid mac address.
func (v Mac) ValidateParameter(ctx context.Context, req function.ValidateParameterRequest, resp *function.ValidateParameterResponse) {
	if v.IsUnknown() || v.IsNull() {
		return
	}

	s := v.ValueString()
	sanitised := strings.ReplaceAll(strings.ToLower(v.StringValue.ValueString()), "-", ":")
	if sanitised != s {
		resp.Error = function.NewArgumentFuncError(
			req.Position,
			"Invalid Mac String Value: "+
				"A string value that container upper case characters or dashes was provided.\n\n"+
				"Given Value: "+s+"\n"+
				"Expected Value: "+sanitised+"\n",
		)
	}

	if _, err := net.ParseMAC(v.ValueString()); err != nil {
		resp.Error = function.NewArgumentFuncError(
			req.Position,
			"Invalid Mac String Value: "+
				"A string value was provided that is not valid mac string format.\n\n"+
				"Given Value: "+v.ValueString()+"\n"+
				"Error: "+err.Error(),
		)

		return
	}
}

// ValueMac calls net.ParseMAC with the Mac StringValue. A null or unknown value will produce an error diagnostic.
func (v Mac) ValueMac() (net.HardwareAddr, diag.Diagnostics) {
	var diags diag.Diagnostics

	if v.IsNull() {
		diags.Append(diag.NewErrorDiagnostic("Mac ValueMac Error", "IPv4 CIDR string value is null"))
		return net.HardwareAddr{}, diags
	}

	if v.IsUnknown() {
		diags.Append(diag.NewErrorDiagnostic("Mac ValueMac Error", "IPv4 CIDR string value is unknown"))
		return net.HardwareAddr{}, diags
	}

	mac, err := net.ParseMAC(v.ValueString())
	if err != nil {
		diags.Append(diag.NewErrorDiagnostic("Mac ValueMac Error", err.Error()))
		return net.HardwareAddr{}, diags
	}

	return mac, nil
}

// NewMacNull creates an Mac with a null value. Determine whether the value is null via IsNull method.
func NewMacNull() Mac {
	return Mac{
		StringValue: basetypes.NewStringNull(),
	}
}

// NewMacUnknown creates an Mac with an unknown value. Determine whether the value is unknown via IsUnknown method.
func NewMacUnknown() Mac {
	return Mac{
		StringValue: basetypes.NewStringUnknown(),
	}
}

// NewMacValue creates an Mac with a known value. Access the value via ValueString method.
func NewMacValue(value string) Mac {
	return Mac{
		StringValue: basetypes.NewStringValue(value),
	}
}

// NewMacPointerValue creates an Mac with a null value if nil or a known value. Access the value via ValueStringPointer
// method.
func NewMacPointerValue(value *string) Mac {
	return Mac{
		StringValue: basetypes.NewStringPointerValue(value),
	}
}
