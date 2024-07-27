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
				Config: testAccDeviceConfigPortOverrides(*device.MAC, *network.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device_switch.test", "name", "Test Switch"),
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
				Config: testAccDeviceConfigPortOverrides(*device.MAC, *network.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device_switch.test", "name", "Updated Test Switch"),
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

func testAccDeviceConfigPortOverrides(macAddress, managementNetworkID string) string {
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
    // "2" = {
    //   excluded_tagged_network_ids = [
    //     "669c08b2329aae15c4b3d60a",
    //   ]
    //   name                   = "Custom Tagged VLAN"
    //   native_network_id      = "669c0336329aae15c4b318f2"
    //   operation              = "switch"
    //   poe_mode               = "auto"
    //   tagged_vlan_management = "custom"
    // },
    "3" = {
      name                   = "Block All Tagged VLAN"
      tagged_vlan_management = "block_all"
    }
    // "4" = {
    //   name              = "Native Network Override"
    //   native_network_id = "669c08b2329aae15c4b3d60a"
    // }
    "5" = {
      full_duplex = true
      link_speed  = "1000"
      name        = "Link Speed"
      operation   = "switch"
    }
    "6" = {
      poe_mode = "off"
      name     = "POE Off"
    }
    // "7" = {
    //   port_profile_id = "669c1ef8329aae15c4b3f791"
    //   name            = "Port Profile"
    // }
    "8" = {
      name              = "Disabled native network"
      native_network_id = ""
    }
    "9" = {
      aggregate_num_ports = 2
      operation           = "aggregate"
      name                = "Aggregate 1"
    }
    "10" = {
      name = "Aggregate 2"
    }
    "11" = {
      name = "Mirror Target"
    }
    "12" = {
      mirror_port_index = 47
      name              = "Mirror Root"
      operation         = "mirror"
    }
  }
}
`, macAddress, managementNetworkID)
}

// func TestAccDeviceSwitchResource(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:                 func() { testAccPreCheck(t) },
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			// Create and Read testing
// 			{
// 				Config: testAccExampleResourceConfig("one"),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("unifi_example.test", "configurable_attribute", "one"),
// 					resource.TestCheckResourceAttr("unifi_example.test", "defaulted", "example value when not configured"),
// 					resource.TestCheckResourceAttr("unifi_example.test", "id", "example-id"),
// 				),
// 			},
// 			// ImportState testing
// 			{
// 				ResourceName:      "unifi_example.test",
// 				ImportState:       true,
// 				ImportStateVerify: true,
// 				// This is not normally necessary, but is here because this
// 				// example code does not have an actual upstream service.
// 				// Once the Read method is able to refresh information from
// 				// the upstream service, this can be removed.
// 				ImportStateVerifyIgnore: []string{"configurable_attribute", "defaulted"},
// 			},
// 			// Update and Read testing
// 			{
// 				Config: testAccExampleResourceConfig("two"),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("unifi_example.test", "configurable_attribute", "two"),
// 				),
// 			},
// 			// Delete testing automatically occurs in TestCase
// 		},
// 	})
// }

func testAccDeviceSwitchResourceConfig(macAddress, managementNetworkID string) string {
	return fmt.Sprintf(`
provider "unifi" {}

resource "unifi_device_switch" "example" {
  name = "Example Switch Update"
  mac  = %[1]q

  management_network_id = %[2]q
}


`, macAddress, managementNetworkID)
}
