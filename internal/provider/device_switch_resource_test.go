package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"regexp"
	"testing"
)

func TestAccDeviceSwitchResource_Empty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDeviceConfigEmpty(),
				ExpectError: regexp.MustCompile(`The argument "management_network_id" is required`),
			},
			{
				Config:      testAccDeviceConfigEmpty(),
				ExpectError: regexp.MustCompile(`The argument "mac" is required`),
			},
			{
				Config:      testAccDeviceConfigEmpty(),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
		},
	})
}

func testAccDeviceConfigEmpty() string {
	return `
provider "unifi" {}
resource "unifi_device_switch" "example" {}
`
}

func TestAccDeviceSwitchResource_Simple(t *testing.T) {
	ctx := context.Background()
	device, releaseDevice := getSwitchDevice(ctx, t)
	defer releaseDevice()

	network := getNetwork(ctx, t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDeviceConfigSimple(*device.MAC, *network.ID, "Test Switch"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device_switch.test", "name", "Test Switch"),
					resource.TestCheckNoResourceAttr("unifi_device_switch.test", "static_ip_settings"),
					resource.TestCheckResourceAttrWith("unifi_device_switch.test", "id", func(value string) error {
						if value == "" {
							return errors.New("id is required")
						}

						return nil
					}),
				),
			},
			// Update and Read testing
			{
				Config: testAccDeviceConfigSimple(*device.MAC, *network.ID, "Updated Test Switch"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device_switch.test", "name", "Updated Test Switch"),
					resource.TestCheckNoResourceAttr("unifi_device_switch.test", "static_ip_settings"),
					resource.TestCheckResourceAttrWith("unifi_device_switch.test", "id", func(value string) error {
						if value == "" {
							return errors.New("id is required")
						}

						return nil
					}),
				),
			},
		},
	})
}

func testAccDeviceConfigSimple(macAddress, managementNetworkID, name string) string {
	return fmt.Sprintf(`
provider "unifi" {}
resource "unifi_device_switch" "test" {
  name                  = %[1]q
  mac                   = %[2]q
  management_network_id = %[3]q
}
`, name, macAddress, managementNetworkID)
}

func TestAccDeviceSwitchResource_StaticIPSettings(t *testing.T) {
	ctx := context.Background()
	device, releaseDevice := getSwitchDevice(ctx, t)
	defer releaseDevice()

	network := getNetwork(ctx, t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDeviceConfigStaticIPSettings(*device.MAC, *network.ID, "10.2.3.4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device_switch.test", "static_ip_settings.ip", "10.2.3.4"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", "static_ip_settings.gateway", "10.2.0.1"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", "static_ip_settings.netmask", "255.255.255.0"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", "static_ip_settings.preferred_dns", "1.2.3.4"),
				),
			},
			// Update and Read testing
			{
				Config: testAccDeviceConfigStaticIPSettings(*device.MAC, *network.ID, "192.168.1.10"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device_switch.test", "static_ip_settings.ip", "192.168.1.10"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", "static_ip_settings.gateway", "10.2.0.1"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", "static_ip_settings.netmask", "255.255.255.0"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", "static_ip_settings.preferred_dns", "1.2.3.4"),
				),
			},
		},
	})
}

func testAccDeviceConfigStaticIPSettings(macAddress, managementNetworkID, ip string) string {
	return fmt.Sprintf(`
provider "unifi" {}
resource "unifi_device_switch" "test" {
  name                  = "Static IP Settings"
  mac                   = %[1]q
  management_network_id = %[2]q

  static_ip_settings = {
    ip            = %[3]q
    gateway       = "10.2.0.1"
    netmask       = "255.255.255.0"
    preferred_dns = "1.2.3.4"
  }
}
`, macAddress, managementNetworkID, ip)
}

func TestAccDeviceSwitchResource_PortOverrides(t *testing.T) {
	// TODO: (jtoyer) Actually implement the test
	ctx := context.Background()
	device, releaseDevice := getSwitchDevice(ctx, t)
	defer releaseDevice()

	network := getNetwork(ctx, t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDeviceConfigPortOverridesCreate(*device.MAC, *network.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device_switch.test", "port_overrides.%", "9"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.1.name`, "Disabled"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.1.disabled`, "true"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.2.name`, "Block All Tagged VLAN"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.2.tagged_vlan_management`, "block_all"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.3.full_duplex`, "true"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.3.link_speed`, "1000"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.3.name`, "Link Speed"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.3.operation`, "switch"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.4.poe_mode`, "off"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.4.name`, "POE Off"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.5.name`, "Disabled native network"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.5.native_network_id`, ""),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.6.aggregate_num_ports`, "2"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.6.operation`, "aggregate"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.6.name`, "Aggregate 1"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.7.name`, "Aggregate 2"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.8.mirror_port_index`, "9"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.8.name`, "Mirror Root"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.8.operation`, "mirror"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.9.name`, "Mirror Target"),
				),
			},
			// Update and Read testing
			{
				Config: testAccDeviceConfigPortOverridesUpdate(*device.MAC, *network.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device_switch.test", "port_overrides.%", "2"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.3.full_duplex`, "false"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.3.link_speed`, "100"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.3.name`, "Link Speed"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.3.operation`, "switch"),

					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.4.poe_mode`, "pasv24"),
					resource.TestCheckResourceAttr("unifi_device_switch.test", `port_overrides.4.name`, "Updated POE"),
				),
			},
		},
	})
}

func testAccDeviceConfigPortOverridesCreate(macAddress, managementNetworkID string) string {
	// TODO: (jtoyer) Add support for tagged network tests
	// TODO: (jtoyer) Add support for native network override tests
	// TODO: (jtoyer) Add support for port profile tests
	return fmt.Sprintf(`
provider "unifi" {}
resource "unifi_device_switch" "test" {
  name                  = "Port Overrides"
  mac                   = %[1]q
  management_network_id = %[2]q

  port_overrides = {
    "1" = {
      name     = "Disabled"
      disabled = true
    }
    "2" = {
      name                   = "Block All Tagged VLAN"
      tagged_vlan_management = "block_all"
    }
    "3" = {
      full_duplex = true
      link_speed  = "1000"
      name        = "Link Speed"
      operation   = "switch"
    }
    "4" = {
      poe_mode = "off"
      name     = "POE Off"
    }
    "5" = {
      name              = "Disabled native network"
      native_network_id = ""
    }
    "6" = {
      aggregate_num_ports = 2
      operation           = "aggregate"
      name                = "Aggregate 1"
    }
    "7" = {
      name = "Aggregate 2"
    }
    "8" = {
      mirror_port_index = 9
      name              = "Mirror Root"
      operation         = "mirror"
    }
    "9" = {
      name = "Mirror Target"
    }
    // "10" = {
    //   excluded_tagged_network_ids = [
    //     "669c08b2329aae15c4b3d60a",
    //   ]
    //   name                   = "Custom Tagged VLAN"
    //   native_network_id      = "669c0336329aae15c4b318f2"
    //   operation              = "switch"
    //   poe_mode               = "auto"
    //   tagged_vlan_management = "custom"
    // },
    // "11" = {
    //   name              = "Native Network Override"
    //   native_network_id = "669c08b2329aae15c4b3d60a"
    // }
    // "12" = {
    //   port_profile_id = "669c1ef8329aae15c4b3f791"
    //   name            = "Port Profile"
    // }
  }
}
`, macAddress, managementNetworkID)
}

func testAccDeviceConfigPortOverridesUpdate(macAddress, managementNetworkID string) string {
	// TODO: (jtoyer) Add support for tagged network tests
	// TODO: (jtoyer) Add support for native network override tests
	// TODO: (jtoyer) Add support for port profile tests
	return fmt.Sprintf(`
provider "unifi" {}
resource "unifi_device_switch" "test" {
  name                  = "Port Overrides"
  mac                   = %[1]q
  management_network_id = %[2]q

  port_overrides = {
    "3" = {
      full_duplex = false
      link_speed  = "100"
      name        = "Link Speed"
      operation   = "switch"
    }
    "4" = {
      poe_mode = "pasv24"
      name     = "Updated POE"
    }
    // "10" = {
    //   excluded_tagged_network_ids = [
    //     "669c08b2329aae15c4b3d60a",
    //   ]
    //   name                   = "Custom Tagged VLAN"
    //   native_network_id      = "669c0336329aae15c4b318f2"
    //   operation              = "switch"
    //   poe_mode               = "auto"
    //   tagged_vlan_management = "custom"
    // },
    // "11" = {
    //   name              = "Native Network Override"
    //   native_network_id = "669c08b2329aae15c4b3d60a"
    // }
    // "12" = {
    //   port_profile_id = "669c1ef8329aae15c4b3f791"
    //   name            = "Port Profile"
    // }
  }
}
`, macAddress, managementNetworkID)
}
