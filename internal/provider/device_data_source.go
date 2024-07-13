// Copyright (c) James Toyer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
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
	Name              types.String `tfsdk:"name"`
	PortProfileID     types.String `tfsdk:"port_profile_id"`
	OpMode            types.String `tfsdk:"op_mode"`
	POEMode           types.String `tfsdk:"poe_mode"`
	AggregateNumPorts types.Int32  `tfsdk:"aggregate_num_ports"`
}

func (d *DeviceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *DeviceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Device data source",

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
						"name": schema.StringAttribute{
							Computed: true,
						},
						"port_profile_id": schema.StringAttribute{
							Computed: true,
						},
						"op_mode": schema.StringAttribute{
							Computed: true,
						},
						"poe_mode": schema.StringAttribute{
							Computed: true,
						},
						"aggregate_num_ports": schema.Int32Attribute{
							Computed: true,
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
		data.PortOverrides[strconv.Itoa(override.PortIDX)] = DevicePortOverrideDataSourceModel{
			Name:              types.StringValue(override.Name),
			PortProfileID:     types.StringValue(override.PortProfileID),
			OpMode:            types.StringValue(override.OpMode),
			POEMode:           types.StringValue(override.PoeMode),
			AggregateNumPorts: types.Int32Value(int32(override.AggregateNumPorts)),
		}
	}

	tflog.Trace(ctx, "device read", map[string]interface{}{"mac": data.Mac.ValueString()})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
