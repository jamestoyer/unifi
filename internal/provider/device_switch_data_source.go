// Copyright (c) James Toyer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamestoyer/terraform-provider-unifi/internal/provider/utils"
	"strconv"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DeviceSwitchDataSource{}

func NewDeviceSwitchDataSource() datasource.DataSource {
	return &DeviceSwitchDataSource{}
}

// DeviceSwitchDataSource defines the data source implementation.
type DeviceSwitchDataSource struct {
	client *unifiClient
}

// DeviceSwitchDataSourceModel describes the data source data model.
type DeviceSwitchDataSourceModel struct {
	Mac  types.String `tfsdk:"mac"`
	Site types.String `tfsdk:"site"`

	// Read Only
	ID                         types.String                                       `tfsdk:"id"`
	Adopted                    types.Bool                                         `tfsdk:"adopted"`
	ConfigNetwork              *DeviceSwitchConfigNetworkDataSourceModel          `tfsdk:"config_network"`
	Disabled                   types.Bool                                         `tfsdk:"disabled"`
	Dot1XFallbackNetworkID     types.String                                       `tfsdk:"dot1x_fallback_networkconf_id"`
	Dot1XPortctrlEnabled       types.Bool                                         `tfsdk:"dot1x_portctrl_enabled"`
	FlowctrlEnabled            types.Bool                                         `tfsdk:"flowctrl_enabled"`
	IP                         types.String                                       `tfsdk:"ip"`
	JumboframeEnabled          types.Bool                                         `tfsdk:"jumboframe_enabled"`
	LEDOverride                types.String                                       `tfsdk:"led_override"`
	LEDOverrideColor           types.String                                       `tfsdk:"led_override_color"`
	LEDOverrideColorBrightness types.Int32                                        `tfsdk:"led_override_color_brightness"`
	MgmtNetworkID              types.String                                       `tfsdk:"mgmt_network_id"`
	Model                      types.String                                       `tfsdk:"model"`
	Name                       types.String                                       `tfsdk:"name"`
	PortOverrides              map[string]DeviceSwitchPortOverrideDataSourceModel `tfsdk:"port_overrides"`
	SnmpContact                types.String                                       `tfsdk:"snmp_contact"`
	SnmpLocation               types.String                                       `tfsdk:"snmp_location"`
	State                      types.String                                       `tfsdk:"state"`
	StpPriority                types.String                                       `tfsdk:"stp_priority"`
	StpVersion                 types.String                                       `tfsdk:"stp_version"`
	Type                       types.String                                       `tfsdk:"type"`
}

type DeviceSwitchConfigNetworkDataSourceModel struct {
	AlternativeDNS iptypes.IPv4Address `tfsdk:"alternative_dns"`
	BondingEnabled types.Bool          `tfsdk:"bonding_enabled"`
	DNSSuffix      types.String        `tfsdk:"dns_suffix"`
	Gateway        iptypes.IPv4Address `tfsdk:"gateway"`
	IP             iptypes.IPv4Address `tfsdk:"ip"`
	Netmask        iptypes.IPv4Address `tfsdk:"netmask"`
	PreferredDNS   iptypes.IPv4Address `tfsdk:"preferred_dns"`
	Type           types.String        `tfsdk:"type"`
}

type DeviceSwitchPortOverrideDataSourceModel struct {
	AggregateNumPorts            types.Int32                                        `tfsdk:"aggregate_num_ports"`
	AutoNegotiate                types.Bool                                         `tfsdk:"auto_negotiate"`
	Dot1XCtrl                    types.String                                       `tfsdk:"dot1x_ctrl"`
	Dot1XIDleTimeout             types.Int32                                        `tfsdk:"dot1x_idle_timeout"`
	EgressRateLimitKbps          types.Int32                                        `tfsdk:"egress_rate_limit_kbps"`
	EgressRateLimitKbpsEnabled   types.Bool                                         `tfsdk:"egress_rate_limit_kbps_enabled"`
	ExcludedNetworkIDs           types.List                                         `tfsdk:"excluded_network_ids"`
	FECMode                      types.String                                       `tfsdk:"fec_mode"`
	Forward                      types.String                                       `tfsdk:"forward"`
	FullDuplex                   types.Bool                                         `tfsdk:"full_duplex"`
	Isolation                    types.Bool                                         `tfsdk:"isolation"`
	LLPMEDEnabled                types.Bool                                         `tfsdk:"lldp_med_enabled"`
	LLPMEDNotifyEnabled          types.Bool                                         `tfsdk:"lldp_med_notify_enabled"`
	MirrorPortIDX                types.Int32                                        `tfsdk:"mirror_port_idx"`
	Name                         types.String                                       `tfsdk:"name"`
	NativeNetworkID              types.String                                       `tfsdk:"native_network_id"`
	Operation                    types.String                                       `tfsdk:"operation"`
	PriorityQueue1Level          types.Int32                                        `tfsdk:"priority_queue1_level"`
	PriorityQueue2Level          types.Int32                                        `tfsdk:"priority_queue2_level"`
	PriorityQueue3Level          types.Int32                                        `tfsdk:"priority_queue3_level"`
	PriorityQueue4Level          types.Int32                                        `tfsdk:"priority_queue4_level"`
	POEMode                      types.String                                       `tfsdk:"poe_mode"`
	PortKeepaliveEnabled         types.Bool                                         `tfsdk:"port_keepalive_enabled"`
	PortProfileID                types.String                                       `tfsdk:"port_profile_id"`
	PortSecurityEnabled          types.Bool                                         `tfsdk:"port_security_enabled"`
	PortSecurityMACAddresses     types.List                                         `tfsdk:"port_security_mac_addresses"`
	QOSProfile                   *DeviceSwitchPortOverrideQOSProfileDataSourceModel `tfsdk:"qos_profile"`
	SettingPreference            types.String                                       `tfsdk:"setting_preference"`
	Speed                        types.Int32                                        `tfsdk:"speed"`
	StormControlBroadcastEnabled types.Bool                                         `tfsdk:"storm_control_broadcast_enabled"`
	StormControlBroadcastLevel   types.Int32                                        `tfsdk:"storm_control_broadcast_level"`
	StormControlBroadcastRate    types.Int32                                        `tfsdk:"storm_control_broadcast_rate"`
	StormControlMulticastEnabled types.Bool                                         `tfsdk:"storm_control_multicast_enabled"`
	StormControlMulticastLevel   types.Int32                                        `tfsdk:"storm_control_multicast_level"`
	StormControlMulticastRate    types.Int32                                        `tfsdk:"storm_control_mulitcast_rate"`
	StormControlType             types.String                                       `tfsdk:"storm_control_type"`
	StormControlUnicastEnabled   types.Bool                                         `tfsdk:"storm_control_unicast_enabled"`
	StormControlUnicastLevel     types.Int32                                        `tfsdk:"storm_control_unicast_level"`
	StormControlUnicastRate      types.Int32                                        `tfsdk:"storm_control_unicast_rate"`
	STPPortMode                  types.Bool                                         `tfsdk:"stp_port_mode"`
	TaggedVLANMgmt               types.String                                       `tfsdk:"tagged_vlan_mgmt"`
	VoiceNetworkID               types.String                                       `tfsdk:"voice_networkconf_id"`
}

type DeviceSwitchPortOverrideQOSProfileDataSourceModel struct {
	QOSPolicies    []DeviceSwitchPortOverrideQOSPolicyDataSourceModel `tfsdk:"qos_policies"`
	QOSProfileMode types.String                                       `tfsdk:"qos_profile_mode"`
}

type DeviceSwitchPortOverrideQOSPolicyDataSourceModel struct {
	QOSMarking  DeviceSwitchPortOverrideQOSMarkingDataSourceModel  `tfsdk:"qos_marking"`
	QOSMatching DeviceSwitchPortOverrideQOSMatchingDataSourceModel `tfsdk:"qos_matching"`
}

type DeviceSwitchPortOverrideQOSMarkingDataSourceModel struct {
	CosCode          types.Int32 `tfsdk:"cos_code"`
	DscpCode         types.Int32 `tfsdk:"dscp_code"`
	IPPrecedenceCode types.Int32 `tfsdk:"ip_precedence_code"`
	Queue            types.Int32 `tfsdk:"queue"`
}

type DeviceSwitchPortOverrideQOSMatchingDataSourceModel struct {
	CosCode          types.Int32  `tfsdk:"cos_code"`
	DscpCode         types.Int32  `tfsdk:"dscp_code"`
	DstPort          types.Int32  `tfsdk:"dst_port"`
	IPPrecedenceCode types.Int32  `tfsdk:"ip_precedence_code"`
	Protocol         types.String `tfsdk:"protocol"`
	SrcPort          types.Int32  `tfsdk:"src_port"`
}

func (d *DeviceSwitchDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_switch"
}

func (d *DeviceSwitchDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a Unifi switch device",

		Attributes: map[string]schema.Attribute{
			"mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the device",
				Required:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The site the switch belongs to. Setting this overrides the default site set in " +
					"the provider",
				Computed: true,
				Optional: true,
			},

			// Read only
			"id": schema.StringAttribute{
				MarkdownDescription: "The Unifi device identifier",
				Computed:            true,
			},
			"adopted": schema.BoolAttribute{
				Computed: true,
			},
			"config_network": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"alternative_dns": schema.StringAttribute{
						Computed: true,
					},
					"bonding_enabled": schema.BoolAttribute{
						Computed: true,
					},
					"dns_suffix": schema.StringAttribute{
						Computed: true,
					},
					"gateway": schema.StringAttribute{
						Computed: true,
					},
					"ip": schema.StringAttribute{
						Computed: true,
					},
					"netmask": schema.StringAttribute{
						Computed: true,
					},
					"preferred_dns": schema.StringAttribute{
						Computed: true,
					},
					"type": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"disabled": schema.BoolAttribute{
				Computed: true,
			},
			"dot1x_fallback_networkconf_id": schema.StringAttribute{
				Computed: true,
			},
			"dot1x_portctrl_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"flowctrl_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "The currently assigned IP address of the device",
				Computed:            true,
			},
			"jumboframe_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"led_override": schema.StringAttribute{
				Computed: true,
			},
			"led_override_color": schema.StringAttribute{
				Computed: true,
			},
			"led_override_color_brightness": schema.Int32Attribute{
				Computed: true,
			},
			"mgmt_network_id": schema.StringAttribute{
				Computed: true,
			},
			"model": schema.StringAttribute{
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
							Optional: true,
							Validators: []validator.Int32{
								int32validator.Between(1, 8),
							},
						},
						"auto_negotiate": schema.BoolAttribute{
							Computed: true,
						},
						"dot1x_ctrl": schema.StringAttribute{
							Computed: true,
						},
						"dot1x_idle_timeout": schema.Int32Attribute{
							Computed: true,
						},
						"egress_rate_limit_kbps": schema.Int32Attribute{
							MarkdownDescription: "Sets a port's maximum rate of data transfer.",
							Computed:            true,
						},
						"egress_rate_limit_kbps_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"excluded_network_ids": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
						},
						"fec_mode": schema.StringAttribute{
							Computed: true,
						},
						"forward": schema.StringAttribute{
							Computed: true,
						},
						"full_duplex": schema.BoolAttribute{
							Computed: true,
						},
						"isolation": schema.BoolAttribute{
							MarkdownDescription: "Allows you to prohibit traffic between isolated ports. This only " +
								"applies to ports on the same device.",
							Computed: true,
						},
						"lldp_med_enabled": schema.BoolAttribute{
							MarkdownDescription: "Extension for LLPD user alongside the voice VLAN feature to " +
								"discover the presence of a VoIP phone. Disabling LLPD-MED will also disable the " +
								"Voice VLAN.",
							Computed: true,
						},
						"lldp_med_notify_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"mirror_port_idx": schema.Int32Attribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(0, 128),
							},
						},
						"native_network_id": schema.StringAttribute{
							MarkdownDescription: "The native network used for VLAN traffic, i.e. not tagged with a " +
								"VLAN ID. Untagged traffic from devices connected to this port will be placed on to " +
								"the selected VLAN",
							Optional: true,
						},
						"operation": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("switch", "mirror", "aggregate"),
							},
						},
						"poe_mode": schema.StringAttribute{
							Computed: true,
						},
						"port_keepalive_enabled": schema.BoolAttribute{
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
						"priority_queue1_level": schema.Int32Attribute{
							Computed: true,
						},
						"priority_queue2_level": schema.Int32Attribute{
							Computed: true,
						},
						"priority_queue3_level": schema.Int32Attribute{
							Computed: true,
						},
						"priority_queue4_level": schema.Int32Attribute{
							Computed: true,
						},
						"qos_profile": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"qos_policies": schema.SetNestedAttribute{
									Computed: true,
									NestedObject: schema.NestedAttributeObject{

										Attributes: map[string]schema.Attribute{
											"qos_marking": schema.SingleNestedAttribute{
												Computed: true,
												Attributes: map[string]schema.Attribute{
													"cos_code": schema.Int32Attribute{
														Computed: true,
													},
													"dscp_code": schema.Int32Attribute{
														Computed: true,
													},
													"ip_precedence_code": schema.Int32Attribute{
														Computed: true,
													},
													"queue": schema.Int32Attribute{
														Computed: true,
													},
												},
											},
											"qos_matching": schema.SingleNestedAttribute{
												Computed: true,
												Attributes: map[string]schema.Attribute{
													"cos_code": schema.Int32Attribute{
														Computed: true,
													},
													"dscp_code": schema.Int32Attribute{
														Computed: true,
													},
													"dst_port": schema.Int32Attribute{
														Computed: true,
													},
													"ip_precedence_code": schema.Int32Attribute{
														Computed: true,
													},
													"protocol": schema.StringAttribute{
														Computed: true,
													},
													"src_port": schema.Int32Attribute{
														Computed: true,
													},
												},
											},
										},
									},
								},
								"qos_profile_mode": schema.StringAttribute{
									Computed: true,
								},
							},
						},
						"setting_preference": schema.StringAttribute{
							Computed: true,
						},
						"speed": schema.Int32Attribute{
							Computed: true,
						},
						"storm_control_broadcast_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"storm_control_broadcast_level": schema.Int32Attribute{
							Computed: true,
						},
						"storm_control_broadcast_rate": schema.Int32Attribute{
							Computed: true,
						},
						"storm_control_multicast_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"storm_control_multicast_level": schema.Int32Attribute{
							Computed: true,
						},
						"storm_control_mulitcast_rate": schema.Int32Attribute{
							Computed: true,
						},
						"storm_control_type": schema.StringAttribute{
							Computed: true,
						},
						"storm_control_unicast_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"storm_control_unicast_level": schema.Int32Attribute{
							Computed: true,
						},
						"storm_control_unicast_rate": schema.Int32Attribute{
							Computed: true,
						},
						"stp_port_mode": schema.BoolAttribute{
							Computed: true,
						},
						"tagged_vlan_mgmt": schema.StringAttribute{
							Computed: true,
						},
						"voice_networkconf_id": schema.StringAttribute{
							MarkdownDescription: "Uses LLPD-MED to place a VoIP phone on the specified VLAN. Devices " +
								"connected to the phone are placed in the Native VLAN.",
							Computed: true,
						},
					},
				},
			},
			"snmp_contact": schema.StringAttribute{
				Computed: true,
			},
			"snmp_location": schema.StringAttribute{
				Optional: true,
			},
			"state": schema.StringAttribute{
				Computed: true,
			},
			"stp_priority": schema.StringAttribute{
				Computed: true,
			},
			"stp_version": schema.StringAttribute{
				Computed: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *DeviceSwitchDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DeviceSwitchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeviceSwitchDataSourceModel

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

	tflog.Info(ctx, "Site ID", map[string]interface{}{"site ID": device.SiteID})

	data.ID = types.StringPointerValue(device.ID)
	data.Adopted = types.BoolValue(device.Adopted)
	data.Disabled = types.BoolPointerValue(device.Disabled)
	data.Dot1XFallbackNetworkID = types.StringPointerValue(device.Dot1XFallbackNetworkID)
	data.Dot1XPortctrlEnabled = types.BoolPointerValue(device.Dot1XPortctrlEnabled)
	data.FlowctrlEnabled = types.BoolPointerValue(device.FlowctrlEnabled)
	data.IP = types.StringPointerValue(device.IP)
	data.JumboframeEnabled = types.BoolPointerValue(device.JumboframeEnabled)
	data.LEDOverrideColorBrightness = types.Int32PointerValue(utils.Int32PtrValue(device.LedOverrideColorBrightness))
	data.LEDOverride = types.StringPointerValue(device.LedOverride)
	data.LEDOverrideColor = types.StringPointerValue(device.LedOverrideColor)
	data.MgmtNetworkID = types.StringPointerValue(device.MgmtNetworkID)
	data.Model = types.StringPointerValue(device.Model)
	data.Name = types.StringPointerValue(device.Name)
	data.SnmpContact = types.StringPointerValue(device.SnmpContact)
	data.SnmpLocation = types.StringPointerValue(device.SnmpLocation)
	data.State = types.StringValue(device.State.String())
	data.StpPriority = types.StringPointerValue(device.StpPriority)
	data.StpVersion = types.StringPointerValue(device.StpVersion)
	data.Type = types.StringPointerValue(device.Type)

	data.ConfigNetwork = &DeviceSwitchConfigNetworkDataSourceModel{
		BondingEnabled: types.BoolPointerValue(device.ConfigNetwork.BondingEnabled),
		Type:           types.StringPointerValue(device.ConfigNetwork.Type),
	}

	// if device.ConfigNetwork.DNS1 != "" {
	data.ConfigNetwork.PreferredDNS = iptypes.NewIPv4AddressPointerValue(device.ConfigNetwork.DNS1)
	// }
	// if device.ConfigNetwork.DNS2 != "" {
	data.ConfigNetwork.AlternativeDNS = iptypes.NewIPv4AddressPointerValue(device.ConfigNetwork.DNS2)
	// }
	// if device.ConfigNetwork.DNSsuffix != "" {
	data.ConfigNetwork.DNSSuffix = types.StringPointerValue(device.ConfigNetwork.DNSsuffix)
	// }
	// if device.ConfigNetwork.Gateway != "" {
	data.ConfigNetwork.Gateway = iptypes.NewIPv4AddressPointerValue(device.ConfigNetwork.Gateway)
	// }
	// if device.ConfigNetwork.IP != "" {
	data.ConfigNetwork.IP = iptypes.NewIPv4AddressPointerValue(device.ConfigNetwork.IP)
	// }
	// if device.ConfigNetwork.Netmask != "" {
	data.ConfigNetwork.Netmask = iptypes.NewIPv4AddressPointerValue(device.ConfigNetwork.Netmask)
	// }

	data.PortOverrides = make(map[string]DeviceSwitchPortOverrideDataSourceModel, len(device.PortOverrides))
	for _, override := range device.PortOverrides {
		excludedNetworkIDs := types.ListNull(types.StringType)
		if override.ExcludedNetworkIDs != nil {
			var attrs []attr.Value
			for _, id := range *override.ExcludedNetworkIDs {
				attrs = append(attrs, types.StringValue(id))
			}

			excludedNetworkIDs = types.ListValueMust(types.StringType, attrs)
		}

		portSecurityMACAddresses := types.ListNull(types.StringType)
		if override.PortSecurityMACAddress != nil {
			var attrs []attr.Value
			for _, id := range *override.PortSecurityMACAddress {
				attrs = append(attrs, types.StringValue(id))
			}

			portSecurityMACAddresses = types.ListValueMust(types.StringType, attrs)
		}

		var qosProfile *DeviceSwitchPortOverrideQOSProfileDataSourceModel
		if override.QOSProfile != nil {
			qosProfile = &DeviceSwitchPortOverrideQOSProfileDataSourceModel{
				QOSProfileMode: types.StringPointerValue(override.QOSProfile.QOSProfileMode),
			}

			if override.QOSProfile.QOSPolicies != nil {
				for _, policy := range *override.QOSProfile.QOSPolicies {
					qosProfile.QOSPolicies = append(qosProfile.QOSPolicies, DeviceSwitchPortOverrideQOSPolicyDataSourceModel{
						QOSMarking: DeviceSwitchPortOverrideQOSMarkingDataSourceModel{
							CosCode:          types.Int32PointerValue(utils.Int32PtrValue(policy.QOSMarking.CosCode)),
							DscpCode:         types.Int32PointerValue(utils.Int32PtrValue(policy.QOSMarking.DscpCode)),
							IPPrecedenceCode: types.Int32PointerValue(utils.Int32PtrValue(policy.QOSMarking.IPPrecedenceCode)),
							Queue:            types.Int32PointerValue(utils.Int32PtrValue(policy.QOSMarking.Queue)),
						},
						QOSMatching: DeviceSwitchPortOverrideQOSMatchingDataSourceModel{
							CosCode:          types.Int32PointerValue(utils.Int32PtrValue(policy.QOSMatching.CosCode)),
							DscpCode:         types.Int32PointerValue(utils.Int32PtrValue(policy.QOSMatching.DscpCode)),
							DstPort:          types.Int32PointerValue(utils.Int32PtrValue(policy.QOSMatching.DstPort)),
							IPPrecedenceCode: types.Int32PointerValue(utils.Int32PtrValue(policy.QOSMatching.IPPrecedenceCode)),
							Protocol:         types.StringPointerValue(policy.QOSMatching.Protocol),
							SrcPort:          types.Int32PointerValue(utils.Int32PtrValue(policy.QOSMatching.SrcPort)),
						},
					})
				}
			}
		}

		// TODO: (jtoyer) In the client make port index value not a pointer as it must always be set
		data.PortOverrides[strconv.Itoa(*override.PortIDX)] = DeviceSwitchPortOverrideDataSourceModel{
			AggregateNumPorts:            types.Int32PointerValue(utils.Int32PtrValue(override.AggregateNumPorts)),
			AutoNegotiate:                types.BoolPointerValue(override.Autoneg),
			Dot1XCtrl:                    types.StringPointerValue(override.Dot1XCtrl),
			Dot1XIDleTimeout:             types.Int32PointerValue(utils.Int32PtrValue(override.Dot1XIDleTimeout)),
			EgressRateLimitKbps:          types.Int32PointerValue(utils.Int32PtrValue(override.EgressRateLimitKbps)),
			EgressRateLimitKbpsEnabled:   types.BoolPointerValue(override.EgressRateLimitKbpsEnabled),
			ExcludedNetworkIDs:           excludedNetworkIDs,
			FECMode:                      types.StringPointerValue(override.FecMode),
			Forward:                      types.StringPointerValue(override.Forward),
			FullDuplex:                   types.BoolPointerValue(override.FullDuplex),
			Isolation:                    types.BoolPointerValue(override.Isolation),
			LLPMEDEnabled:                types.BoolPointerValue(override.LldpmedEnabled),
			LLPMEDNotifyEnabled:          types.BoolPointerValue(override.LldpmedNotifyEnabled),
			MirrorPortIDX:                types.Int32PointerValue(utils.Int32PtrValue(override.MirrorPortIDX)),
			Name:                         types.StringPointerValue(override.Name),
			NativeNetworkID:              types.StringPointerValue(override.NATiveNetworkID),
			Operation:                    types.StringPointerValue(override.OpMode),
			POEMode:                      types.StringPointerValue(override.PoeMode),
			PortKeepaliveEnabled:         types.BoolPointerValue(override.PortKeepaliveEnabled),
			PortProfileID:                types.StringPointerValue(override.PortProfileID),
			PortSecurityEnabled:          types.BoolPointerValue(override.PortSecurityEnabled),
			PortSecurityMACAddresses:     portSecurityMACAddresses,
			PriorityQueue1Level:          types.Int32PointerValue(utils.Int32PtrValue(override.PriorityQueue1Level)),
			PriorityQueue2Level:          types.Int32PointerValue(utils.Int32PtrValue(override.PriorityQueue2Level)),
			PriorityQueue3Level:          types.Int32PointerValue(utils.Int32PtrValue(override.PriorityQueue3Level)),
			PriorityQueue4Level:          types.Int32PointerValue(utils.Int32PtrValue(override.PriorityQueue4Level)),
			QOSProfile:                   qosProfile,
			SettingPreference:            types.StringPointerValue(override.SettingPreference),
			Speed:                        types.Int32PointerValue(utils.Int32PtrValue(override.Speed)),
			StormControlBroadcastEnabled: types.BoolPointerValue(override.StormctrlBroadcastastEnabled),
			StormControlBroadcastLevel:   types.Int32PointerValue(utils.Int32PtrValue(override.StormctrlBroadcastastLevel)),
			StormControlBroadcastRate:    types.Int32PointerValue(utils.Int32PtrValue(override.StormctrlBroadcastastRate)),
			StormControlMulticastEnabled: types.BoolPointerValue(override.StormctrlMcastEnabled),
			StormControlMulticastLevel:   types.Int32PointerValue(utils.Int32PtrValue(override.StormctrlUcastLevel)),
			StormControlMulticastRate:    types.Int32PointerValue(utils.Int32PtrValue(override.StormctrlUcastRate)),
			StormControlType:             types.StringPointerValue(override.StormctrlType),
			StormControlUnicastEnabled:   types.BoolPointerValue(override.StormctrlUcastEnabled),
			StormControlUnicastLevel:     types.Int32PointerValue(utils.Int32PtrValue(override.StormctrlUcastLevel)),
			StormControlUnicastRate:      types.Int32PointerValue(utils.Int32PtrValue(override.StormctrlUcastRate)),
			STPPortMode:                  types.BoolPointerValue(override.StpPortMode),
			TaggedVLANMgmt:               types.StringPointerValue(override.TaggedVLANMgmt),
			VoiceNetworkID:               types.StringPointerValue(override.VoiceNetworkID),
		}
	}

	tflog.Trace(ctx, "device read", map[string]interface{}{"mac": data.Mac.ValueString()})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
