package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/jamestoyer/go-unifi/unifi"
	"sync"
	"testing"
	"time"
)

var (
	devicesReady   = sync.Once{}
	switchPool     []*unifi.Device
	switchPoolLock sync.Mutex
)

func cacheDeviceDetails(ctx context.Context, t *testing.T) {
	t.Helper()

	devicesReady.Do(func() {
		err := retry.RetryContext(ctx, 1*time.Minute, func() *retry.RetryError {
			devices, err := testClient.ListDevice(ctx, testClient.site)
			if err != nil {
				return retry.NonRetryableError(fmt.Errorf("listing devices failed: %w", err))
			}

			if len(devices) == 0 {
				return retry.RetryableError(fmt.Errorf("no devices found"))
			}

			switchPoolLock.Lock()
			defer switchPoolLock.Unlock()
			for _, device := range devices {
				switch *device.Type {
				case "usw":
					switchPool = append(switchPool, &device)
				}
			}
			return nil
		})

		if err != nil {
			t.Fatal(err)
		}
	})
}

func getSwitchDevice(ctx context.Context, t *testing.T) (*unifi.Device, func()) {
	t.Helper()

	// Devices take a little bit of time to load so retry until we have devices
	cacheDeviceDetails(ctx, t)

	var device *unifi.Device

	switchPoolLock.Lock()
	defer switchPoolLock.Unlock()

	device = switchPool[0]
	switchPool = switchPool[1:]

	release := func() {
		switchPoolLock.Lock()
		defer switchPoolLock.Unlock()
		switchPool = append(switchPool, device)
	}

	return device, release
}

func getNetwork(ctx context.Context, t *testing.T) *unifi.Network {
	t.Helper()

	var network unifi.Network
	err := retry.RetryContext(ctx, 1*time.Minute, func() *retry.RetryError {
		networks, err := testClient.ListNetwork(ctx, testClient.site)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("listing networks failed: %w", err))
		}

		if len(networks) == 0 {
			return retry.RetryableError(fmt.Errorf("no networks found"))
		}

		network = networks[0]
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return &network
}
