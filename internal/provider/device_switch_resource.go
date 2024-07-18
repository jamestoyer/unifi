package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamestoyer/go-unifi/unifi"
	"github.com/jamestoyer/terraform-provider-unifi/internal/provider/customvalidator"
	"github.com/jamestoyer/terraform-provider-unifi/internal/provider/utils"
	"regexp"
)

var (
	// Ensure provider defined types fully satisfy framework interfaces.
	_ resource.Resource                = &DeviceSwitchResource{}
	_ resource.ResourceWithImportState = &DeviceSwitchResource{}

	defaultDeviceSwitchResourceModel              = DeviceSwitchResourceModel{}
	defaultDeviceSwitchConfigNetworkResourceModel = DeviceSwitchConfigNetworkResourceModel{
		BondingEnabled: types.BoolValue(false),
		Type:           types.StringValue("dhcp"),
	}
)

func NewDeviceSwitchResource() resource.Resource {
	return &DeviceSwitchResource{}
}

// DeviceSwitchResource defines the resource implementation.
type DeviceSwitchResource struct {
	client *unifiClient
}

func (r *DeviceSwitchResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_switch"
}

func (r *DeviceSwitchResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = defaultDeviceSwitchResourceModel.schema(ctx, req, resp)
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

	data = newDeviceSwitchResourceModel(device, site, data)

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

	device := data.toUnifiDevice()
	device.ID = data.ID.ValueString()

	if _, err := r.client.UpdateDevice(ctx, site, device); err != nil {
		// When there are no changes in v8 the API doesn't return the device details. This causes the client to assume
		// the device doesn't exist. To work around this for now do a read to get the status.
		// TODO: (jtoyer) Update the client to handle no changes on update, i.e. a 200 but no body
		if !errors.Is(err, &unifi.NotFoundError{}) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update switch, got error: %s", err))
			return
		}
	}

	// TODO: (jtoyer) Wait until device has finished updating after before saving the state

	// Save updated request data into Terraform state. Do not update with actual values as this will cause inconsistent
	// state errors
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

// DeviceSwitchResourceModel describes the resource data model.
type DeviceSwitchResourceModel struct {
	// Computed Values
	ID     types.String `tfsdk:"id"`
	Model  types.String `tfsdk:"model"`
	SiteID types.String `tfsdk:"site_id"`

	// Configurable Values
	Disabled            types.Bool                              `tfsdk:"disabled"`
	IPSettings          *DeviceSwitchConfigNetworkResourceModel `tfsdk:"ip_settings"`
	Mac                 types.String                            `tfsdk:"mac"`
	ManagementNetworkID types.String                            `tfsdk:"management_network_id"`
	Name                types.String                            `tfsdk:"name"`
	Site                types.String                            `tfsdk:"site"`
	SNMPContact         types.String                            `tfsdk:"snmp_contact"`
	SNMPLocation        types.String                            `tfsdk:"snmp_location"`
	STPPriority         types.String                            `tfsdk:"stp_priority"`
}

func (m *DeviceSwitchResourceModel) schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) schema.Schema {
	return schema.Schema{
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
			"site_id": schema.StringAttribute{
				MarkdownDescription: "The Unifi internal ID of the site.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Configurable values
			"disabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			// TODO: (jtoyer) To enable these we need to set an exclusion on unifi.SettingGlobalSwitch
			// "dot1x_fallback_networkconf_id": schema.StringAttribute{
			// 	Computed: true,
			// },
			// TODO: (jtoyer) To enable these we need to set an exclusion on unifi.SettingGlobalSwitch
			// "dot1x_portctrl_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			// TODO: (jtoyer) To enable these we need to set an exclusion on unifi.SettingGlobalSwitch
			// "flowctrl_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			"ip_settings": defaultDeviceSwitchConfigNetworkResourceModel.schema(ctx, req, resp),
			// TODO: (jtoyer) To enable these we need to set an exclusion on unifi.SettingGlobalSwitch
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
			// TODO: (jtoyer) To enable these we need to set an exclusion on unifi.SettingGlobalSwitch
			// "radius_profile_id": schema.StringAttribute{}
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
			"stp_priority": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("0", "4096", "8192", "12288", "16384", "20480", "24576", "28672",
						"32768", "36864", "40960", "45056", "49152", "53248", "57344", "61440"),
				},
				Default: stringdefault.StaticString("0"),
			},
			// TODO: (jtoyer) To enable these we need to set an exclusion on unifi.SettingGlobalSwitch
			// "stp_version": schema.StringAttribute{
			// 	Computed: true,
			// },
		},
	}
}

func (m *DeviceSwitchResourceModel) toUnifiDevice() *unifi.Device {
	return &unifi.Device{
		ConfigNetwork: m.IPSettings.toUnifiStruct(),
		Disabled:      m.Disabled.ValueBool(),
		MAC:           m.Mac.String(),
		MgmtNetworkID: m.ManagementNetworkID.ValueString(),
		Name:          m.Name.ValueString(),
		// TODO: (jtoyer) Populate with real values once we're there
		PortOverrides: []unifi.DevicePortOverrides{},
		SnmpContact:   m.SNMPContact.ValueString(),
		SnmpLocation:  m.SNMPLocation.ValueString(),
		StpPriority:   m.STPPriority.ValueString(),
	}
}

func newDeviceSwitchResourceModel(device *unifi.Device, site string, model DeviceSwitchResourceModel) DeviceSwitchResourceModel {
	// Computed values
	model.Model = types.StringValue(device.Model)
	model.Site = types.StringValue(site)
	model.SiteID = types.StringValue(device.SiteID)

	// Configurable Values
	model.Disabled = types.BoolValue(device.Disabled)
	model.IPSettings = newDeviceSwitchConfigNetworkResourceModel(device.ConfigNetwork, model.IPSettings)
	model.Mac = types.StringValue(device.MAC)
	model.ManagementNetworkID = utils.StringValue(device.MgmtNetworkID)
	model.Name = types.StringValue(device.Name)
	model.SNMPContact = utils.StringValue(device.SnmpContact)
	model.SNMPLocation = utils.StringValue(device.SnmpLocation)
	model.STPPriority = types.StringValue(device.StpPriority)

	return model
}

type DeviceSwitchConfigNetworkResourceModel struct {
	AlternativeDNS iptypes.IPv4Address `tfsdk:"alternative_dns"`
	BondingEnabled types.Bool          `tfsdk:"bonding_enabled"`
	DNSSuffix      types.String        `tfsdk:"dns_suffix"`
	Gateway        iptypes.IPv4Address `tfsdk:"gateway"`
	IP             iptypes.IPv4Address `tfsdk:"ip"`
	Netmask        iptypes.IPv4Address `tfsdk:"netmask"`
	PreferredDNS   iptypes.IPv4Address `tfsdk:"preferred_dns"`
	Type           types.String        `tfsdk:"type"`
}

func (m *DeviceSwitchConfigNetworkResourceModel) attributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"alternative_dns": iptypes.IPv4AddressType{},
		"bonding_enabled": types.BoolType,
		"dns_suffix":      types.StringType,
		"gateway":         iptypes.IPv4AddressType{},
		"ip":              iptypes.IPv4AddressType{},
		"netmask":         iptypes.IPv4AddressType{},
		"preferred_dns":   iptypes.IPv4AddressType{},
		"type":            types.StringType,
	}
}

func (m *DeviceSwitchConfigNetworkResourceModel) schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) schema.Attribute {
	defaultValues, diags := types.ObjectValueFrom(ctx,
		defaultDeviceSwitchConfigNetworkResourceModel.attributeTypes(),
		defaultDeviceSwitchConfigNetworkResourceModel,
	)

	resp.Diagnostics.Append(diags...)

	return schema.SingleNestedAttribute{
		Computed: true,
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"alternative_dns": schema.StringAttribute{
				Optional:   true,
				CustomType: iptypes.IPv4AddressType{},
				Validators: []validator.String{
					// Can only be set when a preferred_dns is set
					stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("preferred_dns")),
				},
			},
			"bonding_enabled": schema.BoolAttribute{
				Computed: true,
				Optional: true,
				Default:  booldefault.StaticBool(false),
			},
			"dns_suffix": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					// Can only be set when the IP is set
					stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("ip")),
				},
			},
			"gateway": schema.StringAttribute{
				Optional:   true,
				CustomType: iptypes.IPv4AddressType{},
			},
			"ip": schema.StringAttribute{
				Optional:   true,
				CustomType: iptypes.IPv4AddressType{},
			},
			"netmask": schema.StringAttribute{
				Optional:   true,
				CustomType: iptypes.IPv4AddressType{},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^((128|192|224|240|248|252|254)\\.0\\.0\\.0)|(255\\.(((0|128|192|224|240|248|252|254)\\.0\\.0)|(255\\.(((0|128|192|224|240|248|252|254)\\.0)|255\\.(0|128|192|224|240|248|252|254)))))$"), "invalid net mask"),
				},
			},
			"preferred_dns": schema.StringAttribute{
				Optional:   true,
				CustomType: iptypes.IPv4AddressType{},
			},
			"type": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Default:  stringdefault.StaticString("dhcp"),
				Validators: []validator.String{
					stringvalidator.OneOf("dhcp", "static"),
					customvalidator.StringValueWithPaths("static",
						path.MatchRelative().AtParent().AtName("gateway"),
						path.MatchRelative().AtParent().AtName("ip"),
						path.MatchRelative().AtParent().AtName("netmask"),
						path.MatchRelative().AtParent().AtName("preferred_dns"),
					),
					customvalidator.StringValueConflictsWithPaths("dhcp",
						path.MatchRelative().AtParent().AtName("alternative_dns"),
						path.MatchRelative().AtParent().AtName("bonding_enabled"),
						path.MatchRelative().AtParent().AtName("dns_suffix"),
						path.MatchRelative().AtParent().AtName("gateway"),
						path.MatchRelative().AtParent().AtName("ip"),
						path.MatchRelative().AtParent().AtName("netmask"),
						path.MatchRelative().AtParent().AtName("preferred_dns"),
					),
				},
			},
		},
		Default:    objectdefault.StaticValue(defaultValues),
		Validators: []validator.Object{},
	}
}

func (m *DeviceSwitchConfigNetworkResourceModel) toUnifiStruct() unifi.DeviceConfigNetwork {
	return unifi.DeviceConfigNetwork{
		DNS2:           m.AlternativeDNS.ValueString(),
		BondingEnabled: m.BondingEnabled.ValueBool(),
		// TODO: (jtoyer) fix DNS Suffix field name in the unifi client
		DNSsuffix: m.DNSSuffix.ValueString(),
		Gateway:   m.Gateway.ValueString(),
		IP:        m.IP.ValueString(),
		Netmask:   m.Netmask.ValueString(),
		DNS1:      m.PreferredDNS.ValueString(),
		Type:      m.Type.ValueString(),
	}
}

func newDeviceSwitchConfigNetworkResourceModel(network unifi.DeviceConfigNetwork, model *DeviceSwitchConfigNetworkResourceModel) *DeviceSwitchConfigNetworkResourceModel {
	if model == nil {
		model = &DeviceSwitchConfigNetworkResourceModel{Type: types.StringValue("dhcp")}
	}

	model.AlternativeDNS = utils.IPv4AddressValue(network.DNS2)
	model.BondingEnabled = types.BoolValue(network.BondingEnabled)
	model.DNSSuffix = utils.StringValue(network.DNSsuffix)
	model.Gateway = utils.IPv4AddressValue(network.Gateway)
	model.IP = utils.IPv4AddressValue(network.IP)
	model.Netmask = utils.IPv4AddressValue(network.Netmask)
	model.PreferredDNS = utils.IPv4AddressValue(network.DNS1)
	model.Type = utils.StringValue(network.Type)

	return model
}
