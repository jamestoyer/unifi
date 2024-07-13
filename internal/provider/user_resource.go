// Copyright (c) James Toyer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamestoyer/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

// UserResource defines the resource implementation.
type UserResource struct {
	client *unifiClient
}

// UserResourceModel describes the resource data model.
type UserResourceModel struct {
	MAC          types.String `tfsdk:"mac"`
	Name         types.String `tfsdk:"name"`
	UserGroupID  types.String `tfsdk:"user_group_id"`
	Note         types.String `tfsdk:"note"`
	FixedIP      types.String `tfsdk:"fixed_ip"`
	NetworkID    types.String `tfsdk:"network_id"`
	Blocked      types.Bool   `tfsdk:"blocked"`
	DeviceIconID types.Int64  `tfsdk:"device_icon_id"`

	// Computed
	ID                            types.String `tfsdk:"id"`
	Hidden                        types.Bool   `tfsdk:"attr_hidden"`
	HiddenID                      types.String `tfsdk:"attr_hidden_id"`
	NoDelete                      types.Bool   `tfsdk:"attr_no_delete"`
	NoEdit                        types.Bool   `tfsdk:"attr_no_edit"`
	IP                            types.String `tfsdk:"ip"` // non-generated field
	FixedApEnabled                types.Bool   `tfsdk:"fixed_ap_enabled"`
	FixedApMAC                    types.String `tfsdk:"fixed_ap_mac"` // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	Hostname                      types.String `tfsdk:"hostname"`
	LocalDNSRecord                types.String `tfsdk:"local_dns_record"`
	LocalDNSRecordEnabled         types.Bool   `tfsdk:"local_dns_record_enabled"`
	UseFixedIP                    types.Bool   `tfsdk:"use_fixedip"`
	VirtualNetworkOverrideEnabled types.Bool   `tfsdk:"virtual_network_override_enabled"`
	VirtualNetworkOverrideID      types.String `tfsdk:"virtual_network_override_id"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"mac": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "",
			},
			"user_group_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "",
			},
			"note": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "",
			},
			"fixed_ip": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "",
			},
			"network_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "",
			},
			"blocked": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "",
				Default:             booldefault.StaticBool(false),
			},
			"device_icon_id": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "ID of the the icon to assign to the device",
			},

			// Computed
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Example identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"attr_hidden": schema.BoolAttribute{
				Optional: true,
			},
			"attr_hidden_id": schema.StringAttribute{
				Optional: true,
			},
			"attr_no_delete": schema.BoolAttribute{
				Optional: true,
			},
			"attr_no_edit": schema.BoolAttribute{
				Optional: true,
			},
			"ip": schema.StringAttribute{
				Computed: true,
			},
			"fixed_ap_enabled": schema.BoolAttribute{
				Computed: true,
				Optional: true,
			},
			"fixed_ap_mac": schema.StringAttribute{
				Optional: true,
			},
			"hostname": schema.StringAttribute{
				Optional: true,
			},
			"local_dns_record": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"local_dns_record_enabled": schema.BoolAttribute{
				Computed: true,
				Optional: true,
			},
			"use_fixedip": schema.BoolAttribute{
				Computed: true,
			},
			"virtual_network_override_enabled": schema.BoolAttribute{
				Computed: true,
				Optional: true,
			},
			"virtual_network_override_id": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unifiClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *unifiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	user := convertUserResourceModel(data)
	user, err := r.client.CreateUser(ctx, r.client.site, user)
	if err != nil {
		resp.Diagnostics.AddError("Create user error", fmt.Sprintf("Unable to create user, got error: %s", err))
		return
	}

	data.ID = types.StringValue(*user.ID)
	data = setComputedUserAttributes(user, data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, r.client.site, data.ID.ValueString())
	if err != nil {
		var notFoundError *unifi.NotFoundError
		if errors.As(err, &notFoundError) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Read user error", fmt.Sprintf("Unable to read user, got error: %s", err))
		return
	}

	data = setComputedUserAttributes(user, data)
	data = setUserDefinedUserAttributes(user, data)

	data.FixedIP = types.StringNull()
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	user := convertUserResourceModel(data)
	user.ID = data.ID.ValueStringPointer()
	tflog.Debug(ctx, "User ID", map[string]interface{}{"user_id": user.ID})
	user, err := r.client.UpdateUser(ctx, r.client.site, user)
	if err != nil {
		var notFoundError *unifi.NotFoundError
		if errors.As(err, &notFoundError) {
			tflog.Debug(ctx, "User not found", map[string]interface{}{"response": err})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Read user error", fmt.Sprintf("Unable to read user, got error: %s", err))
		return
	}

	tflog.Debug(ctx, "Update User", map[string]interface{}{"user": user})
	data = setComputedUserAttributes(user, data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteUserByMAC(ctx, r.client.site, data.MAC.ValueString())
	if err != nil {
		var notFoundError *unifi.NotFoundError
		if errors.As(err, &notFoundError) {
			return
		}

		resp.Diagnostics.AddError("Delete user error", fmt.Sprintf("Unable to delete user, got error: %s", err))
		return
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func convertUserResourceModel(data UserResourceModel) *unifi.User {
	user := &unifi.User{
		Blocked:     data.Blocked.ValueBoolPointer(),
		FixedIP:     data.FixedIP.ValueStringPointer(),
		MAC:         data.MAC.ValueStringPointer(),
		Name:        data.Name.ValueStringPointer(),
		NetworkID:   data.NetworkID.ValueString(),
		Note:        data.Note.ValueStringPointer(),
		UseFixedIP:  data.FixedIP.ValueString() != "",
		UserGroupID: data.UserGroupID.ValueString(),
	}

	if !data.DeviceIconID.IsNull() {
		override := int(data.DeviceIconID.ValueInt64())
		user.DevIdOverride = &override
	}

	return user
}

func setUserDefinedUserAttributes(user *unifi.User, data UserResourceModel) UserResourceModel {
	data.MAC = types.StringPointerValue(user.MAC)
	data.Name = types.StringPointerValue(user.Name)
	data.UserGroupID = types.StringValue(user.UserGroupID)
	data.Note = types.StringPointerValue(user.Note)
	data.FixedIP = types.StringPointerValue(user.FixedIP)
	data.NetworkID = types.StringValue(user.NetworkID)
	data.Blocked = types.BoolPointerValue(user.Blocked)

	if user.DevIdOverride == nil {
		data.DeviceIconID = types.Int64Null()
	} else {
		data.DeviceIconID = types.Int64Value(int64(*user.DevIdOverride))
	}

	return data
}

func setComputedUserAttributes(user *unifi.User, data UserResourceModel) UserResourceModel {
	data.Hidden = types.BoolPointerValue(user.Hidden)
	data.HiddenID = types.StringPointerValue(user.HiddenID)
	data.NoDelete = types.BoolPointerValue(user.NoDelete)
	data.NoEdit = types.BoolPointerValue(user.NoEdit)
	data.IP = types.StringPointerValue(user.IP)
	data.FixedApEnabled = types.BoolValue(user.FixedApEnabled)
	data.FixedApMAC = types.StringPointerValue(user.FixedApMAC)
	data.Hostname = types.StringPointerValue(user.Hostname)
	data.LocalDNSRecord = types.StringPointerValue(user.LocalDNSRecord)
	data.LocalDNSRecordEnabled = types.BoolValue(user.LocalDNSRecordEnabled)
	data.UseFixedIP = types.BoolValue(user.UseFixedIP)
	data.VirtualNetworkOverrideEnabled = types.BoolValue(user.VirtualNetworkOverrideEnabled)
	data.VirtualNetworkOverrideID = types.StringValue(user.VirtualNetworkOverrideID)

	return data
}
