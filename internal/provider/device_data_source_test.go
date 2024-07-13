// Copyright (c) James Toyer
// SPDX-License-Identifier: MPL-2.0

package provider

// TODO: (jtoyer) Implement tests once tests start adopting devices
// func TestAccDeviceDataSource(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		PreCheck:                 func() { testAccPreCheck(t) },
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps:                    []resource.TestStep{
// 			{
// 				Config: testAccDeviceDataSourceConfig,
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("data.unifi_device.test", "id", "example-id"),
// 				),
// 			},
// 		},
// 	})
// }
//
// const testAccDeviceDataSourceConfig = `
// provider "unifi" {}
// data "unifi_device" "test" {
//   mac = "dc:9f:db:00:00:01"
// }
// `
