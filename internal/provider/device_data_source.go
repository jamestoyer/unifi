// Copyright (c) James Toyer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strconv"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DeviceDataSource{}

func NewDeviceDataSource() datasource.DataSource {
	return &DeviceDataSource{}
}

// DeviceDataSource defines the data source implementation.
type DeviceDataSource struct {
	client *unifiClient
}

// DeviceDataSourceModel describes the data source data model.
type DeviceDataSourceModel struct {
	Mac  types.String `tfsdk:"mac"`
	Site types.String `tfsdk:"site"`

	// Read Only
	ID            types.String                                 `tfsdk:"id"`
	Adopted       types.Bool                                   `tfsdk:"adopted"`
	Disabled      types.Bool                                   `tfsdk:"disabled"`
	Name          types.String                                 `tfsdk:"name"`
	PortOverrides map[string]DevicePortOverrideDataSourceModel `tfsdk:"port_overrides"`
	State         types.String                                 `tfsdk:"state"`
	Type          types.String                                 `tfsdk:"type"`
}

type DevicePortOverrideDataSourceModel struct {
	AggregateNumPorts        types.Int32  `tfsdk:"aggregate_num_ports"`
	ExcludedNetworkIDs       types.List   `tfsdk:"excluded_network_ids"`
	Name                     types.String `tfsdk:"name"`
	NativeNetworkID          types.String `tfsdk:"native_network_id"`
	OpMode                   types.String `tfsdk:"op_mode"`
	POEMode                  types.String `tfsdk:"poe_mode"`
	PortProfileID            types.String `tfsdk:"port_profile_id"`
	PortSecurityEnabled      types.Bool   `tfsdk:"port_security_enabled"`
	PortSecurityMACAddresses types.List   `tfsdk:"port_security_mac_addresses"`
}

func (d *DeviceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *DeviceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a Unifi device",

		Attributes: map[string]schema.Attribute{
			"mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the device",
				Required:            true,
				Validators:          []validator.String{
					// TODO: (jtoyer) Add a mac address validator
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The site of the device. When set this overrides the default provider site",
				Computed:            true,
				Optional:            true,
			},

			// Read only
			"id": schema.StringAttribute{
				MarkdownDescription: "Device identifier",
				Computed:            true,
			},
			"adopted": schema.BoolAttribute{
				Computed: true,
			},
			"disabled": schema.BoolAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"port_overrides": schema.MapNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"aggregate_num_ports": schema.Int32Attribute{
							Computed: true,
						},
						"excluded_network_ids": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"native_network_id": schema.StringAttribute{
							MarkdownDescription: "The native network used for VLAN traffic, i.e. not tagged with a " +
								"VLAN ID. Untagged traffic from devices connected to this port will be placed on to " +
								"the selected VLAN",
							Computed: true,
						},
						"op_mode": schema.StringAttribute{
							Computed: true,
						},
						"poe_mode": schema.StringAttribute{
							Computed: true,
						},
						"port_profile_id": schema.StringAttribute{
							Computed: true,
						},
						"port_security_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"port_security_mac_addresses": schema.ListAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
			"state": schema.StringAttribute{
				Computed: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *DeviceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unifiClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *unifiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *DeviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeviceDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.site
	}

	data.Site = types.StringValue(site)

	device, err := d.client.GetDeviceByMAC(ctx, site, data.Mac.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read device, got error: %s", err))
		return
	}

	data.ID = types.StringValue(device.ID)
	data.Adopted = types.BoolValue(device.Adopted)
	data.Disabled = types.BoolValue(device.Disabled)
	data.Name = types.StringValue(device.Name)
	data.State = types.StringValue(device.State.String())
	data.Type = types.StringValue(device.Type)

	data.PortOverrides = make(map[string]DevicePortOverrideDataSourceModel, len(device.PortOverrides))
	for _, override := range device.PortOverrides {
		excludedNetworkIDs := types.ListNull(types.StringType)
		if override.ExcludedNetworkIDs != nil {
			var attrs []attr.Value
			for _, id := range override.ExcludedNetworkIDs {
				attrs = append(attrs, types.StringValue(id))
			}

			excludedNetworkIDs = types.ListValueMust(types.StringType, attrs)
		}

		portSecurityMACAddresses := types.ListNull(types.StringType)
		if override.PortSecurityMACAddress != nil {
			var attrs []attr.Value
			for _, id := range override.PortSecurityMACAddress {
				attrs = append(attrs, types.StringValue(id))
			}

			portSecurityMACAddresses = types.ListValueMust(types.StringType, attrs)
		}

		data.PortOverrides[strconv.Itoa(override.PortIDX)] = DevicePortOverrideDataSourceModel{
			AggregateNumPorts:        types.Int32Value(int32(override.AggregateNumPorts)),
			ExcludedNetworkIDs:       excludedNetworkIDs,
			Name:                     types.StringValue(override.Name),
			NativeNetworkID:          types.StringValue(override.NATiveNetworkID),
			OpMode:                   types.StringValue(override.OpMode),
			POEMode:                  types.StringValue(override.PoeMode),
			PortProfileID:            types.StringValue(override.PortProfileID),
			PortSecurityEnabled:      types.BoolValue(override.PortSecurityEnabled),
			PortSecurityMACAddresses: portSecurityMACAddresses,
		}
	}

	tflog.Trace(ctx, "device read", map[string]interface{}{"mac": data.Mac.ValueString()})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
