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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/paultyng/go-unifi/unifi"
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
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"attr_hidden_id": schema.StringAttribute{
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"attr_no_delete": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"attr_no_edit": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"ip": schema.StringAttribute{
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"fixed_ap_enabled": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"fixed_ap_mac": schema.StringAttribute{
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"hostname": schema.StringAttribute{
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"local_dns_record": schema.StringAttribute{
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"local_dns_record_enabled": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"use_fixedip": schema.BoolAttribute{
				Computed: true,
			},
			"virtual_network_override_enabled": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"virtual_network_override_id": schema.StringAttribute{
				Computed: true,
				Default:  stringdefault.StaticString(""),
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

	data.ID = types.StringValue(user.ID)
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
	user.ID = data.ID.ValueString()
	user, err := r.client.UpdateUser(ctx, r.client.site, user)
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
		DevIdOverride: int(data.DeviceIconID.ValueInt64()),
		Blocked:       data.Blocked.ValueBool(),
		FixedIP:       data.FixedIP.ValueString(),
		MAC:           data.MAC.ValueString(),
		Name:          data.Name.ValueString(),
		NetworkID:     data.NetworkID.ValueString(),
		Note:          data.Note.ValueString(),
		UseFixedIP:    data.FixedIP.ValueString() != "",
		UserGroupID:   data.UserGroupID.ValueString(),
	}

	return user
}

func setUserDefinedUserAttributes(user *unifi.User, data UserResourceModel) UserResourceModel {
	data.MAC = types.StringValue(user.MAC)
	data.Name = types.StringValue(user.Name)
	data.UserGroupID = types.StringValue(user.UserGroupID)
	data.Note = types.StringValue(user.Note)
	data.FixedIP = types.StringValue(user.FixedIP)
	data.NetworkID = types.StringValue(user.NetworkID)
	data.Blocked = types.BoolValue(user.Blocked)

	if user.DevIdOverride == 0 {
		data.DeviceIconID = types.Int64Null()
	} else {
		data.DeviceIconID = types.Int64Value(int64(user.DevIdOverride))
	}

	return data
}

func setComputedUserAttributes(user *unifi.User, data UserResourceModel) UserResourceModel {
	data.Hidden = types.BoolValue(user.Hidden)
	data.HiddenID = types.StringValue(user.HiddenID)
	data.NoDelete = types.BoolValue(user.NoDelete)
	data.NoEdit = types.BoolValue(user.NoEdit)
	data.IP = types.StringValue(user.IP)
	data.FixedApEnabled = types.BoolValue(user.FixedApEnabled)
	data.FixedApMAC = types.StringValue(user.FixedApMAC)
	data.Hostname = types.StringValue(user.Hostname)
	data.LocalDNSRecord = types.StringValue(user.LocalDNSRecord)
	data.LocalDNSRecordEnabled = types.BoolValue(user.LocalDNSRecordEnabled)
	data.UseFixedIP = types.BoolValue(user.UseFixedIP)
	data.VirtualNetworkOverrideEnabled = types.BoolValue(user.VirtualNetworkOverrideEnabled)
	data.VirtualNetworkOverrideID = types.StringValue(user.VirtualNetworkOverrideID)

	return data
}
