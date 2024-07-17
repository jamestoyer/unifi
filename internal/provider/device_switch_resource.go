package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamestoyer/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DeviceSwitchResource{}
var _ resource.ResourceWithImportState = &DeviceSwitchResource{}

func NewDeviceSwitchResource() resource.Resource {
	return &DeviceSwitchResource{}
}

// DeviceSwitchResource defines the resource implementation.
type DeviceSwitchResource struct {
	client *unifiClient
}

// DeviceSwitchResourceModel describes the resource data model.
type DeviceSwitchResourceModel struct {
	// Computed Values
	ID    types.String `tfsdk:"id"`
	Model types.String `tfsdk:"model"`

	// Configurable Values
	Disabled            types.Bool   `tfsdk:"disabled"`
	Mac                 types.String `tfsdk:"mac"`
	ManagementNetworkID types.String `tfsdk:"management_network_id"`
	Name                types.String `tfsdk:"name"`
	Site                types.String `tfsdk:"site"`
	SnmpContact         types.String `tfsdk:"snmp_contact"`
	SnmpLocation        types.String `tfsdk:"snmp_location"`
}

func (r *DeviceSwitchResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_switch"
}

func (r *DeviceSwitchResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Unifi switch device.",

		Attributes: map[string]schema.Attribute{
			// Computed values
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Unifi switch device identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"model": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Configurable values

			// "config_network": schema.SingleNestedAttribute{
			// 	Computed: true,
			// 	Attributes: map[string]schema.Attribute{
			// 		"alternative_dns": schema.StringAttribute{
			// 			Computed: true,
			// 		},
			// 		"bonding_enabled": schema.BoolAttribute{
			// 			Computed: true,
			// 		},
			// 		"dns_suffix": schema.StringAttribute{
			// 			Computed: true,
			// 		},
			// 		"gateway": schema.StringAttribute{
			// 			Computed: true,
			// 		},
			// 		"ip": schema.StringAttribute{
			// 			Computed: true,
			// 		},
			// 		"netmask": schema.StringAttribute{
			// 			Computed: true,
			// 		},
			// 		"preferred_dns": schema.StringAttribute{
			// 			Computed: true,
			// 		},
			// 		"type": schema.StringAttribute{
			// 			Computed: true,
			// 		},
			// 	},
			// },
			"disabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			// "dot1x_fallback_networkconf_id": schema.StringAttribute{
			// 	Computed: true,
			// },
			// "dot1x_portctrl_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			// "flowctrl_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			// "jumboframe_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			"mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the device",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					// TODO: (jtoyer) Add a mac address validator
				},
			},
			"management_network_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the VLAN to use as the management VLAN instead of the default tagged " +
					"network from the upstream device.",
				Optional: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "A name to assign to the device",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(128),
				},
			},
			// "port_overrides": schema.MapNestedAttribute{
			// 	Computed: true,
			// 	NestedObject: schema.NestedAttributeObject{
			// 		Attributes: map[string]schema.Attribute{
			// 			"aggregate_num_ports": schema.Int32Attribute{
			// 				Optional: true,
			// 				Validators: []validator.Int32{
			// 					int32validator.Between(1, 8),
			// 				},
			// 			},
			// 			"auto_negotiate": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"dot1x_ctrl": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"dot1x_idle_timeout": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"egress_rate_limit_kbps": schema.Int32Attribute{
			// 				MarkdownDescription: "Sets a port's maximum rate of data transfer.",
			// 				Computed:            true,
			// 			},
			// 			"egress_rate_limit_kbps_enabled": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"excluded_network_ids": schema.ListAttribute{
			// 				ElementType: types.StringType,
			// 				Optional:    true,
			// 			},
			// 			"fec_mode": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"forward": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"full_duplex": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"isolation": schema.BoolAttribute{
			// 				MarkdownDescription: "Allows you to prohibit traffic between isolated ports. This only " +
			// 					"applies to ports on the same device.",
			// 				Computed: true,
			// 			},
			// 			"lldp_med_enabled": schema.BoolAttribute{
			// 				MarkdownDescription: "Extension for LLPD user alongside the voice VLAN feature to " +
			// 					"discover the presence of a VoIP phone. Disabling LLPD-MED will also disable the " +
			// 					"Voice VLAN.",
			// 				Computed: true,
			// 			},
			// 			"lldp_med_notify_enabled": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"mirror_port_idx": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"name": schema.StringAttribute{
			// 				Required: true,
			// 				Validators: []validator.String{
			// 					stringvalidator.LengthBetween(0, 128),
			// 				},
			// 			},
			// 			"native_network_id": schema.StringAttribute{
			// 				MarkdownDescription: "The native network used for VLAN traffic, i.e. not tagged with a " +
			// 					"VLAN ID. Untagged traffic from devices connected to this port will be placed on to " +
			// 					"the selected VLAN",
			// 				Optional: true,
			// 			},
			// 			"operation": schema.StringAttribute{
			// 				Required: true,
			// 				Validators: []validator.String{
			// 					stringvalidator.OneOf("switch", "mirror", "aggregate"),
			// 				},
			// 			},
			// 			"poe_mode": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"port_keepalive_enabled": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"port_profile_id": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"port_security_enabled": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"port_security_mac_addresses": schema.ListAttribute{
			// 				ElementType: types.StringType,
			// 				Computed:    true,
			// 			},
			// 			"priority_queue1_level": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"priority_queue2_level": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"priority_queue3_level": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"priority_queue4_level": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"qos_profile": schema.SingleNestedAttribute{
			// 				Computed: true,
			// 				Attributes: map[string]schema.Attribute{
			// 					"qos_policies": schema.SetNestedAttribute{
			// 						Computed: true,
			// 						NestedObject: schema.NestedAttributeObject{
			//
			// 							Attributes: map[string]schema.Attribute{
			// 								"qos_marking": schema.SingleNestedAttribute{
			// 									Computed: true,
			// 									Attributes: map[string]schema.Attribute{
			// 										"cos_code": schema.Int32Attribute{
			// 											Computed: true,
			// 										},
			// 										"dscp_code": schema.Int32Attribute{
			// 											Computed: true,
			// 										},
			// 										"ip_precedence_code": schema.Int32Attribute{
			// 											Computed: true,
			// 										},
			// 										"queue": schema.Int32Attribute{
			// 											Computed: true,
			// 										},
			// 									},
			// 								},
			// 								"qos_matching": schema.SingleNestedAttribute{
			// 									Computed: true,
			// 									Attributes: map[string]schema.Attribute{
			// 										"cos_code": schema.Int32Attribute{
			// 											Computed: true,
			// 										},
			// 										"dscp_code": schema.Int32Attribute{
			// 											Computed: true,
			// 										},
			// 										"dst_port": schema.Int32Attribute{
			// 											Computed: true,
			// 										},
			// 										"ip_precedence_code": schema.Int32Attribute{
			// 											Computed: true,
			// 										},
			// 										"protocol": schema.StringAttribute{
			// 											Computed: true,
			// 										},
			// 										"src_port": schema.Int32Attribute{
			// 											Computed: true,
			// 										},
			// 									},
			// 								},
			// 							},
			// 						},
			// 					},
			// 					"qos_profile_mode": schema.StringAttribute{
			// 						Computed: true,
			// 					},
			// 				},
			// 			},
			// 			"setting_preference": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"speed": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_broadcast_enabled": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_broadcast_level": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_broadcast_rate": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_multicast_enabled": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_multicast_level": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_mulitcast_rate": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_type": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_unicast_enabled": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_unicast_level": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"storm_control_unicast_rate": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"stp_port_mode": schema.BoolAttribute{
			// 				Computed: true,
			// 			},
			// 			"tagged_vlan_mgmt": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"voice_networkconf_id": schema.StringAttribute{
			// 				MarkdownDescription: "Uses LLPD-MED to place a VoIP phone on the specified VLAN. Devices " +
			// 					"connected to the phone are placed in the Native VLAN.",
			// 				Computed: true,
			// 			},
			// 		},
			// 	},
			// },
			"site": schema.StringAttribute{
				MarkdownDescription: "The site the switch belongs to. Setting this overrides the default site set in " +
					"the provider",
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"snmp_contact": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"snmp_location": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			// "stp_priority": schema.StringAttribute{
			// 	Computed: true,
			// },
			// "stp_version": schema.StringAttribute{
			// 	Computed: true,
			// },

			// "configurable_attribute": schema.StringAttribute{
			// 	MarkdownDescription: "Example configurable attribute",
			// 	Optional:            true,
			// },
			// "defaulted": schema.StringAttribute{
			// 	MarkdownDescription: "Example configurable attribute with default value",
			// 	Optional:            true,
			// 	Computed:            true,
			// 	Default:             stringdefault.StaticString("example value when not configured"),
			// },
		},
	}
}

func (r *DeviceSwitchResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeviceSwitchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeviceSwitchResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
	//     return
	// }

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.ID = types.StringValue("example-id")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceSwitchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeviceSwitchResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	site := r.client.site
	if data.Site.ValueString() != "" {
		site = data.Site.ValueString()
	}

	device, err := r.client.GetDevice(ctx, site, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read switch, got error: %s", err))
		return
	}

	// Computed values
	data.Model = types.StringValue(device.Model)

	// Configurable Values
	data.Disabled = types.BoolValue(device.Disabled)
	data.Mac = types.StringValue(device.MAC)
	if device.MgmtNetworkID != "" {
		data.ManagementNetworkID = types.StringValue(device.MgmtNetworkID)
	}
	data.Name = types.StringValue(device.Name)
	if device.SnmpContact != "" {
		data.SnmpContact = types.StringValue(device.SnmpContact)
	}
	if device.SnmpLocation != "" {
		data.SnmpLocation = types.StringValue(device.SnmpLocation)
	}

	data.Site = types.StringValue(site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceSwitchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeviceSwitchResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	site := r.client.site
	if data.Site.ValueString() != "" {
		site = data.Site.ValueString()
	}

	device := &unifi.Device{
		ID: data.ID.ValueString(),

		Disabled:      data.Disabled.ValueBool(),
		MgmtNetworkID: data.ManagementNetworkID.ValueString(),
		Name:          data.Name.ValueString(),
		// TODO: (jtoyer) Populate with real values once we're there
		PortOverrides: []unifi.DevicePortOverrides{},
		SnmpContact:   data.SnmpContact.ValueString(),
		SnmpLocation:  data.SnmpLocation.ValueString(),
	}

	device, err := r.client.UpdateDevice(ctx, site, device)
	if err != nil {
		// When there are no changes in v8 the API doesn't return the device details. This causes the client to assume
		// the device doesn't exist. To work around this for now do a read to get the status.
		// TODO: (jtoyer) Update the client to handle no changes on update, i.e. a 200 but no body
		if !errors.Is(err, &unifi.NotFoundError{}) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update switch, got error: %s", err))
			return
		}

		device, err = r.client.GetDevice(ctx, site, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read updated switch, got error: %s", err))
			return
		}
	}

	// TODO: (jtoyer) Wait until device has finished updating after before setting these values
	// Computed values
	data.Model = types.StringValue(device.Model)
	//
	// Configurable Values
	data.Disabled = types.BoolValue(device.Disabled)
	data.Mac = types.StringValue(device.MAC)
	if device.MgmtNetworkID != "" {
		data.ManagementNetworkID = types.StringValue(device.MgmtNetworkID)
	}
	data.Name = types.StringValue(device.Name)
	if device.SnmpContact != "" {
		data.SnmpContact = types.StringValue(device.SnmpContact)
	}
	if device.SnmpLocation != "" {
		data.SnmpLocation = types.StringValue(device.SnmpLocation)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceSwitchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeviceSwitchResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
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

func (r *DeviceSwitchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
