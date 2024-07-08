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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	Id           types.String `tfsdk:"id"`
	MAC          types.String `tfsdk:"mac"`
	Name         types.String `tfsdk:"name"`
	UserGroupID  types.String `tfsdk:"user_group_id"`
	Note         types.String `tfsdk:"note"`
	FixedIP      types.String `tfsdk:"fixed_ip"`
	NetworkID    types.String `tfsdk:"network_id"`
	Blocked      types.Bool   `tfsdk:"blocked"`
	DeviceIconID types.Int64  `tfsdk:"device_icon_id"`
	SiteID       types.String `tfsdk:"site_id"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Example identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
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
			"site_id": schema.StringAttribute{
				Computed: true,
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

	user := &unifi.User{
		Hidden:                        false,
		HiddenID:                      "",
		NoDelete:                      false,
		NoEdit:                        false,
		DevIdOverride:                 int(data.DeviceIconID.ValueInt64()),
		IP:                            "",
		Blocked:                       data.Blocked.ValueBool(),
		FixedApEnabled:                false,
		FixedApMAC:                    "",
		FixedIP:                       data.FixedIP.ValueString(),
		Hostname:                      "",
		LastSeen:                      0,
		LocalDNSRecord:                "",
		LocalDNSRecordEnabled:         false,
		MAC:                           data.MAC.ValueString(),
		Name:                          data.Name.ValueString(),
		NetworkID:                     data.NetworkID.ValueString(),
		Note:                          data.Note.ValueString(),
		UseFixedIP:                    data.FixedIP.ValueString() != "",
		UserGroupID:                   data.UserGroupID.ValueString(),
		VirtualNetworkOverrideEnabled: false,
		VirtualNetworkOverrideID:      "",
	}

	user, err := r.client.CreateUser(ctx, r.client.site, user)
	if err != nil {
		resp.Diagnostics.AddError("Create user error", fmt.Sprintf("Unable to create user, got error: %s", err))
		return
	}

	data.Id = types.StringValue(user.ID)
	data.SiteID = types.StringValue(user.SiteID)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

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

	user, err := r.client.GetUser(ctx, r.client.site, data.Id.ValueString())
	if err != nil {
		var notFoundError *unifi.NotFoundError
		if errors.As(err, &notFoundError) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Read user error", fmt.Sprintf("Unable to read user, got error: %s", err))
		return
	}

	data.Blocked = basetypes.NewBoolValue(user.Blocked)
	data.MAC = types.StringValue(user.MAC)

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

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

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

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
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Delete user error", fmt.Sprintf("Unable to delete user, got error: %s", err))
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
