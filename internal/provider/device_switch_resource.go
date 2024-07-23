package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamestoyer/go-unifi/unifi"
	"github.com/jamestoyer/terraform-provider-unifi/internal/provider/customplanmodifier"
	"github.com/jamestoyer/terraform-provider-unifi/internal/provider/customvalidator"
	"github.com/jamestoyer/terraform-provider-unifi/internal/provider/utils"
	"regexp"
	"strconv"
)

const (
	configNetworkTypeDHCP   = "dhcp"
	configNetworkTypeStatic = "static"

	ledOverrideDefault = "default"
	ledOverrideOff     = "off"
	ledOverrideOn      = "on"

	portOverrideSettingPreferenceAuto   = "auto"
	portOverrideSettingPreferenceManual = "manual"
)

var (
	// Ensure provider defined types fully satisfy framework interfaces.
	_ resource.Resource                = &DeviceSwitchResource{}
	_ resource.ResourceWithImportState = &DeviceSwitchResource{}

	defaultDeviceSwitchLEDOverrideResourceModel = DeviceSwitchLEDSettingsResourceModel{
		Enabled: types.BoolValue(true),
	}
	defaultDeviceSwitchPortOverrideModel             = DeviceSwitchPortOverrideResourceModel{}
	defaultDeviceSwitchResourceModel                 = DeviceSwitchResourceModel{}
	defaultDeviceSwitchStaticIPSettingsResourceModel = DeviceSwitchStaticIPSettingResourceModel{}
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
	resp.Schema = defaultDeviceSwitchResourceModel.schema(ctx, resp)
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

	data, diags := newDeviceSwitchResourceModel(ctx, device, site, data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

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

	device, diags := data.toUnifiDevice(ctx)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	device.ID = data.ID.ValueStringPointer()

	device, err := r.client.UpdateDevice(ctx, site, device)
	if err != nil {
		// When there are no changes in v8 the API doesn't return the device details. This causes the client to assume
		// the device doesn't exist. To work around this for now do a read to get the status.
		// TODO: (jtoyer) Update the client to handle no changes on update, i.e. a 200 but no body
		if !errors.Is(err, &unifi.NotFoundError{}) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update switch, got error: %s", err))
			return
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// TODO: (jtoyer) Wait until device has finished updating after before saving the state
	data, diags = newDeviceSwitchResourceModel(ctx, device, site, data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

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
	Disabled            types.Bool                                       `tfsdk:"disabled"`
	LEDSettings         *DeviceSwitchLEDSettingsResourceModel            `tfsdk:"led_settings"`
	Mac                 types.String                                     `tfsdk:"mac"`
	ManagementNetworkID types.String                                     `tfsdk:"management_network_id"`
	Name                types.String                                     `tfsdk:"name"`
	PortOverrides       map[string]DeviceSwitchPortOverrideResourceModel `tfsdk:"port_overrides"`
	Site                types.String                                     `tfsdk:"site"`
	SNMPContact         types.String                                     `tfsdk:"snmp_contact"`
	SNMPLocation        types.String                                     `tfsdk:"snmp_location"`
	StaticIPSettings    *DeviceSwitchStaticIPSettingResourceModel        `tfsdk:"static_ip_settings"`
}

func (m *DeviceSwitchResourceModel) schema(ctx context.Context, resp *resource.SchemaResponse) schema.Schema {
	// Create a default for port overrides
	overrideAttributes := types.ObjectType{AttrTypes: map[string]attr.Type{}}
	for name, attribute := range defaultDeviceSwitchPortOverrideModel.schema().Attributes {
		overrideAttributes.AttrTypes[name] = attribute.GetType()
	}

	defaultPortOverrides, diags := types.MapValue(overrideAttributes, map[string]attr.Value{})
	if diags.HasError() {
		panic(diags)
	}

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
			// TODO: (jtoyer) To enable these we need to set an exclusion on unifi.SettingGlobalSwitch
			// "jumboframe_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			"led_settings": defaultDeviceSwitchLEDOverrideResourceModel.schema(ctx, resp),
			"mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the device",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// TODO: (jtoyer) Add a plan modifier to ignore case changes for the MAC
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					// TODO: (jtoyer) Add a mac address validator
				},
			},
			"management_network_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the VLAN to use as the management VLAN instead of the default tagged " +
					"network from the upstream device.",
				Required: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "A name to assign to the device",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(128),
				},
			},
			"port_overrides": schema.MapNestedAttribute{
				Computed:     true,
				Optional:     true,
				Default:      mapdefault.StaticValue(defaultPortOverrides),
				NestedObject: defaultDeviceSwitchPortOverrideModel.schema(),
			},
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
				Computed: true,
				Optional: true,
				Default:  stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"snmp_location": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Default:  stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			// TODO: (jtoyer) To enable these we need to set an exclusion on unifi.SettingGlobalSwitch
			// "stp_priority": schema.StringAttribute{
			// 	Computed: true,
			// 	Optional: true,
			// 	Validators: []validator.String{
			// 		stringvalidator.OneOf("0", "4096", "8192", "12288", "16384", "20480", "24576", "28672",
			// 			"32768", "36864", "40960", "45056", "49152", "53248", "57344", "61440"),
			// 	},
			// 	Default: stringdefault.StaticString("0"),
			// },
			// TODO: (jtoyer) To enable these we need to set an exclusion on unifi.SettingGlobalSwitch
			// "stp_version": schema.StringAttribute{
			// 	Computed: true,
			// },
			"static_ip_settings": defaultDeviceSwitchStaticIPSettingsResourceModel.schema(),
		},
	}
}

func (m *DeviceSwitchResourceModel) toUnifiDevice(ctx context.Context) (*unifi.Device, diag.Diagnostics) {
	var diags diag.Diagnostics
	var portOverrides []unifi.DevicePortOverrides
	for index, override := range m.PortOverrides {
		i, err := strconv.Atoi(index)
		if err != nil {
			diags.AddAttributeWarning(path.Root("port_overrides"), "Invalid Port Index",
				fmt.Sprintf("Expected a number for the port index instead got %s: %s", index, err))
			continue
		}

		p, pDiags := override.toUnifiStruct(ctx, i)
		diags = append(diags, pDiags...)
		portOverrides = append(portOverrides, p)
	}

	if m.ManagementNetworkID.ValueString() == "" {
		diags.AddAttributeError(path.Root("management_network_id"), "Invalid ID", "The ID of the management network must not be empty")
	}

	device := &unifi.Device{
		ConfigNetwork:              m.StaticIPSettings.toUnifiStruct(),
		Disabled:                   m.Disabled.ValueBoolPointer(),
		LedOverride:                m.LEDSettings.GetOverrideState().ValueStringPointer(),
		LedOverrideColor:           m.LEDSettings.Color.ValueStringPointer(),
		LedOverrideColorBrightness: utils.IntPtrValue(m.LEDSettings.Brightness.ValueInt32Pointer()),
		MAC:                        m.Mac.ValueStringPointer(),
		MgmtNetworkID:              m.ManagementNetworkID.ValueStringPointer(),
		Name:                       m.Name.ValueStringPointer(),
		PortOverrides:              portOverrides,
		SnmpContact:                m.SNMPContact.ValueStringPointer(),
		SnmpLocation:               m.SNMPLocation.ValueStringPointer(),
	}

	return device, diags
}

func newDeviceSwitchResourceModel(ctx context.Context, device *unifi.Device, site string, model DeviceSwitchResourceModel) (DeviceSwitchResourceModel, diag.Diagnostics) {
	// Computed values
	model.Model = types.StringPointerValue(device.Model)
	model.Site = types.StringValue(site)
	model.SiteID = types.StringPointerValue(device.SiteID)

	// Configurable Values
	model.Disabled = types.BoolPointerValue(device.Disabled)
	model.LEDSettings = newDeviceSwitchLEDOverrideResourceModel(device, model.LEDSettings)
	model.Mac = types.StringPointerValue(device.MAC)
	model.ManagementNetworkID = types.StringPointerValue(device.MgmtNetworkID)
	model.Name = types.StringPointerValue(device.Name)
	model.SNMPContact = types.StringPointerValue(device.SnmpContact)
	model.SNMPLocation = types.StringPointerValue(device.SnmpLocation)
	model.StaticIPSettings = newDeviceSwitchStaticIPSettingsResourceModel(device.ConfigNetwork, model.StaticIPSettings)

	var diags diag.Diagnostics
	overrides := make(map[string]DeviceSwitchPortOverrideResourceModel, len(device.PortOverrides))
	for _, override := range device.PortOverrides {
		index := strconv.Itoa(*override.PortIDX)
		m, err := newDeviceSwitchPortOverrideResourceModel(ctx, override)
		if err.HasError() {
			diags.Append(err...)
			continue
		}

		overrides[index] = m
	}

	model.PortOverrides = overrides
	return model, diags
}

type DeviceSwitchStaticIPSettingResourceModel struct {
	AlternativeDNS iptypes.IPv4Address `tfsdk:"alternative_dns"`
	BondingEnabled types.Bool          `tfsdk:"bonding_enabled"`
	DNSSuffix      types.String        `tfsdk:"dns_suffix"`
	Gateway        iptypes.IPv4Address `tfsdk:"gateway"`
	IP             iptypes.IPv4Address `tfsdk:"ip"`
	Netmask        iptypes.IPv4Address `tfsdk:"netmask"`
	PreferredDNS   iptypes.IPv4Address `tfsdk:"preferred_dns"`
}

func (m *DeviceSwitchStaticIPSettingResourceModel) schema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Force the switch to use a static IP address instead of one assigned by DHCP.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"alternative_dns": schema.StringAttribute{
				Optional:   true,
				CustomType: iptypes.IPv4AddressType{},
			},
			"bonding_enabled": schema.BoolAttribute{
				Computed: true,
				Optional: true,
				Default:  booldefault.StaticBool(false),
			},
			"dns_suffix": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Default:  stringdefault.StaticString(""),
			},
			"gateway": schema.StringAttribute{
				Required:   true,
				CustomType: iptypes.IPv4AddressType{},
			},
			"ip": schema.StringAttribute{
				Required:   true,
				CustomType: iptypes.IPv4AddressType{},
			},
			"netmask": schema.StringAttribute{
				Required:   true,
				CustomType: iptypes.IPv4AddressType{},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^((128|192|224|240|248|252|254)\.0\.0\.0)|(255\.(((0|128|192|224|240|248|252|254)\.0\.0)|(255\.(((0|128|192|224|240|248|252|254)\.0)|255\.(0|128|192|224|240|248|252|254)))))$`), "invalid net mask"),
				},
			},
			"preferred_dns": schema.StringAttribute{
				Required:   true,
				CustomType: iptypes.IPv4AddressType{},
			},
		},
	}
}

func (m *DeviceSwitchStaticIPSettingResourceModel) toUnifiStruct() *unifi.DeviceConfigNetwork {
	if m == nil {
		return &unifi.DeviceConfigNetwork{Type: utils.StringPtr(configNetworkTypeDHCP)}
	}

	return &unifi.DeviceConfigNetwork{
		DNS2:           m.AlternativeDNS.ValueStringPointer(),
		BondingEnabled: m.BondingEnabled.ValueBoolPointer(),
		// TODO: (jtoyer) fix DNS Suffix field name in the unifi client
		DNSsuffix: m.DNSSuffix.ValueStringPointer(),
		Gateway:   m.Gateway.ValueStringPointer(),
		IP:        m.IP.ValueStringPointer(),
		Netmask:   m.Netmask.ValueStringPointer(),
		DNS1:      m.PreferredDNS.ValueStringPointer(),
		Type:      utils.StringPtr(configNetworkTypeStatic),
	}
}

func newDeviceSwitchStaticIPSettingsResourceModel(network *unifi.DeviceConfigNetwork, model *DeviceSwitchStaticIPSettingResourceModel) *DeviceSwitchStaticIPSettingResourceModel {
	if *network.Type == configNetworkTypeDHCP {
		return nil
	}

	if model == nil {
		model = &DeviceSwitchStaticIPSettingResourceModel{}
	}

	if network.DNS2 != nil && *network.DNS2 != "" {
		model.AlternativeDNS = iptypes.NewIPv4AddressPointerValue(network.DNS2)
	}

	model.BondingEnabled = types.BoolPointerValue(network.BondingEnabled)
	model.DNSSuffix = types.StringPointerValue(network.DNSsuffix)
	model.Gateway = iptypes.NewIPv4AddressPointerValue(network.Gateway)
	model.IP = iptypes.NewIPv4AddressPointerValue(network.IP)
	model.Netmask = iptypes.NewIPv4AddressPointerValue(network.Netmask)
	model.PreferredDNS = iptypes.NewIPv4AddressPointerValue(network.DNS1)

	return model
}

type DeviceSwitchLEDSettingsResourceModel struct {
	Brightness types.Int32  `tfsdk:"brightness"`
	Color      types.String `tfsdk:"color"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func (m *DeviceSwitchLEDSettingsResourceModel) schema(ctx context.Context, resp *resource.SchemaResponse) schema.Attribute {
	attrs := map[string]schema.Attribute{
		"brightness": schema.Int32Attribute{
			Optional: true,
			Validators: []validator.Int32{
				int32validator.Between(0, 100),
			},
		},
		"color": schema.StringAttribute{
			Optional: true,
			Validators: []validator.String{
				stringvalidator.RegexMatches(regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}){1,2}$`), "invalid color code"),
			},
		},
		"enabled": schema.BoolAttribute{
			Computed: true,
			Optional: true,
			Default:  booldefault.StaticBool(true),
		},
	}

	typeAttrs := map[string]attr.Type{}
	for name, attribute := range attrs {
		typeAttrs[name] = attribute.GetType()
	}

	defaultValue, diags := types.ObjectValueFrom(ctx, typeAttrs, defaultDeviceSwitchLEDOverrideResourceModel)
	resp.Diagnostics.Append(diags...)

	return schema.SingleNestedAttribute{
		MarkdownDescription: "Overrides for the switch LEDs.",
		Computed:            true,
		Optional:            true,
		Default:             objectdefault.StaticValue(defaultValue),
		Attributes:          attrs,
	}
}

func (m *DeviceSwitchLEDSettingsResourceModel) GetOverrideState() types.String {
	if m == nil || m.Enabled.ValueBool() {
		return types.StringValue(ledOverrideOn)
	}

	return types.StringValue(ledOverrideOff)
}

func newDeviceSwitchLEDOverrideResourceModel(device *unifi.Device, model *DeviceSwitchLEDSettingsResourceModel) *DeviceSwitchLEDSettingsResourceModel {
	if model == nil {
		model = &DeviceSwitchLEDSettingsResourceModel{}
	}

	if device.LedOverride == nil || *device.LedOverride == ledOverrideDefault {
		model.Enabled = types.BoolValue(true)
		return model
	}

	model.Brightness = types.Int32PointerValue(utils.Int32PtrValue(device.LedOverrideColorBrightness))
	model.Color = types.StringPointerValue(device.LedOverrideColor)
	if *device.LedOverride == ledOverrideOn {
		model.Enabled = types.BoolValue(true)
	} else {
		model.Enabled = types.BoolValue(false)
	}

	return model
}

type DeviceSwitchPortOverrideResourceModel struct {
	// Configurable Values
	// Disabled   types.Bool   `tfsdk:"disabled"`
	AggregateNumPorts        types.Int32  `tfsdk:"aggregate_num_ports"`
	ExcludedTaggedNetworkIds types.List   `tfsdk:"excluded_tagged_network_ids"`
	FullDuplex               types.Bool   `tfsdk:"full_duplex"`
	LinkSpeed                types.Int32  `tfsdk:"link_speed"`
	MirrorPortIndex          types.Int32  `tfsdk:"mirror_port_index"`
	Name                     types.String `tfsdk:"name"`
	NativeNetworkID          types.String `tfsdk:"native_network_id"`
	Operation                types.String `tfsdk:"operation"`
	POEMode                  types.String `tfsdk:"poe_mode"`
	PortProfileID            types.String `tfsdk:"port_profile_id"`
	TaggedVLANManagement     types.String `tfsdk:"tagged_vlan_management"`
}

func (m *DeviceSwitchPortOverrideResourceModel) schema() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"aggregate_num_ports": schema.Int32Attribute{
				Optional: true,
				Validators: []validator.Int32{
					int32validator.Between(1, 8),
					int32validator.AlsoRequires(path.MatchRelative().AtParent().AtName("operation")),
				},
			},
			// "disabled": schema.BoolAttribute{
			// 	Computed: true,
			// 	Optional: true,
			// 	Default:  booldefault.StaticBool(false),
			// },
			// "dot1x_ctrl": schema.StringAttribute{
			// 	Computed: true,
			// },
			// "dot1x_idle_timeout": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "egress_rate_limit_kbps": schema.Int32Attribute{
			// 	MarkdownDescription: "Sets a port's maximum rate of data transfer.",
			// 	Computed:            true,
			// },
			// "egress_rate_limit_kbps_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			"excluded_tagged_network_ids": schema.ListAttribute{
				MarkdownDescription: "One or more VLANs that are tagged on this port.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.List{
					listvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("tagged_vlan_management")),
				},
			},
			// "fec_mode": schema.StringAttribute{
			// 	Computed: true,
			// },
			// "forward": schema.StringAttribute{
			// 	Computed: true,
			// },
			"full_duplex": schema.BoolAttribute{
				Optional: true,
				Validators: []validator.Bool{
					boolvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("link_speed")),
				},
			},
			// "isolation": schema.BoolAttribute{
			// 	MarkdownDescription: "Allows you to prohibit traffic between isolated ports. This only " +
			// 		"applies to ports on the same device.",
			// 	Computed: true,
			// },
			"link_speed": schema.Int32Attribute{
				MarkdownDescription: "An override for the link speed of the port.",
				Optional:            true,
				Validators: []validator.Int32{
					int32validator.OneOf(10, 100, 1000, 2500, 5000, 10000, 20000, 25000, 40000, 50000, 100000),
					int32validator.AlsoRequires(path.MatchRelative().AtParent().AtName("full_duplex")),
				},
			},
			// "lldp_med_enabled": schema.BoolAttribute{
			// 	MarkdownDescription: "Extension for LLPD user alongside the voice VLAN feature to " +
			// 		"discover the presence of a VoIP phone. Disabling LLPD-MED will also disable the " +
			// 		"Voice VLAN.",
			// 	Computed: true,
			// },
			// "lldp_med_notify_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			"mirror_port_index": schema.Int32Attribute{
				MarkdownDescription: "The index of the port to mirror traffic to.",
				Optional:            true,
				Validators: []validator.Int32{
					int32validator.Between(1, 52),
					int32validator.AlsoRequires(path.MatchRelative().AtParent().AtName("operation")),
				},
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
					"the selected VLAN. Setting this to and empty string (which this defaults to) will prevent" +
					" untagged traffic from being placed in to a VLAN by default.",
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					// When this is unknown we should just use the state. If the remote value has changed that's means
					// it's been hand jammed and should really be set in TF instead. The only time it will change is if
					// there is a port profile set.
					stringplanmodifier.UseStateForUnknown(),
					customplanmodifier.PortOverridePortProfileIDString(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("port_profile_id")),
				},
			},
			"operation": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Default:  stringdefault.StaticString("switch"),
				PlanModifiers: []planmodifier.String{
					customplanmodifier.PortOverridePortProfileIDString(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("switch", "mirror", "aggregate"),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("port_profile_id")),
					customvalidator.StringValueWithPaths("aggregate", path.MatchRelative().AtParent().AtName("aggregate_num_ports")),
					customvalidator.StringValueWithPaths("mirror", path.MatchRelative().AtParent().AtName("mirror_port_index")),
				},
			},
			"poe_mode": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Default:  stringdefault.StaticString("auto"),
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "pasv24", "passthrough", "off"),
				},
			},
			// "port_keepalive_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			"port_profile_id": schema.StringAttribute{
				MarkdownDescription: "The ID of a port profile to assign to the port. This will override nearly all" +
					" local settings of the port.",
				Optional: true,
			},
			// "port_security_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			// "port_security_mac_addresses": schema.ListAttribute{
			// 	ElementType: types.StringType,
			// 	Computed:    true,
			// },
			// "priority_queue1_level": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "priority_queue2_level": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "priority_queue3_level": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "priority_queue4_level": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "qos_profile": schema.SingleNestedAttribute{
			// 	Computed: true,
			// 	Attributes: map[string]schema.Attribute{
			// 		"qos_policies": schema.SetNestedAttribute{
			// 			Computed: true,
			// 			NestedObject: schema.NestedAttributeObject{
			//
			// 				Attributes: map[string]schema.Attribute{
			// 					"qos_marking": schema.SingleNestedAttribute{
			// 						Computed: true,
			// 						Attributes: map[string]schema.Attribute{
			// 							"cos_code": schema.Int32Attribute{
			// 								Computed: true,
			// 							},
			// 							"dscp_code": schema.Int32Attribute{
			// 								Computed: true,
			// 							},
			// 							"ip_precedence_code": schema.Int32Attribute{
			// 								Computed: true,
			// 							},
			// 							"queue": schema.Int32Attribute{
			// 								Computed: true,
			// 							},
			// 						},
			// 					},
			// 					"qos_matching": schema.SingleNestedAttribute{
			// 						Computed: true,
			// 						Attributes: map[string]schema.Attribute{
			// 							"cos_code": schema.Int32Attribute{
			// 								Computed: true,
			// 							},
			// 							"dscp_code": schema.Int32Attribute{
			// 								Computed: true,
			// 							},
			// 							"dst_port": schema.Int32Attribute{
			// 								Computed: true,
			// 							},
			// 							"ip_precedence_code": schema.Int32Attribute{
			// 								Computed: true,
			// 							},
			// 							"protocol": schema.StringAttribute{
			// 								Computed: true,
			// 							},
			// 							"src_port": schema.Int32Attribute{
			// 								Computed: true,
			// 							},
			// 						},
			// 					},
			// 				},
			// 			},
			// 		},
			// 		"qos_profile_mode": schema.StringAttribute{
			// 			Computed: true,
			// 		},
			// 	},
			// },
			// "storm_control_broadcast_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			// "storm_control_broadcast_level": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "storm_control_broadcast_rate": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "storm_control_multicast_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			// "storm_control_multicast_level": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "storm_control_mulitcast_rate": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "storm_control_type": schema.StringAttribute{
			// 	Computed: true,
			// },
			// "storm_control_unicast_enabled": schema.BoolAttribute{
			// 	Computed: true,
			// },
			// "storm_control_unicast_level": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "storm_control_unicast_rate": schema.Int32Attribute{
			// 	Computed: true,
			// },
			// "stp_port_mode": schema.BoolAttribute{
			// 	Computed: true,
			// },
			"tagged_vlan_management": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Default:  stringdefault.StaticString("auto"),
				PlanModifiers: []planmodifier.String{
					customplanmodifier.PortOverridePortProfileIDString(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "block_all", "custom"),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("port_profile_id")),
					customvalidator.StringValueWithPaths("custom", path.MatchRelative().AtParent().AtName("excluded_tagged_network_ids")),
				},
			},
			// "voice_networkconf_id": schema.StringAttribute{
			// 	MarkdownDescription: "Uses LLPD-MED to place a VoIP phone on the specified VLAN. Devices " +
			// 		"connected to the phone are placed in the Native VLAN.",
			// 	Computed: true,
			// },
		},
	}
}

func (m *DeviceSwitchPortOverrideResourceModel) toUnifiStruct(ctx context.Context, portIndex int) (unifi.DevicePortOverrides, diag.Diagnostics) {
	var diags diag.Diagnostics

	settingPreference := portOverrideSettingPreferenceAuto
	autoNegotiateLinkSpeed := true
	if !m.LinkSpeed.IsNull() {
		autoNegotiateLinkSpeed = false
		settingPreference = portOverrideSettingPreferenceManual
	}

	nativeNetworkID := m.NativeNetworkID
	if m.NativeNetworkID.IsUnknown() {
		// When the value is unknown, set it to a nil string. This will ensure the value isn't overridden
		nativeNetworkID = types.StringPointerValue(nil)
	}

	var excludedNetworkIDs *[]string
	if !m.ExcludedTaggedNetworkIds.IsNull() {
		elements := make([]string, 0, len(m.ExcludedTaggedNetworkIds.Elements()))
		eDiags := m.ExcludedTaggedNetworkIds.ElementsAs(ctx, &elements, false)
		diags.Append(eDiags...)
		excludedNetworkIDs = &elements
	}

	return unifi.DevicePortOverrides{
		// Computed values
		Autoneg:           &autoNegotiateLinkSpeed,
		PortIDX:           &portIndex,
		SettingPreference: &settingPreference,

		// Configurable Values
		AggregateNumPorts:  utils.IntPtrValue(m.AggregateNumPorts.ValueInt32Pointer()),
		ExcludedNetworkIDs: excludedNetworkIDs,
		FullDuplex:         m.FullDuplex.ValueBoolPointer(),
		MirrorPortIDX:      utils.IntPtrValue(m.MirrorPortIndex.ValueInt32Pointer()),
		Name:               m.Name.ValueStringPointer(),
		NATiveNetworkID:    nativeNetworkID.ValueStringPointer(),
		OpMode:             m.Operation.ValueStringPointer(),
		PoeMode:            m.POEMode.ValueStringPointer(),
		PortProfileID:      m.PortProfileID.ValueStringPointer(),
		Speed:              utils.IntPtrValue(m.LinkSpeed.ValueInt32Pointer()),
		TaggedVLANMgmt:     m.TaggedVLANManagement.ValueStringPointer(),
	}, diags
}

func newDeviceSwitchPortOverrideResourceModel(ctx context.Context, override unifi.DevicePortOverrides) (DeviceSwitchPortOverrideResourceModel, diag.Diagnostics) {
	excludedNetworkIDs, diags := types.ListValueFrom(ctx, types.StringType, override.ExcludedNetworkIDs)

	return DeviceSwitchPortOverrideResourceModel{
		// Configurable Values
		AggregateNumPorts:        types.Int32PointerValue(utils.Int32PtrValue(override.AggregateNumPorts)),
		ExcludedTaggedNetworkIds: excludedNetworkIDs,
		FullDuplex:               types.BoolPointerValue(override.FullDuplex),
		LinkSpeed:                types.Int32PointerValue(utils.Int32PtrValue(override.Speed)),
		MirrorPortIndex:          types.Int32PointerValue(utils.Int32PtrValue(override.MirrorPortIDX)),
		Name:                     types.StringPointerValue(override.Name),
		NativeNetworkID:          types.StringPointerValue(override.NATiveNetworkID),
		Operation:                types.StringPointerValue(override.OpMode),
		POEMode:                  types.StringPointerValue(override.PoeMode),
		PortProfileID:            types.StringPointerValue(override.PortProfileID),
		TaggedVLANManagement:     types.StringPointerValue(override.TaggedVLANMgmt),
	}, diags
}
