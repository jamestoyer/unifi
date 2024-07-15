// Copyright (c) James Toyer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"regexp"
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
	ID      types.String `tfsdk:"id"`
	Adopted types.Bool   `tfsdk:"adopted"`
	// ConfigNetwork          DeviceSwitchConfigNetworkDataSourceModel `tfsdk:"config_network"`
	Dot1XFallbackNetworkID types.String `tfsdk:"dot1x_fallback_networkconf_id"`
	Dot1XPortctrlEnabled   types.Bool   `tfsdk:"dot1x_portctrl_enabled"`
	FlowctrlEnabled        types.Bool   `tfsdk:"flowctrl_enabled"`
	JumboframeEnabled      types.Bool   `tfsdk:"jumboframe_enabled"`
	MgmtNetworkID          types.String `tfsdk:"mgmt_network_id"` // [\d\w]+
	Model                  types.String `tfsdk:"model"`
	Name                   types.String `tfsdk:"name"`
	// PortOverrides map[string]DeviceSwitchPortOverrideDataSourceModel `tfsdk:"port_overrides"`
	SnmpContact  types.String `tfsdk:"snmp_contact"`  // .{0,255}
	SnmpLocation types.String `tfsdk:"snmp_location"` // .{0,255}
	State        types.String `tfsdk:"state"`
	StpPriority  types.String `tfsdk:"stp_priority"`
	StpVersion   types.String `tfsdk:"stp_version"`
	Type         types.String `tfsdk:"type"`

	// PortOverrides          []DevicePortOverrides `json:"port_overrides"`
}

type DeviceSwitchConfigNetworkDataSourceModel struct {
	BondingEnabled bool   `json:"bonding_enabled,omitempty"`
	DNS1           string `json:"dns1,omitempty"` // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$|^$
	DNS2           string `json:"dns2,omitempty"` // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$|^$
	DNSsuffix      string `json:"dnssuffix,omitempty"`
	Gateway        string `json:"gateway,omitempty"` // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	IP             string `json:"ip,omitempty"`      // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$
	Netmask        string `json:"netmask,omitempty"` // ^((128|192|224|240|248|252|254)\.0\.0\.0)|(255\.(((0|128|192|224|240|248|252|254)\.0\.0)|(255\.(((0|128|192|224|240|248|252|254)\.0)|255\.(0|128|192|224|240|248|252|254)))))$
	Type           string `json:"type,omitempty"`    // dhcp|static
}

//
// type DeviceSwitchPortOverrideDataSourceModel struct {
// 	AggregateNumPorts        types.Int32  `tfsdk:"aggregate_num_ports"`
// 	ExcludedNetworkIDs       types.List   `tfsdk:"excluded_network_ids"`
// 	Name                     types.String `tfsdk:"name"`
// 	NativeNetworkID          types.String `tfsdk:"native_network_id"`
// 	OpMode                   types.String `tfsdk:"op_mode"`
// 	POEMode                  types.String `tfsdk:"poe_mode"`
// 	PortProfileID            types.String `tfsdk:"port_profile_id"`
// 	PortSecurityEnabled      types.Bool   `tfsdk:"port_security_enabled"`
// 	PortSecurityMACAddresses types.List   `tfsdk:"port_security_mac_addresses"`
//
// 	AggregateNumPorts            int              `json:"aggregate_num_ports,omitempty"` // [1-8]
// 	Autoneg                      bool             `json:"autoneg,omitempty"`
// 	Dot1XCtrl                    string           `json:"dot1x_ctrl,omitempty"`             // auto|force_authorized|force_unauthorized|mac_based|multi_host
// 	Dot1XIDleTimeout             int              `json:"dot1x_idle_timeout,omitempty"`     // [0-9]|[1-9][0-9]{1,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5]
// 	EgressRateLimitKbps          int              `json:"egress_rate_limit_kbps,omitempty"` // 6[4-9]|[7-9][0-9]|[1-9][0-9]{2,6}
// 	EgressRateLimitKbpsEnabled   bool             `json:"egress_rate_limit_kbps_enabled,omitempty"`
// 	ExcludedNetworkIDs           []string         `json:"excluded_networkconf_ids,omitempty"`
// 	FecMode                      string           `json:"fec_mode,omitempty"` // rs-fec|fc-fec|default|disabled
// 	Forward                      string           `json:"forward,omitempty"`  // all|native|customize|disabled
// 	FullDuplex                   bool             `json:"full_duplex,omitempty"`
// 	Isolation                    bool             `json:"isolation,omitempty"`
// 	LldpmedEnabled               bool             `json:"lldpmed_enabled,omitempty"`
// 	LldpmedNotifyEnabled         bool             `json:"lldpmed_notify_enabled,omitempty"`
// 	MirrorPortIDX                int              `json:"mirror_port_idx,omitempty"` // [1-9]|[1-4][0-9]|5[0-2]
// 	NATiveNetworkID              string           `json:"native_networkconf_id,omitempty"`
// 	Name                         string           `json:"name,omitempty"`     // .{0,128}
// 	OpMode                       string           `json:"op_mode,omitempty"`  // switch|mirror|aggregate
// 	PoeMode                      string           `json:"poe_mode,omitempty"` // auto|pasv24|passthrough|off
// 	PortIDX                      int              `json:"port_idx,omitempty"` // [1-9]|[1-4][0-9]|5[0-2]
// 	PortKeepaliveEnabled         bool             `json:"port_keepalive_enabled,omitempty"`
// 	PortProfileID                string           `json:"portconf_id,omitempty"` // [\d\w]+
// 	PortSecurityEnabled          bool             `json:"port_security_enabled,omitempty"`
// 	PortSecurityMACAddress       []string         `json:"port_security_mac_address,omitempty"` // ^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$
// 	PriorityQueue1Level          int              `json:"priority_queue1_level,omitempty"`     // [0-9]|[1-9][0-9]|100
// 	PriorityQueue2Level          int              `json:"priority_queue2_level,omitempty"`     // [0-9]|[1-9][0-9]|100
// 	PriorityQueue3Level          int              `json:"priority_queue3_level,omitempty"`     // [0-9]|[1-9][0-9]|100
// 	PriorityQueue4Level          int              `json:"priority_queue4_level,omitempty"`     // [0-9]|[1-9][0-9]|100
// 	QOSProfile                   DeviceQOSProfile `json:"qos_profile,omitempty"`
// 	SettingPreference            string           `json:"setting_preference,omitempty"` // auto|manual
// 	Speed                        int              `json:"speed,omitempty"`              // 10|100|1000|2500|5000|10000|20000|25000|40000|50000|100000
// 	StormctrlBroadcastastEnabled bool             `json:"stormctrl_bcast_enabled,omitempty"`
// 	StormctrlBroadcastastLevel   int              `json:"stormctrl_bcast_level,omitempty"` // [0-9]|[1-9][0-9]|100
// 	StormctrlBroadcastastRate    int              `json:"stormctrl_bcast_rate,omitempty"`  // [0-9]|[1-9][0-9]{1,6}|1[0-3][0-9]{6}|14[0-7][0-9]{5}|148[0-7][0-9]{4}|14880000
// 	StormctrlMcastEnabled        bool             `json:"stormctrl_mcast_enabled,omitempty"`
// 	StormctrlMcastLevel          int              `json:"stormctrl_mcast_level,omitempty"` // [0-9]|[1-9][0-9]|100
// 	StormctrlMcastRate           int              `json:"stormctrl_mcast_rate,omitempty"`  // [0-9]|[1-9][0-9]{1,6}|1[0-3][0-9]{6}|14[0-7][0-9]{5}|148[0-7][0-9]{4}|14880000
// 	StormctrlType                string           `json:"stormctrl_type,omitempty"`        // level|rate
// 	StormctrlUcastEnabled        bool             `json:"stormctrl_ucast_enabled,omitempty"`
// 	StormctrlUcastLevel          int              `json:"stormctrl_ucast_level,omitempty"` // [0-9]|[1-9][0-9]|100
// 	StormctrlUcastRate           int              `json:"stormctrl_ucast_rate,omitempty"`  // [0-9]|[1-9][0-9]{1,6}|1[0-3][0-9]{6}|14[0-7][0-9]{5}|148[0-7][0-9]{4}|14880000
// 	StpPortMode                  bool             `json:"stp_port_mode,omitempty"`
// 	TaggedVLANMgmt               string           `json:"tagged_vlan_mgmt,omitempty"` // auto|block_all|custom
// 	VoiceNetworkID               string           `json:"voice_networkconf_id,omitempty"`
// }
//
// type DeviceQOSProfile struct {
// 	QOSPolicies    []DeviceQOSPolicies `json:"qos_policies,omitempty"`
// 	QOSProfileMode string              `json:"qos_profile_mode,omitempty"` // custom|unifi_play|aes67_audio|crestron_audio_video|dante_audio|ndi_aes67_audio|ndi_dante_audio|qsys_audio_video|qsys_video_dante_audio|sdvoe_aes67_audio|sdvoe_dante_audio|shure_audio
// }
//
// type DeviceQOSPolicies struct {
// 	QOSMarking  DeviceQOSMarking  `json:"qos_marking,omitempty"`
// 	QOSMatching DeviceQOSMatching `json:"qos_matching,omitempty"`
// }
//
// type DeviceQOSMarking struct {
// 	CosCode          int `json:"cos_code,omitempty"`           // [0-7]
// 	DscpCode         int `json:"dscp_code,omitempty"`          // 0|8|16|24|32|40|48|56|10|12|14|18|20|22|26|28|30|34|36|38|44|46
// 	IPPrecedenceCode int `json:"ip_precedence_code,omitempty"` // [0-7]
// 	Queue            int `json:"queue,omitempty"`              // [0-7]
// }
//
// type DeviceQOSMatching struct {
// 	CosCode          int    `json:"cos_code,omitempty"`           // [0-7]
// 	DscpCode         int    `json:"dscp_code,omitempty"`          // [0-9]|[1-5][0-9]|6[0-3]
// 	DstPort          int    `json:"dst_port,omitempty"`           // [0-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]|[1-5][0-9][0-9][0-9][0-9]|6[0-4][0-9][0-9][0-9]|65[0-4][0-9][0-9]|655[0-2][0-9]|6553[0-4]|65535
// 	IPPrecedenceCode int    `json:"ip_precedence_code,omitempty"` // [0-7]
// 	Protocol         string `json:"protocol,omitempty"`           // ([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])|ah|ax.25|dccp|ddp|egp|eigrp|encap|esp|etherip|fc|ggp|gre|hip|hmp|icmp|idpr-cmtp|idrp|igmp|igp|ip|ipcomp|ipencap|ipip|ipv6|ipv6-frag|ipv6-icmp|ipv6-nonxt|ipv6-opts|ipv6-route|isis|iso-tp4|l2tp|manet|mobility-header|mpls-in-ip|ospf|pim|pup|rdp|rohc|rspf|rsvp|sctp|shim6|skip|st|tcp|udp|udplite|vmtp|vrrp|wesp|xns-idp|xtp
// 	SrcPort          int    `json:"src_port,omitempty"`           // [0-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]|[1-5][0-9][0-9][0-9][0-9]|6[0-4][0-9][0-9][0-9]|65[0-4][0-9][0-9]|655[0-2][0-9]|6553[0-4]|65535
// }

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
				Validators:          []validator.String{
					// TODO: (jtoyer) Add a mac address validator
				},
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
			"disabled": schema.BoolAttribute{
				Computed: true,
			},
			"dot1x_fallback_networkconf_id": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^([\\d\\w]+|)$"), "must only contain letters and numbers or must be empty"),
				},
			},
			"dot1x_portctrl_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"flowctrl_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"jumboframe_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"mgmt_network_id": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^([\\d\\w]+)$"), "must only contain letters and numbers"),
				},
			},
			"model": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 128),
				},
			},
			// "port_overrides": schema.MapNestedAttribute{
			// 	Computed: true,
			// 	NestedObject: schema.NestedAttributeObject{
			// 		Attributes: map[string]schema.Attribute{
			// 			"aggregate_num_ports": schema.Int32Attribute{
			// 				Computed: true,
			// 			},
			// 			"excluded_network_ids": schema.ListAttribute{
			// 				ElementType: types.StringType,
			// 				Computed:    true,
			// 			},
			// 			"name": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"native_network_id": schema.StringAttribute{
			// 				MarkdownDescription: "The native network used for VLAN traffic, i.e. not tagged with a " +
			// 					"VLAN ID. Untagged traffic from devices connected to this port will be placed on to " +
			// 					"the selected VLAN",
			// 				Computed: true,
			// 			},
			// 			"op_mode": schema.StringAttribute{
			// 				Computed: true,
			// 			},
			// 			"poe_mode": schema.StringAttribute{
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
			// 		},
			// 	},
			// },
			"snmp_contact": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 255),
				},
			},
			"snmp_location": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 255),
				},
			},
			"state": schema.StringAttribute{
				Computed: true,
			},
			"stp_priority": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("0", "4096", "8192", "12288", "16384", "20480", "24576", "28672", "32768", "36864", "40960", "45056", "49152", "53248", "57344", "61440"),
				},
			},
			"stp_version": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("stp", "rstp", "disabled"),
					stringvalidator.RegexMatches(regexp.MustCompile("^([\\d\\w]+|)$"), "must only contain letters and numbers or must be empty"),
				},
			},
			"type": schema.StringAttribute{
				Computed: true,
			},

			// "config_network": schema.StringAttribute{},
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

	data.ID = types.StringValue(device.ID)
	data.Adopted = types.BoolValue(device.Adopted)
	data.Dot1XFallbackNetworkID = types.StringValue(device.Dot1XFallbackNetworkID)
	data.Dot1XPortctrlEnabled = types.BoolValue(device.Dot1XPortctrlEnabled)
	data.FlowctrlEnabled = types.BoolValue(device.FlowctrlEnabled)
	data.JumboframeEnabled = types.BoolValue(device.JumboframeEnabled)
	data.MgmtNetworkID = types.StringValue(device.MgmtNetworkID)
	data.Model = types.StringValue(device.Model)
	data.Name = types.StringValue(device.Name)
	data.SnmpContact = types.StringValue(device.SnmpContact)
	data.SnmpLocation = types.StringValue(device.SnmpLocation)
	data.State = types.StringValue(device.State.String())
	data.StpPriority = types.StringValue(device.StpPriority)
	data.StpVersion = types.StringValue(device.StpVersion)
	data.Type = types.StringValue(device.Type)

	// data.PortOverrides = make(map[string]DeviceSwitchPortOverrideDataSourceModel, len(device.PortOverrides))
	// for _, override := range device.PortOverrides {
	// 	excludedNetworkIDs := types.ListNull(types.StringType)
	// 	if override.ExcludedNetworkIDs != nil {
	// 		var attrs []attr.Value
	// 		for _, id := range override.ExcludedNetworkIDs {
	// 			attrs = append(attrs, types.StringValue(id))
	// 		}
	//
	// 		excludedNetworkIDs = types.ListValueMust(types.StringType, attrs)
	// 	}
	//
	// 	portSecurityMACAddresses := types.ListNull(types.StringType)
	// 	if override.PortSecurityMACAddress != nil {
	// 		var attrs []attr.Value
	// 		for _, id := range override.PortSecurityMACAddress {
	// 			attrs = append(attrs, types.StringValue(id))
	// 		}
	//
	// 		portSecurityMACAddresses = types.ListValueMust(types.StringType, attrs)
	// 	}
	//
	// 	data.PortOverrides[strconv.Itoa(override.PortIDX)] = DeviceSwitchPortOverrideDataSourceModel{
	// 		AggregateNumPorts:        types.Int32Value(int32(override.AggregateNumPorts)),
	// 		ExcludedNetworkIDs:       excludedNetworkIDs,
	// 		Name:                     types.StringValue(override.Name),
	// 		NativeNetworkID:          types.StringValue(override.NATiveNetworkID),
	// 		OpMode:                   types.StringValue(override.OpMode),
	// 		POEMode:                  types.StringValue(override.PoeMode),
	// 		PortProfileID:            types.StringValue(override.PortProfileID),
	// 		PortSecurityEnabled:      types.BoolValue(override.PortSecurityEnabled),
	// 		PortSecurityMACAddresses: portSecurityMACAddresses,
	// 	}
	// }

	tflog.Trace(ctx, "device read", map[string]interface{}{"mac": data.Mac.ValueString()})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
