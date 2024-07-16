// Copyright (c) James Toyer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
	"time"
)

func TestAccDeviceSwitchDataSource(t *testing.T) {
	var switchMac string
	// Devices take a little bit of time to load so retry until we have devices
	ctx := context.Background()
	err := retry.RetryContext(ctx, 1*time.Minute, func() *retry.RetryError {
		devices, err := testClient.ListDevice(ctx, testClient.site)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("listing devices failed: %w", err))
		}

		if len(devices) == 0 {
			return retry.RetryableError(fmt.Errorf("no devices found"))
		}

		for _, device := range devices {
			if device.Type == "usw" {
				switchMac = device.MAC
				break
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceSwitchDataSourceConfig(switchMac),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_device_switch.test", "mac", switchMac),
					resource.TestCheckResourceAttr("data.unifi_device_switch.test", "type", "usw"),
				),
			},
		},
	})
}

func testAccDeviceSwitchDataSourceConfig(mac string) string {
	return `
provider "unifi" {}
data "unifi_device_switch" "test" {
  mac = "` + mac + `"
}
`
}
