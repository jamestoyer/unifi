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

func TestAccDeviceDataSource(t *testing.T) {
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
				Config: testAccDeviceDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_device.test", "mac", "dc:9f:db:00:00:01"),
					resource.TestCheckResourceAttr("data.unifi_device.test", "type", "ugw"),
					resource.TestCheckResourceAttr("data.unifi_device.test", "type", "ugw"),
				),
			},
		},
	})
}

const testAccDeviceDataSourceConfig = `
provider "unifi" {}
data "unifi_device" "test" {
  mac = "dc:9f:db:00:00:01"
}
`
